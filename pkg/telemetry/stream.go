package telemetry

import "math"

type DriverStreamState struct {
	DriverCode string
	Samples    []TelemetrySample
	ArcDist    []float64
	LastGridIdx int
}

func NewDriverStreamState(code string) *DriverStreamState {
	return &DriverStreamState{
		DriverCode:  code,
		Samples:     make([]TelemetrySample, 0),
		ArcDist:     make([]float64, 0),
		LastGridIdx: 0,
	}
}

func (s *DriverStreamState) PushSample(sample TelemetrySample) {
	s.Samples = append(s.Samples, sample)
	n := len(s.Samples)
	if n == 1 {
		s.ArcDist = append(s.ArcDist, 0.0)
		return
	}

	prev := s.Samples[n-2]
	dx := sample.X - prev.X
	dy := sample.Y - prev.Y
	step := math.Hypot(dx, dy)
	if step == 0 {
		step = sample.Distance - prev.Distance
		if step <= 0 {
			step = 8.537
		}
	}
	s.ArcDist = append(s.ArcDist, s.ArcDist[n-2]+step)
}

// ComputeIncrementalPoints processes available samples into resampled distance grid up to current max arc distance
func (s *DriverStreamState) ComputeIncrementalPoints(gridStep float64) []ResampledPoint {
	if len(s.Samples) == 0 {
		return nil
	}
	maxDist := s.ArcDist[len(s.ArcDist)-1]
	targetSteps := int(maxDist / gridStep)

	var points []ResampledPoint
	for i := s.LastGridIdx; i < targetSteps; i++ {
		targetDist := float64(i) * gridStep

		// Locate interpolation interval
		idx := 0
		for idx < len(s.ArcDist)-1 && s.ArcDist[idx+1] <= targetDist {
			idx++
		}

		if idx >= len(s.ArcDist)-1 {
			last := s.Samples[len(s.Samples)-1]
			points = append(points, ResampledPoint{
				Distance:   targetDist,
				RelTimeSec: last.RelTimeSec,
				Speed:      last.Speed,
				Throttle:   last.Throttle,
				Brake:      b2f(last.Brake),
				Gear:       float64(last.Gear),
				X:          last.X,
				Y:          last.Y,
			})
			continue
		}

		d0, d1 := s.ArcDist[idx], s.ArcDist[idx+1]
		t := 0.0
		if d1 > d0 {
			t = (targetDist - d0) / (d1 - d0)
		}

		s0, s1 := s.Samples[idx], s.Samples[idx+1]
		points = append(points, ResampledPoint{
			Distance:   targetDist,
			RelTimeSec: lerp(s0.RelTimeSec, s1.RelTimeSec, t),
			Speed:      lerp(s0.Speed, s1.Speed, t),
			Throttle:   clamp(lerp(s0.Throttle, s1.Throttle, t), 0, 100),
			Brake:      clamp(lerp(b2f(s0.Brake), b2f(s1.Brake), t), 0, 1),
			Gear:       math.Round(lerp(float64(s0.Gear), float64(s1.Gear), t)),
			X:          lerp(s0.X, s1.X, t),
			Y:          lerp(s0.Y, s1.Y, t),
		})
	}

	s.LastGridIdx = targetSteps
	return points
}
