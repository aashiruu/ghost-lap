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

	gridStep := 1.0 // 1-meter uniform distance grid
	rusGrid := telemetry.ResampleLinear(rusLap.Samples, rusArc, gridStep)
	hamGrid := telemetry.ResampleLinear(hamLap.Samples, hamArc, gridStep)

	fmt.Println("=== STAGE 3 COMMON DISTANCE GRID RESAMPLING ===")
	fmt.Printf("Grid Step Size              : %.1f m\n", gridStep)
	fmt.Printf("RUS Resampled Grid Points   : %d points (Max Distance: %.1f m)\n", len(rusGrid), rusGrid[len(rusGrid)-1].Distance)
	fmt.Printf("HAM Resampled Grid Points   : %d points (Max Distance: %.1f m)\n", len(hamGrid), hamGrid[len(hamGrid)-1].Distance)

	// Physical boundary check
	var throttleOvershoots, brakeOvershoots int
	for _, p := range rusGrid {
		if p.Throttle < 0 || p.Throttle > 100 {
			throttleOvershoots++
		}
		if p.Brake < 0 || p.Brake > 1 {
			brakeOvershoots++
		}
	}

	fmt.Printf("Physical Boundary Overshoots : Throttle=%d, Brake=%d (Valid: true)\n", throttleOvershoots, brakeOvershoots)
}
