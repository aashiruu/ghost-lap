package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"

	"ghost-lap/pkg/telemetry"
)

func loadLap(path string) (*telemetry.LapExport, []telemetry.TelemetrySample, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var lap telemetry.LapExport
	if err := json.Unmarshal(data, &lap); err != nil {
		return nil, nil, err
	}
	return &lap, telemetry.TrimToOfficialLapWindow(lap.Samples, lap.OfficialLapTime), nil
}

func main() {
	rusLap, rusSamples, err := loadLap("data/RUS_lap.json")
	if err != nil {
		log.Fatal(err)
	}
	hamLap, hamSamples, err := loadLap("data/HAM_lap.json")
	if err != nil {
		log.Fatal(err)
	}

	officialDelta := hamLap.OfficialLapTime - rusLap.OfficialLapTime
	rusBase := telemetry.ResampleLinear(rusSamples, telemetry.ComputeArcLengthDistance(rusSamples), 1.0)
	hamBase := telemetry.ResampleLinear(hamSamples, telemetry.ComputeArcLengthDistance(hamSamples), 1.0)
	baseDeltas, _, _ := telemetry.ComputeDistanceDelta(rusBase, hamBase, officialDelta)

	seed := int64(42)
	dropRates := []float64{0.0, 0.05, 0.10, 0.25}

	fmt.Println("=== STAGE 6 SAMPLE-LOSS DEGRADATION TEST (PRNG Seed = 42, Endpoints Anchored) ===")
	fmt.Printf("%-10s | %-16s | %-18s | %-18s | %-18s\n",
		"Loss Rate", "Finish Delta (s)", "Err vs Official (ms)", "Max Curve Err (ms)", "MAE vs Base (ms)")
	fmt.Println("-------------------------------------------------------------------------------------------------")

	for _, rate := range dropRates {
		rusDeg := telemetry.FilterSamplesWithLoss(rusSamples, rate, seed)
		hamDeg := telemetry.FilterSamplesWithLoss(hamSamples, rate, seed+1)

		rusGrid := telemetry.ResampleLinear(rusDeg, telemetry.ComputeArcLengthDistance(rusDeg), 1.0)
		hamGrid := telemetry.ResampleLinear(hamDeg, telemetry.ComputeArcLengthDistance(hamDeg), 1.0)
		deltas, finishDelta, officialErr := telemetry.ComputeDistanceDelta(rusGrid, hamGrid, officialDelta)

		minLen := len(deltas)
		if len(baseDeltas) < minLen {
			minLen = len(baseDeltas)
		}

		var sumAbs, maxErr float64
		for i := 0; i < minLen; i++ {
			e := math.Abs(deltas[i].DeltaSec - baseDeltas[i].DeltaSec)
			sumAbs += e
			if e > maxErr {
				maxErr = e
			}
		}
		mae := sumAbs / float64(minLen)

		fmt.Printf("%-10s | %+-16.4f | %-18.2f | %-18.2f | %-18.2f\n",
			fmt.Sprintf("%.0f%%", rate*100),
			finishDelta,
			officialErr*1000,
			maxErr*1000,
			mae*1000,
		)
	}
}
