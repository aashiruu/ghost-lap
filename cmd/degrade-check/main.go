package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"

	"ghost-lap/pkg/telemetry"
)

func loadLap(path string) (*telemetry.LapExport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lap telemetry.LapExport
	if err := json.Unmarshal(data, &lap); err != nil {
		return nil, err
	}
	return &lap, nil
}

// integrateTimeFromSpeed computes elapsed time along a 1m grid from speed (km/h),
// calibrated to the driver's official lap time at the finish line.
func integrateTimeFromSpeed(grid []telemetry.ResampledPoint, gridStep float64, officialLapTime float64) []float64 {
	times := make([]float64, len(grid))
	if len(grid) == 0 {
		return times
	}
	var rawTotal float64
	for i := 1; i < len(grid); i++ {
		vMS := ((grid[i-1].Speed + grid[i].Speed) / 2.0) / 3.6
		if vMS < 5.0 {
			vMS = 5.0 // floor to prevent division by zero in pits/stationary ticks
		}
		dt := gridStep / vMS
		rawTotal += dt
		times[i] = rawTotal
	}
	if rawTotal > 0 {
		scale := officialLapTime / rawTotal
		for i := range times {
			times[i] *= scale
		}
	}
	return times
}

func evaluateDegradation(rusLap, hamLap *telemetry.LapExport, dropRate float64, seed int64, officialDelta float64) (float64, float64, float64) {
	gridStep := 1.0

	// Baseline (0% drop) speed-integrated time curve
	rusBaseArc := telemetry.ComputeArcLengthDistance(rusLap.Samples)
	hamBaseArc := telemetry.ComputeArcLengthDistance(hamLap.Samples)
	rusBaseGrid := telemetry.ResampleLinear(rusLap.Samples, rusBaseArc, gridStep)
	hamBaseGrid := telemetry.ResampleLinear(hamLap.Samples, hamBaseArc, gridStep)
	rusBaseTimes := integrateTimeFromSpeed(rusBaseGrid, gridStep, rusLap.OfficialLapTime)
	hamBaseTimes := integrateTimeFromSpeed(hamBaseGrid, gridStep, hamLap.OfficialLapTime)

	// Degraded sample streams
	rusDeg := telemetry.FilterSamplesWithLoss(rusLap.Samples, dropRate, seed)
	hamDeg := telemetry.FilterSamplesWithLoss(hamLap.Samples, dropRate, seed+1)

	rusArc := telemetry.ComputeArcLengthDistance(rusDeg)
	hamArc := telemetry.ComputeArcLengthDistance(hamDeg)
	rusGrid := telemetry.ResampleLinear(rusDeg, rusArc, gridStep)
	hamGrid := telemetry.ResampleLinear(hamDeg, hamArc, gridStep)

	// Integrate uncalibrated time loss caused by missing speed samples relative to baseline scale
	rusDegTimes := integrateTimeFromSpeed(rusGrid, gridStep, rusLap.OfficialLapTime)
	hamDegTimes := integrateTimeFromSpeed(hamGrid, gridStep, hamLap.OfficialLapTime)

	minLen := len(rusBaseTimes)
	for _, l := range []int{len(hamBaseTimes), len(rusDegTimes), len(hamDegTimes)} {
		if l < minLen {
			minLen = l
		}
	}

	var sumAbsErr, maxErr float64
	for i := 0; i < minLen; i++ {
		baseDelta := hamBaseTimes[i] - rusBaseTimes[i]
		degDelta := hamDegTimes[i] - rusDegTimes[i]
		err := math.Abs(degDelta - baseDelta)
		sumAbsErr += err
		if err > maxErr {
			maxErr = err
		}
	}

	mae := sumAbsErr / float64(minLen)
	return officialDelta, maxErr, mae
}

func main() {
	rusLap, err := loadLap("data/RUS_lap.json")
	if err != nil {
		log.Fatalf("Failed to load RUS lap: %v", err)
	}
	hamLap, err := loadLap("data/HAM_lap.json")
	if err != nil {
		log.Fatalf("Failed to load HAM lap: %v", err)
	}

	officialDelta := hamLap.OfficialLapTime - rusLap.OfficialLapTime
	seed := int64(42)
	dropRates := []float64{0.0, 0.05, 0.10, 0.25}

	fmt.Println("=== STAGE 6 SAMPLE-LOSS DEGRADATION TEST ===")
	fmt.Printf("%-12s | %-18s | %-18s\n", "Loss Rate", "Max Delta Err (ms)", "MAE vs Base (ms)")
	fmt.Println("---------------------------------------------------------")

	for _, rate := range dropRates {
		_, maxErr, mae := evaluateDegradation(rusLap, hamLap, rate, seed, officialDelta)
		fmt.Printf("%-12s | %-18.2f | %-18.2f\n",
			fmt.Sprintf("%.0f%%", rate*100),
			maxErr*1000,
			mae*1000,
		)
	}
}
