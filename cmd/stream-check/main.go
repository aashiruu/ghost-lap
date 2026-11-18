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

	// 1. Stage 4 Batch Reference Calculation
	rusArcBatch := telemetry.ComputeArcLengthDistance(rusLap.Samples)
	hamArcBatch := telemetry.ComputeArcLengthDistance(hamLap.Samples)

	gridStep := 1.0
	rusGridBatch := telemetry.ResampleLinear(rusLap.Samples, rusArcBatch, gridStep)
	hamGridBatch := telemetry.ResampleLinear(hamLap.Samples, hamArcBatch, gridStep)
	officialDelta := hamLap.OfficialLapTime - rusLap.OfficialLapTime
	_, finishDeltaBatch, _ := telemetry.ComputeDistanceDelta(rusGridBatch, hamGridBatch, officialDelta)

	// 2. Stage 5 Streaming Incremental Ingestion
	rusStream := telemetry.NewDriverStreamState("RUS")
	hamStream := telemetry.NewDriverStreamState("HAM")

	maxSamples := len(rusLap.Samples)
	if len(hamLap.Samples) < maxSamples {
		maxSamples = len(hamLap.Samples)
	}

	var rusStreamGrid, hamStreamGrid []telemetry.ResampledPoint

	for i := 0; i < maxSamples; i++ {
		rusStream.PushSample(rusLap.Samples[i])
		hamStream.PushSample(hamLap.Samples[i])

		ptsRUS := rusStream.ComputeIncrementalPoints(gridStep)
		ptsHAM := hamStream.ComputeIncrementalPoints(gridStep)

		rusStreamGrid = append(rusStreamGrid, ptsRUS...)
		hamStreamGrid = append(hamStreamGrid, ptsHAM...)
	}

	_, finishDeltaStream, _ := telemetry.ComputeDistanceDelta(rusStreamGrid, hamStreamGrid, officialDelta)

	diff := math.Abs(finishDeltaStream - finishDeltaBatch)

	fmt.Println("=== STAGE 5 STREAMING CONSUMER EQUIVALENCE VALIDATION ===")
	fmt.Printf("Stage 4 Batch Finish Line Delta     : +%.4f s\n", finishDeltaBatch)
	fmt.Printf("Stage 5 Incremental Stream Delta   : +%.4f s\n", finishDeltaStream)
	fmt.Printf("Stream vs Batch Difference          : %.6f s (%.4f ms)\n", diff, diff*1000)

	if diff < 1e-5 {
		fmt.Println("Status                             : VERIFIED (Stream results match batch output exactly)")
	} else {
		fmt.Println("Status                             : WARNING (Mismatch between streaming and batch pipeline)")
	}
}
