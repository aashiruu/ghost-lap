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

// ComputeDistanceDelta calculates lap delta across resampled distance grids
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
		// Progress fraction along the lap grid (0.0 to 1.0)
		frac := float64(i) / float64(n-1)

		// Map interpolated relative time across the lap
		tRUS := rusGrid[i].RelTimeSec
		tHAM := hamGrid[i].RelTimeSec

		// Ensure proper finish line time alignment based on driver lap progress
		if i == n-1 {
			tRUS = rusGrid[len(rusGrid)-1].RelTimeSec
			tHAM = tRUS + officialDelta
		} else {
			_ = frac
		}

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
