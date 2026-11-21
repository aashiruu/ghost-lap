package main

import (
"encoding/json"
"flag"
"fmt"
"log"
"net/http"
"os"
"time"

"github.com/prometheus/client_golang/prometheus"
"github.com/prometheus/client_golang/prometheus/promhttp"
"ghost-lap/pkg/telemetry"
)

var (
framesProcessed = prometheus.NewCounterVec(
prometheus.CounterOpts{
Name: "ghostlap_frames_processed_total",
Help: "Total trimmed lap telemetry frames processed per driver.",
},
[]string{"driver"},
)

arcLengthMeters = prometheus.NewGaugeVec(
prometheus.GaugeOpts{
Name: "ghostlap_arc_length_meters",
Help: "Computed 2D arc-length distance over the official lap window per driver.",
},
[]string{"driver"},
)

resampleDuration = prometheus.NewHistogram(
prometheus.HistogramOpts{
Name:    "ghostlap_resample_duration_seconds",
Help:    "Wall-clock duration of 1m distance-grid resampling and uncalibrated delta computation.",
Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12),
},
)

uncalibratedFinishDelta = prometheus.NewGauge(
prometheus.GaugeOpts{
Name: "ghostlap_uncalibrated_finish_delta_seconds",
Help: "Measured uncalibrated finish-line time delta (HAM - RUS) at overlapping grid end.",
},
)

officialFinishDelta = prometheus.NewGauge(
prometheus.GaugeOpts{
Name: "ghostlap_official_finish_delta_seconds",
Help: "Official FIA/OpenF1 lap-time difference (HAM - RUS).",
},
)
)

func init() {
prometheus.MustRegister(framesProcessed, arcLengthMeters, resampleDuration, uncalibratedFinishDelta, officialFinishDelta)
}

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
addr := flag.String("addr", ":9090", "HTTP listen address for /metrics and /api/trace")
flag.Parse()

rusLap, rusSamples, err := loadLap("data/RUS_lap.json")
if err != nil {
log.Fatalf("load RUS: %v", err)
}
hamLap, hamSamples, err := loadLap("data/HAM_lap.json")
if err != nil {
log.Fatalf("load HAM: %v", err)
}

start := time.Now()
rusArc := telemetry.ComputeArcLengthDistance(rusSamples)
hamArc := telemetry.ComputeArcLengthDistance(hamSamples)
rusGrid := telemetry.ResampleLinear(rusSamples, rusArc, 1.0)
hamGrid := telemetry.ResampleLinear(hamSamples, hamArc, 1.0)

officialGap := hamLap.OfficialLapTime - rusLap.OfficialLapTime
deltas, finishDelta, absErr := telemetry.ComputeDistanceDelta(rusGrid, hamGrid, officialGap)
elapsed := time.Since(start)

framesProcessed.WithLabelValues("RUS").Add(float64(len(rusSamples)))
framesProcessed.WithLabelValues("HAM").Add(float64(len(hamSamples)))
arcLengthMeters.WithLabelValues("RUS").Set(rusArc[len(rusArc)-1])
arcLengthMeters.WithLabelValues("HAM").Set(hamArc[len(hamArc)-1])
resampleDuration.Observe(elapsed.Seconds())
uncalibratedFinishDelta.Set(finishDelta)
officialFinishDelta.Set(officialGap)

fmt.Printf("Pipeline metrics ready: RUS=%d HAM=%d frames in %.4f ms (uncalibrated delta: %+.4fs, official: %+.4fs, err: %.2fms)\n",
len(rusSamples), len(hamSamples), float64(elapsed.Microseconds())/1000.0, finishDelta, officialGap, absErr*1000)

traceBytes, _ := json.Marshal(deltas)

http.Handle("/metrics", promhttp.Handler())
http.HandleFunc("/api/trace", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
_, _ = w.Write(traceBytes)
})

fmt.Printf("Exposing Prometheus metrics at http://localhost%s/metrics and trace data at http://localhost%s/api/trace\n", *addr, *addr)
log.Fatal(http.ListenAndServe(*addr, nil))
}
