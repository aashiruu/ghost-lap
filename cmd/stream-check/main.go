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

rusBatch := telemetry.ResampleLinear(rusSamples, telemetry.ComputeArcLengthDistance(rusSamples), 1.0)
hamBatch := telemetry.ResampleLinear(hamSamples, telemetry.ComputeArcLengthDistance(hamSamples), 1.0)
_, batchDelta, _ := telemetry.ComputeDistanceDelta(rusBatch, hamBatch, officialDelta)

rusStream := telemetry.NewDriverStreamState("RUS")
hamStream := telemetry.NewDriverStreamState("HAM")
var rusStreamPts, hamStreamPts []telemetry.ResampledPoint

for _, s := range rusSamples {
rusStream.PushSample(s)
rusStreamPts = append(rusStreamPts, rusStream.ComputeIncrementalPoints(1.0)...)
}
for _, s := range hamSamples {
hamStream.PushSample(s)
hamStreamPts = append(hamStreamPts, hamStream.ComputeIncrementalPoints(1.0)...)
}
_, streamDelta, _ := telemetry.ComputeDistanceDelta(rusStreamPts, hamStreamPts, officialDelta)

diff := math.Abs(batchDelta - streamDelta)
fmt.Println("=== STAGE 5 STREAMING CONSUMER EQUIVALENCE VALIDATION ===")
fmt.Printf("Stage 4 Batch Finish Line Delta     : %+.4f s\n", batchDelta)
fmt.Printf("Stage 5 Incremental Stream Delta    : %+.4f s\n", streamDelta)
fmt.Printf("Stream vs Batch Difference          : %.6f s (%.4f ms)\n", diff, diff*1000)
if diff <= 1e-6 {
fmt.Println("Status                              : VERIFIED (Stream results match batch output exactly)")
}
}
