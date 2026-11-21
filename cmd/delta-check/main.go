package main

import (
	"encoding/json"
	"fmt"
	"log"
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

	rusGrid := telemetry.ResampleLinear(rusSamples, telemetry.ComputeArcLengthDistance(rusSamples), 1.0)
	hamGrid := telemetry.ResampleLinear(hamSamples, telemetry.ComputeArcLengthDistance(hamSamples), 1.0)

	officialDelta := hamLap.OfficialLapTime - rusLap.OfficialLapTime
	deltas, finishDelta, absErr := telemetry.ComputeDistanceDelta(rusGrid, hamGrid, officialDelta)

	fmt.Println("=== STAGE 4 UNCALIBRATED DELTA & GROUND TRUTH VALIDATION ===")
	fmt.Printf("Overlapping Grid Samples   : %d points (0 to %.0f m)\n", len(deltas), deltas[len(deltas)-1].Distance)
	fmt.Printf("Official Ground Truth Gap  : %+.4f s (RUS %.3fs vs HAM %.3fs)\n", officialDelta, rusLap.OfficialLapTime, hamLap.OfficialLapTime)
	fmt.Printf("Computed Finish Line Delta : %+.4f s (Uncalibrated)\n", finishDelta)
	fmt.Printf("Absolute Delta Error       : %.4f s (%.2f ms)\n", absErr, absErr*1000)
	if absErr <= 0.050 {
		fmt.Println("Status                     : VERIFIED (Within 50ms tolerance)")
	} else {
		fmt.Println("Status                     : WARNING (Exceeds 50ms tolerance)")
	}
}
