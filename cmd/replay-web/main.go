package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
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
	RusX       float64 `json:"rus_x"`
	RusY       float64 `json:"rus_y"`
	HamX       float64 `json:"ham_x"`
	HamY       float64 `json:"ham_y"`
}

type ReplayPayload struct {
	Session         string        `json:"session"`
	DriverA         string        `json:"driver_a"`
	DriverB         string        `json:"driver_b"`
	OfficialLapA    float64       `json:"official_lap_a"`
	OfficialLapB    float64       `json:"official_lap_b"`
	OfficialDelta   float64       `json:"official_delta"`
	TotalDistanceM  float64       `json:"total_distance_m"`
	Frames          []ReplayFrame `json:"frames"`
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
	addr := flag.String("addr", ":8080", "HTTP listen address")
	exportOnly := flag.Bool("export-only", false, "Generate web/replay_data.json and exit")
	flag.Parse()

	rusLap, err := loadLap("data/RUS_lap.json")
	if err != nil {
		log.Fatalf("Failed to load RUS lap: %v", err)
	}
	hamLap, err := loadLap("data/HAM_lap.json")
	if err != nil {
		log.Fatalf("Failed to load HAM lap: %v", err)
	}

	gridStep := 1.0
	rusArc := telemetry.ComputeArcLengthDistance(rusLap.Samples)
	hamArc := telemetry.ComputeArcLengthDistance(hamLap.Samples)
	rusGrid := telemetry.ResampleLinear(rusLap.Samples, rusArc, gridStep)
	hamGrid := telemetry.ResampleLinear(hamLap.Samples, hamArc, gridStep)

	rusTimes := integrateTimeFromSpeed(rusGrid, gridStep, rusLap.OfficialLapTime)
	hamTimes := integrateTimeFromSpeed(hamGrid, gridStep, hamLap.OfficialLapTime)

	n := len(rusGrid)
	if len(hamGrid) < n {
		n = len(hamGrid)
	}

	// Check if raw X/Y coordinates have real spatial variance; otherwise project onto closed circuit coordinates
	var minX, maxX = math.MaxFloat64, -math.MaxFloat64
	for i := 0; i < n; i++ {
		if rusGrid[i].X < minX {
			minX = rusGrid[i].X
		}
		if rusGrid[i].X > maxX {
			maxX = rusGrid[i].X
		}
	}
	hasRealXY := (maxX - minX) > 10.0

	frames := make([]ReplayFrame, n)
	for i := 0; i < n; i++ {
		dt := hamTimes[i] - rusTimes[i]
		leader := "RUS"
		if dt < 0 {
			leader = "HAM"
		}

		rx, ry := rusGrid[i].X, rusGrid[i].Y
		hx, hy := hamGrid[i].X, hamGrid[i].Y
		if !hasRealXY {
			theta := (float64(i) / float64(n)) * 2.0 * math.Pi
			rx = 400*math.Cos(theta) + 120*math.Sin(2*theta)
			ry = 260*math.Sin(theta) + 70*math.Cos(3*theta)
			hx, hy = rx, ry
		}

		frames[i] = ReplayFrame{
			Distance:   rusGrid[i].Distance,
			TimeRUS:    rusTimes[i],
			TimeHAM:    hamTimes[i],
			DeltaSec:   dt,
			SpeedRUS:   rusGrid[i].Speed,
			SpeedHAM:   hamGrid[i].Speed,
			LeaderCode: leader,
			RusX:       rx,
			RusY:       ry,
			HamX:       hx,
			HamY:       hy,
		}
	}

	payload := ReplayPayload{
		Session:        "2024 Silverstone Qualifying Q3 (Historical Replay)",
		DriverA:        "RUS",
		DriverB:        "HAM",
		OfficialLapA:   rusLap.OfficialLapTime,
		OfficialLapB:   hamLap.OfficialLapTime,
		OfficialDelta:  hamLap.OfficialLapTime - rusLap.OfficialLapTime,
		TotalDistanceM: frames[n-1].Distance,
		Frames:         frames,
	}

	if err := os.MkdirAll("web", 0755); err != nil {
		log.Fatalf("Failed to create web directory: %v", err)
	}
	outBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal replay JSON: %v", err)
	}
	if err := os.WriteFile("web/replay_data.json", outBytes, 0644); err != nil {
		log.Fatalf("Failed to write web/replay_data.json: %v", err)
	}

	fmt.Printf("Exported %d aligned replay frames to web/replay_data.json (Finish delta: %+.4fs)\n",
		len(frames), frames[n-1].DeltaSec)

	if *exportOnly {
		return
	}

	fmt.Printf("Serving track replay view at http://localhost%s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, http.FileServer(http.Dir("web"))))
}
