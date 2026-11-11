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

func main() {
	rusLap, err := loadLap("data/RUS_lap.json")
	if err != nil {
		log.Fatalf("Failed to load RUS lap: %v", err)
	}

	hamLap, err := loadLap("data/HAM_lap.json")
	if err != nil {
		log.Fatalf("Failed to load HAM lap: %v", err)
	}

	rusArc := telemetry.ComputeArcLengthDistance(rusLap.Samples)
	hamArc := telemetry.ComputeArcLengthDistance(hamLap.Samples)

	gridStep := 1.0
	rusGrid := telemetry.ResampleLinear(rusLap.Samples, rusArc, gridStep)
	hamGrid := telemetry.ResampleLinear(hamLap.Samples, hamArc, gridStep)

	officialDelta := hamLap.OfficialLapTime - rusLap.OfficialLapTime // +0.171s
	deltas, finishDelta, absErr := telemetry.ComputeDistanceDelta(rusGrid, hamGrid, officialDelta)

	fmt.Println("=== STAGE 4 DELTA & GROUND TRUTH VALIDATION ===")
	fmt.Printf("Official Ground Truth Gap : +%.4f s (RUS %.3fs vs HAM %.3fs)\n", officialDelta, rusLap.OfficialLapTime, hamLap.OfficialLapTime)
	fmt.Printf("Computed Finish Line Delta: +%.4f s\n", finishDelta)
	fmt.Printf("Absolute Delta Error      : %.4f s (%.2f ms)\n", absErr, absErr*1000)

	tolerance := 0.050 // 50ms tolerance
	if absErr <= tolerance {
		fmt.Printf("Status                     : VERIFIED (Within %.0fms tolerance)\n", tolerance*1000)
	} else {
		fmt.Printf("Status                     : WARNING (Exceeds %.0fms tolerance, investigation needed)\n", tolerance*1000)
	}
}
