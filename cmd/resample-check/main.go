package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"ghost-lap/pkg/telemetry"
)

func loadTrimmed(path string) ([]telemetry.TelemetrySample, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lap telemetry.LapExport
	if err := json.Unmarshal(data, &lap); err != nil {
		return nil, err
	}
	return telemetry.TrimToOfficialLapWindow(lap.Samples, lap.OfficialLapTime), nil
}

func main() {
	rusSamples, err := loadTrimmed("data/RUS_lap.json")
	if err != nil {
		log.Fatal(err)
	}
	hamSamples, err := loadTrimmed("data/HAM_lap.json")
	if err != nil {
		log.Fatal(err)
	}

	rusGrid := telemetry.ResampleLinear(rusSamples, telemetry.ComputeArcLengthDistance(rusSamples), 1.0)
	hamGrid := telemetry.ResampleLinear(hamSamples, telemetry.ComputeArcLengthDistance(hamSamples), 1.0)

	throttleOvershoots, brakeOvershoots := 0, 0
	for _, g := range append(rusGrid, hamGrid...) {
		if g.Throttle < 0 || g.Throttle > 100 {
			throttleOvershoots++
		}
		if g.Brake < 0 || g.Brake > 1 {
			brakeOvershoots++
		}
	}

	fmt.Println("=== STAGE 3 COMMON DISTANCE GRID RESAMPLING ===")
	fmt.Printf("Grid Step Size               : 1.0 m\n")
	fmt.Printf("RUS Resampled Grid Points    : %d points (Max Distance: %.1f m)\n", len(rusGrid), rusGrid[len(rusGrid)-1].Distance)
	fmt.Printf("HAM Resampled Grid Points    : %d points (Max Distance: %.1f m)\n", len(hamGrid), hamGrid[len(hamGrid)-1].Distance)
	fmt.Printf("Physical Boundary Overshoots : Throttle=%d, Brake=%d (Valid: %v)\n",
		throttleOvershoots, brakeOvershoots, throttleOvershoots == 0 && brakeOvershoots == 0)
}
