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

	maxDist := rusGrid[len(rusGrid)-1].Distance
	if hamGrid[len(hamGrid)-1].Distance < maxDist {
		maxDist = hamGrid[len(hamGrid)-1].Distance
	}

	results, sumDiff, conservationErr := telemetry.ComputeSegmentDeltas(rusGrid, hamGrid, telemetry.SilverstoneSegments(maxDist))

	fmt.Println("=== STAGE 7 PER-SEGMENT GAP BREAKDOWN (2024 SILVERSTONE Q3) ===")
	fmt.Printf("%-48s | %-6s | %-7s | %-7s | %-9s | %-9s | %-9s | %-5s\n",
		"Segment", "Len(m)", "RUS(s)", "HAM(s)", "RUS(m/s)", "HAM(m/s)", "Delta(s)", "Ahead")
	fmt.Println("----------------------------------------------------------------------------------------------------------------------")

	realismValid := true
	for _, r := range results {
		fmt.Printf("%-48s | %-6.0f | %-7.3f | %-7.3f | %-9.2f | %-9.2f | %+9.4f | %-5s\n",
			r.Name, r.LengthM, r.TimeRUS, r.TimeHAM, r.AvgSpeedRUS, r.AvgSpeedHAM, r.SegmentDiff, r.Winner)
		if r.LengthM > 600.0 {
			if r.AvgSpeedRUS < 35.0 || r.AvgSpeedRUS > 90.0 || r.AvgSpeedHAM < 35.0 || r.AvgSpeedHAM > 90.0 {
				realismValid = false
			}
		}
	}

	fmt.Println("----------------------------------------------------------------------------------------------------------------------")
	fmt.Printf("Sum of Segment Deltas : %+.4f s (Official Gap: %+.4f s)\n", sumDiff, hamLap.OfficialLapTime-rusLap.OfficialLapTime)
	fmt.Printf("Conservation Error    : %.6f ms\n", conservationErr*1000)
	if !realismValid {
		log.Fatalf("FAIL: Segment > 600m violated physical speed bounds [35, 90] m/s")
	}
	fmt.Println("Physical Realism Check: VERIFIED (All segments > 600m within 35-90 m/s)")
}
