package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"

	"ghost-lap/pkg/telemetry"
)

func loadTrimmed(path string) (*telemetry.LapExport, []telemetry.TelemetrySample, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var lap telemetry.LapExport
	if err := json.Unmarshal(data, &lap); err != nil {
		return nil, nil, err
	}
	trimmed := telemetry.TrimToOfficialLapWindow(lap.Samples, lap.OfficialLapTime)
	return &lap, trimmed, nil
}

func main() {
	_, rusSamples, err := loadTrimmed("data/RUS_lap.json")
	if err != nil {
		log.Fatal(err)
	}
	_, hamSamples, err := loadTrimmed("data/HAM_lap.json")
	if err != nil {
		log.Fatal(err)
	}

	rusInt := telemetry.ComputeSpeedIntegratedDistance(rusSamples)
	hamInt := telemetry.ComputeSpeedIntegratedDistance(hamSamples)
	intTotalRUS := rusInt[len(rusInt)-1]
	intTotalHAM := hamInt[len(hamInt)-1]

	rusArc := telemetry.ComputeArcLengthDistance(rusSamples)
	hamArc := telemetry.ComputeArcLengthDistance(hamSamples)
	arcTotalRUS := rusArc[len(rusArc)-1]
	arcTotalHAM := hamArc[len(hamArc)-1]

	fmt.Println("=== STAGE 2 DISTANCE MEASUREMENT RESULTS ===")
	fmt.Println("Speed Integration (∫ v dt, km/h -> m/s):")
	fmt.Printf("- RUS Computed Total Distance : %.3f m\n", intTotalRUS)
	fmt.Printf("- HAM Computed Total Distance : %.3f m\n", intTotalHAM)
	fmt.Printf("- Inter-Driver Distance Drift : %.3f m\n\n", math.Abs(intTotalRUS-intTotalHAM))

	fmt.Println("2D Arc-Length Projection (X,Y decimeters -> m):")
	fmt.Printf("- RUS Computed Total Distance : %.3f m\n", arcTotalRUS)
	fmt.Printf("- HAM Computed Total Distance : %.3f m\n", arcTotalHAM)
	fmt.Printf("- Inter-Driver Path Difference: %.3f m\n", math.Abs(arcTotalRUS-arcTotalHAM))
}
