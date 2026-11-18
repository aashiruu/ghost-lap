package main

import (
	"encoding/json"
	"fmt"
	"log"
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

func integrateTimeFromSpeed(grid []telemetry.ResampledPoint, gridStep float64, officialLapTime float64) []float64 {
	times := make([]float64, len(grid))
	if len(grid) == 0 {
		return times
	}
	var rawTotal float64
	for i := 1; i < len(grid); i++ {
		vMS := ((grid[i-1].Speed + grid[i].Speed) / 2.0) / 3.6
		if vMS < 5.0 {
			vMS = 5.0
		}
		rawTotal += gridStep / vMS
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

func main() {
	rusLap, err := loadLap("data/RUS_lap.json")
	if err != nil {
		log.Fatalf("Failed to load RUS lap: %v", err)
	}
	hamLap, err := loadLap("data/HAM_lap.json")
	if err != nil {
		log.Fatalf("Failed to load HAM lap: %v", err)
	}

	gridStep := 1.0
	rusArc := telemetry.ComputeArcLengthDistance(rusLap.Samples)
	hamArc := telemetry.ComputeArcLengthDistance(hamLap.Samples)
	rusGrid := telemetry.ResampleLinear(rusLap.Samples, rusArc, gridStep)
	hamGrid := telemetry.ResampleLinear(hamLap.Samples, hamArc, gridStep)

	rusTimes := integrateTimeFromSpeed(rusGrid, gridStep, rusLap.OfficialLapTime)
	hamTimes := integrateTimeFromSpeed(hamGrid, gridStep, hamLap.OfficialLapTime)

	maxDist := rusGrid[len(rusGrid)-1].Distance
	segments := telemetry.SilverstoneSegments(maxDist)
	results, sumDiff, conservationErr := telemetry.ComputeSegmentDeltas(rusGrid, hamGrid, rusTimes, hamTimes, segments)

	fmt.Println("=== STAGE 7 PER-SEGMENT GAP BREAKDOWN (2024 SILVERSTONE Q3) ===")
	fmt.Printf("%-50s | %-8s | %-8s | %-10s | %-6s\n", "Segment", "RUS (s)", "HAM (s)", "Delta (s)", "Ahead")
	fmt.Println("-----------------------------------------------------------------------------------------------")
	for _, r := range results {
		fmt.Printf("%-50s | %-8.3f | %-8.3f | %+10.4f | %-6s\n",
			r.Name, r.TimeRUS, r.TimeHAM, r.SegmentDiff, r.Winner)
	}
	fmt.Println("-----------------------------------------------------------------------------------------------")
	fmt.Printf("Sum of Segment Deltas : %+.4f s\n", sumDiff)
	fmt.Printf("Total Lap Delta       : %+.4f s\n", hamLap.OfficialLapTime-rusLap.OfficialLapTime)
	fmt.Printf("Conservation Error    : %.6f ms\n", conservationErr*1000)
}
