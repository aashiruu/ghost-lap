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

func main() {
	rusLap, err := loadLap("data/RUS_lap.json")
	if err != nil {
		log.Fatalf("Failed to load RUS lap: %v", err)
	}

	hamLap, err := loadLap("data/HAM_lap.json")
	if err != nil {
		log.Fatalf("Failed to load HAM lap: %v", err)
	}

	// 1. Integration Method
	rusIntegDist := telemetry.ComputeIntegratedDistance(rusLap.Samples)
	hamIntegDist := telemetry.ComputeIntegratedDistance(hamLap.Samples)

	rusTotalInteg := rusIntegDist[len(rusIntegDist)-1]
	hamTotalInteg := hamIntegDist[len(hamIntegDist)-1]
	integDiff := math.Abs(rusTotalInteg - hamTotalInteg)

	// 2. Projection Method (using RUS lap as baseline track reference)
	rusProjDist := telemetry.ComputeProjectedDistance(rusLap.Samples, rusLap.Samples)
	hamProjDist := telemetry.ComputeProjectedDistance(hamLap.Samples, rusLap.Samples)

	rusTotalProj := rusProjDist[len(rusProjDist)-1]
	hamTotalProj := hamProjDist[len(hamProjDist)-1]
	projDiff := math.Abs(rusTotalProj - hamTotalProj)

	fmt.Println("=== STAGE 2 DISTANCE MEASUREMENT RESULTS ===")
	fmt.Printf("Speed Integration (∫ v dt):\n")
	fmt.Printf("  - RUS Computed Total Distance : %.3f m\n", rusTotalInteg)
	fmt.Printf("  - HAM Computed Total Distance : %.3f m\n", hamTotalInteg)
	fmt.Printf("  - Inter-Driver Distance Drift  : %.3f m\n\n", integDiff)

	fmt.Printf("Spatial Projection (X,Y -> S):\n")
	fmt.Printf("  - RUS Computed Total Distance : %.3f m\n", rusTotalProj)
	fmt.Printf("  - HAM Computed Total Distance : %.3f m\n", hamTotalProj)
	fmt.Printf("  - Inter-Driver Distance Drift  : %.3f m\n", projDiff)
}
