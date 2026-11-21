package main

import (
"context"
"encoding/json"
"flag"
"fmt"
"log"
"os"
"sort"
"time"

"github.com/twmb/franz-go/pkg/kgo"
"ghost-lap/pkg/telemetry"
)

type TaggedSample struct {
DriverCode string                    `json:"driver_code"`
Sample     telemetry.TelemetrySample `json:"sample"`
}

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
brokers := flag.String("brokers", "localhost:9092", "Redpanda broker addresses")
topic := flag.String("topic", "telemetry-raw", "Target topic")
speedMult := flag.Float64("speed", 10.0, "Replay speed multiplier")
flag.Parse()

rusSamples, err := loadTrimmed("data/RUS_lap.json")
if err != nil {
log.Fatal(err)
}
hamSamples, err := loadTrimmed("data/HAM_lap.json")
if err != nil {
log.Fatal(err)
}

var stream []TaggedSample
for _, s := range rusSamples {
stream = append(stream, TaggedSample{DriverCode: "RUS", Sample: s})
}
for _, s := range hamSamples {
stream = append(stream, TaggedSample{DriverCode: "HAM", Sample: s})
}
sort.Slice(stream, func(i, j int) bool {
return stream[i].Sample.RelTimeSec < stream[j].Sample.RelTimeSec
})

cl, err := kgo.NewClient(kgo.SeedBrokers(*brokers))
if err != nil {
log.Fatalf("failed to create producer: %v", err)
}
defer cl.Close()

ctx := context.Background()
start := time.Now()
prevRel := 0.0

for i, item := range stream {
if i > 0 && *speedMult > 0 {
dt := item.Sample.RelTimeSec - prevRel
if dt > 0 {
time.Sleep(time.Duration((dt / *speedMult) * float64(time.Second)))
}
}
prevRel = item.Sample.RelTimeSec

payload, _ := json.Marshal(item)
rec := &kgo.Record{
Topic: *topic,
Key:   []byte(item.DriverCode),
Value: payload,
}
if err := cl.ProduceSync(ctx, rec).FirstErr(); err != nil {
log.Fatalf("produce failed: %v", err)
}
}

fmt.Printf("Published %d trimmed lap records (RUS=%d, HAM=%d) to %s in %.2fs at %.1fx speed\n",
len(stream), len(rusSamples), len(hamSamples), *topic, time.Since(start).Seconds(), *speedMult)
}
