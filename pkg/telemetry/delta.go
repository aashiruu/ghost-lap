package telemetry

import "math"

type DeltaPoint struct {
	Distance   float64 `json:"distance"`
	TimeRUS    float64 `json:"time_rus"`
	TimeHAM    float64 `json:"time_ham"`
	DeltaSec   float64 `json:"delta_sec"` // Positive means HAM is behind RUS
	SpeedRUS   float64 `json:"speed_rus"`
	SpeedHAM   float64 `json:"speed_ham"`
	LeaderCode string  `json:"leader_code"`
}

// ComputeDistanceDelta computes uncalibrated spatial time delta across the overlapping 1m grid.
func ComputeDistanceDelta(rusGrid, hamGrid []ResampledPoint, officialDelta float64) ([]DeltaPoint, float64, float64) {
	n := len(rusGrid)
	if len(hamGrid) < n {
		n = len(hamGrid)
	}
	if n == 0 {
		return nil, 0, 0
	}

	deltas := make([]DeltaPoint, n)
	for i := 0; i < n; i++ {
		tRUS := rusGrid[i].RelTimeSec
		tHAM := hamGrid[i].RelTimeSec
		dt := tHAM - tRUS

		leader := "RUS"
		if dt < 0 {
			leader = "HAM"
		}

		deltas[i] = DeltaPoint{
			Distance:   rusGrid[i].Distance,
			TimeRUS:    tRUS,
			TimeHAM:    tHAM,
			DeltaSec:   dt,
			SpeedRUS:   rusGrid[i].Speed,
			SpeedHAM:   hamGrid[i].Speed,
			LeaderCode: leader,
		}
	}

	finishDelta := deltas[n-1].DeltaSec
	absoluteError := math.Abs(finishDelta - officialDelta)

	return deltas, finishDelta, absoluteError
}
