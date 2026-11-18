package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"ghost-lap/pkg/telemetry"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Bounded label: driver in {"RUS", "HAM"} -> cardinality = 2
	framesProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ghostlap_frames_processed_total",
			Help: "Total raw telemetry frames processed per driver.",
		},
		[]string{"driver"},
	)

	consumerLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghostlap_consumer_lag_records",
			Help: "Simulated or broker consumer lag in records per driver partition.",
		},
		[]string{"driver"},
	)

	// Unlabeled histogram -> cardinality = 1 series (+ buckets)
	resampleDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ghostlap_resample_duration_seconds",
			Help:    "Duration of 1m distance-grid resampling and delta computation.",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12),
		},
	)

	deltaErrorSeconds = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ghostlap_delta_error_vs_official_seconds",
			Help: "Absolute difference between computed finish-line delta and official gap.",
		},
	)
)

func init() {
	prometheus.MustRegister(framesProcessed, consumerLag, resampleDuration, deltaErrorSeconds)
}

type TraceRow struct {
	Distance float64 `json:"distance"`
	SpeedRUS float64 `json:"speed_rus"`
	SpeedHAM float64 `json:"speed_ham"`
	DeltaSec float64 `json:"delta_sec"`
}

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

func integrateTimeFromSpeed(grid []telemetry.ResampledPoint, gridStep float64, officialLapTime float64) []float64 {
	times := make([]float64, len(grid))
	if len(grid) == 0 {
		return times
	}
	var rawTotal float64
	for i := 1; i < len(grid); i++ {
		vMS := ((grid[i-1].Speed + grid[i].Speed) / 2.0) / 3.6
		if vMS < 5.0 {
			vMS = 5.0
		}
		rawTotal += gridStep / vMS
		times[i] = rawTotal
	}
	if rawTotal > 0 {
		scale := officialLapTime / rawTotal
		for i := range times {
			times[i] *= scale
		}
	}
	return times
}

func main() {
	addr := flag.String("addr", ":9090", "Metrics and trace HTTP listen address")
	flag.Parse()

	rusLap, err := loadLap("data/RUS_lap.json")
	if err != nil {
		log.Fatalf("Failed to load RUS lap: %v", err)
	}
	hamLap, err := loadLap("data/HAM_lap.json")
	if err != nil {
		log.Fatalf("Failed to load HAM lap: %v", err)
	}

	// Execute instrumented streaming pass
	start := time.Now()

	rusStream := telemetry.NewDriverStreamState("RUS")
	hamStream := telemetry.NewDriverStreamState("HAM")

	totalRUS := len(rusLap.Samples)
	totalHAM := len(hamLap.Samples)
	maxSamples := totalRUS
	if totalHAM > maxSamples {
		maxSamples = totalHAM
	}

	for i := 0; i < maxSamples; i++ {
		if i < totalRUS {
			rusStream.PushSample(rusLap.Samples[i])
			framesProcessed.WithLabelValues("RUS").Inc()
			consumerLag.WithLabelValues("RUS").Set(float64(totalRUS - 1 - i))
		}
		if i < totalHAM {
			hamStream.PushSample(hamLap.Samples[i])
			framesProcessed.WithLabelValues("HAM").Inc()
			consumerLag.WithLabelValues("HAM").Set(float64(totalHAM - 1 - i))
		}
	}

	gridStep := 1.0
	rusGrid := rusStream.ComputeIncrementalPoints(gridStep)
	hamGrid := hamStream.ComputeIncrementalPoints(gridStep)

	rusTimes := integrateTimeFromSpeed(rusGrid, gridStep, rusLap.OfficialLapTime)
	hamTimes := integrateTimeFromSpeed(hamGrid, gridStep, hamLap.OfficialLapTime)

	n := len(rusGrid)
	if len(hamGrid) < n {
		n = len(hamGrid)
	}

	traces := make([]TraceRow, n)
	for i := 0; i < n; i++ {
		traces[i] = TraceRow{
			Distance: rusGrid[i].Distance,
			SpeedRUS: rusGrid[i].Speed,
			SpeedHAM: hamGrid[i].Speed,
			DeltaSec: hamTimes[i] - rusTimes[i],
		}
	}

	elapsed := time.Since(start).Seconds()
	resampleDuration.Observe(elapsed)

	officialDelta := hamLap.OfficialLapTime - rusLap.OfficialLapTime
	finishDelta := traces[n-1].DeltaSec
	errSec := finishDelta - officialDelta
	if errSec < 0 {
		errSec = -errSec
	}
	deltaErrorSeconds.Set(errSec)

	fmt.Printf("Pipeline metrics ready: processed %d RUS + %d HAM frames in %.4f ms (delta error: %.6f s)\n",
		totalRUS, totalHAM, elapsed*1000, errSec)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/api/trace", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(traces)
	})

	fmt.Printf("Exposing Prometheus metrics at http://localhost%s/metrics and trace data at http://localhost%s/api/trace\n", *addr, *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
