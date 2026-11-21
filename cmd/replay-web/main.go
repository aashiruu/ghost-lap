package main

import (
"encoding/json"
"flag"
"fmt"
"log"
"net/http"
"os"

"ghost-lap/pkg/telemetry"
)

type ReplayFrame struct {
Distance   float64 `json:"distance"`
TimeRUS    float64 `json:"time_rus"`
TimeHAM    float64 `json:"time_ham"`
DeltaSec   float64 `json:"delta_sec"`
SpeedRUS   float64 `json:"speed_rus"`
SpeedHAM   float64 `json:"speed_ham"`
LeaderCode string  `json:"leader_code"`
XRUS       float64 `json:"x_rus"`
YRUS       float64 `json:"y_rus"`
XHAM       float64 `json:"x_ham"`
YHAM       float64 `json:"y_ham"`
}

type ReplayPayload struct {
Session         string        `json:"session"`
OfficialLapRUS  float64       `json:"official_lap_rus"`
OfficialLapHAM  float64       `json:"official_lap_ham"`
OfficialDelta   float64       `json:"official_delta"`
ComputedDelta   float64       `json:"computed_delta"`
TotalDistanceM  float64       `json:"total_distance_m"`
Frames          []ReplayFrame `json:"frames"`
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
addr := flag.String("addr", ":8080", "HTTP listen address")
flag.Parse()

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

officialDelta := hamLap.OfficialLapTime - rusLap.OfficialLapTime
deltas, finishDelta, _ := telemetry.ComputeDistanceDelta(rusGrid, hamGrid, officialDelta)

frames := make([]ReplayFrame, len(deltas))
for i, d := range deltas {
frames[i] = ReplayFrame{
Distance:   d.Distance,
TimeRUS:    d.TimeRUS,
TimeHAM:    d.TimeHAM,
DeltaSec:   d.DeltaSec,
SpeedRUS:   d.SpeedRUS,
SpeedHAM:   d.SpeedHAM,
LeaderCode: d.LeaderCode,
XRUS:       rusGrid[i].X,
YRUS:       rusGrid[i].Y,
XHAM:       hamGrid[i].X,
YHAM:       hamGrid[i].Y,
}
}

payload := ReplayPayload{
Session:        "2024 Silverstone Qualifying Q3 (OpenF1 Session 9554)",
OfficialLapRUS: rusLap.OfficialLapTime,
OfficialLapHAM: hamLap.OfficialLapTime,
OfficialDelta:  officialDelta,
ComputedDelta:  finishDelta,
TotalDistanceM: frames[len(frames)-1].Distance,
Frames:         frames,
}

outBytes, _ := json.Marshal(payload)
if err := os.WriteFile("web/replay_data.json", outBytes, 0644); err != nil {
log.Fatal(err)
}

fmt.Printf("Exported %d aligned replay frames to web/replay_data.json (Uncalibrated delta: %+.4fs vs Official: %+.4fs)\n",
len(frames), finishDelta, officialDelta)
fmt.Printf("Serving track replay view at http://localhost%s\n", *addr)
http.Handle("/", http.FileServer(http.Dir("web")))
log.Fatal(http.ListenAndServe(*addr, nil))
}
