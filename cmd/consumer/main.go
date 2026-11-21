package main

import (
"context"
"encoding/json"
"flag"
"fmt"
"log"
"math"
"os"
"time"

"github.com/twmb/franz-go/pkg/kgo"
"ghost-lap/pkg/telemetry"
)

type WireRecord struct {
DriverCode string                    `json:"driver_code"`
Sample     telemetry.TelemetrySample `json:"sample"`
}

func main() {
brokers := flag.String("brokers", "localhost:9092", "Redpanda broker addresses")
topic := flag.String("topic", "telemetry-raw", "Kafka/Redpanda topic")
expected := flag.Int("expected", 662, "Total expected records across both drivers")
flag.Parse()

ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
defer cancel()

cl, err := kgo.NewClient(
kgo.SeedBrokers(*brokers),
kgo.ConsumeTopics(*topic),
kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
)
if err != nil {
log.Fatalf("failed to create Redpanda consumer: %v", err)
}
defer cl.Close()

states := map[string]*telemetry.DriverStreamState{
"RUS": telemetry.NewDriverStreamState("RUS"),
"HAM": telemetry.NewDriverStreamState("HAM"),
}
grids := map[string][]telemetry.ResampledPoint{
"RUS": {},
"HAM": {},
}

received := 0
for received < *expected {
fetches := cl.PollFetches(ctx)
if errs := fetches.Errors(); len(errs) > 0 {
log.Fatalf("fetch error: %v", errs[0].Err)
}
fetches.EachRecord(func(r *kgo.Record) {
var msg WireRecord
if err := json.Unmarshal(r.Value, &msg); err != nil {
log.Printf("skipping malformed record: %v", err)
return
}
st, ok := states[msg.DriverCode]
if !ok {
return
}
st.PushSample(msg.Sample)
pts := st.ComputeIncrementalPoints(1.0)
if len(pts) > 0 {
grids[msg.DriverCode] = append(grids[msg.DriverCode], pts...)
}
received++
})
}

rusData, _ := os.ReadFile("data/RUS_lap.json")
hamData, _ := os.ReadFile("data/HAM_lap.json")
var rusLap, hamLap telemetry.LapExport
_ = json.Unmarshal(rusData, &rusLap)
_ = json.Unmarshal(hamData, &hamLap)

officialDelta := hamLap.OfficialLapTime - rusLap.OfficialLapTime
_, streamFinishDelta, _ := telemetry.ComputeDistanceDelta(grids["RUS"], grids["HAM"], officialDelta)

rusTrim := telemetry.TrimToOfficialLapWindow(rusLap.Samples, rusLap.OfficialLapTime)
hamTrim := telemetry.TrimToOfficialLapWindow(hamLap.Samples, hamLap.OfficialLapTime)
rusBatch := telemetry.ResampleLinear(rusTrim, telemetry.ComputeArcLengthDistance(rusTrim), 1.0)
hamBatch := telemetry.ResampleLinear(hamTrim, telemetry.ComputeArcLengthDistance(hamTrim), 1.0)
_, batchFinishDelta, _ := telemetry.ComputeDistanceDelta(rusBatch, hamBatch, officialDelta)

diff := math.Abs(streamFinishDelta - batchFinishDelta)
fmt.Println("=== STAGE 5 REDPANDA CONSUMER EQUIVALENCE VALIDATION ===")
fmt.Printf("Redpanda Topic                     : %s (%d records consumed)\n", *topic, received)
fmt.Printf("Stage 4 Batch Finish Line Delta    : %+.4f s\n", batchFinishDelta)
fmt.Printf("Stage 5 Redpanda Stream Delta      : %+.4f s\n", streamFinishDelta)
fmt.Printf("Stream vs Batch Difference         : %.6f s (%.4f ms)\n", diff, diff*1000)
if diff <= 1e-6 {
fmt.Println("Status                             : VERIFIED (Redpanda consumer matches batch output)")
}
}
