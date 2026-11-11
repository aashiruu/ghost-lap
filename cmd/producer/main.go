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

	"ghost-lap/pkg/telemetry"

	"github.com/twmb/franz-go/pkg/kgo"
)

func loadLap(path, driverCode string) (*telemetry.LapExport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lap telemetry.LapExport
	if err := json.Unmarshal(data, &lap); err != nil {
		return nil, err
	}
	for i := range lap.Samples {
		lap.Samples[i].DriverCode = driverCode
	}
	return &lap, nil
}

func main() {
	brokers := flag.String("brokers", "localhost:9092", "Redpanda broker address")
	topic := flag.String("topic", "telemetry-raw", "Redpanda target topic")
	speedMultiplier := flag.Float64("speed", 1.0, "Replay speed multiplier (e.g. 1.0 = real-time, 2.0 = 2x speed)")
	flag.Parse()

	rusLap, err := loadLap("data/RUS_lap.json", "RUS")
	if err != nil {
		log.Fatalf("Failed to load RUS lap: %v", err)
	}

	hamLap, err := loadLap("data/HAM_lap.json", "HAM")
	if err != nil {
		log.Fatalf("Failed to load HAM lap: %v", err)
	}

	// Merge and sort all samples by relative timestamp to preserve chronological replay order across drivers
	var allSamples []telemetry.TelemetrySample
	allSamples = append(allSamples, rusLap.Samples...)
	allSamples = append(allSamples, hamLap.Samples...)

	sort.Slice(allSamples, func(i, j int) bool {
		return allSamples[i].RelTimeSec < allSamples[j].RelTimeSec
	})

	client, err := kgo.NewClient(
		kgo.SeedBrokers(*brokers),
	)
	if err != nil {
		log.Fatalf("Failed to create Redpanda client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	fmt.Printf("Starting replay producer to topic '%s' at %.1fx speed (%d total samples)...\n", *topic, *speedMultiplier, len(allSamples))

	startTime := time.Now()
	var publishedCount int

	for i, sample := range allSamples {
		if i > 0 {
			deltaSim := sample.RelTimeSec - allSamples[i-1].RelTimeSec
			if deltaSim > 0 {
				sleepDuration := time.Duration((deltaSim / *speedMultiplier) * float64(time.Second))
				time.Sleep(sleepDuration)
			}
		}

		payload, err := json.Marshal(sample)
		if err != nil {
			log.Printf("Failed to marshal sample: %v", err)
			continue
		}

		record := &kgo.Record{
			Topic: *topic,
			Key:   []byte(sample.DriverCode),
			Value: payload,
		}

		results := client.ProduceSync(ctx, record)
		if err := results.FirstErr(); err != nil {
			log.Printf("Failed to produce record for %s: %v", sample.DriverCode, err)
		} else {
			publishedCount++
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Replay complete! Published %d samples in %v.\n", publishedCount, elapsed)
}
