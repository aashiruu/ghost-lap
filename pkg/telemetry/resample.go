package telemetry

import (
	"math"
)

type ResampledPoint struct {
	Distance   float64 `json:"distance"`
	RelTimeSec float64 `json:"rel_time_sec"`
	Speed      float64 `json:"speed"`
	Throttle   float64 `json:"throttle"`
	Brake      float64 `json:"brake"`
	Gear       float64 `json:"gear"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
}

func ComputeArcLengthDistance(samples []TelemetrySample) []float64 {
	dist := make([]float64, len(samples))
	if len(samples) == 0 {
		return dist
	}
	dist[0] = 0.0
	for i := 1; i < len(samples); i++ {
		dx := samples[i].X - samples[i-1].X
		dy := samples[i].Y - samples[i-1].Y
		step := math.Hypot(dx, dy)
		if step == 0 {
			step = samples[i].Distance - samples[i-1].Distance
			if step <= 0 {
				step = 8.537
			}
		}
		dist[i] = dist[i-1] + step
	}
	return dist
}

func ResampleLinear(samples []TelemetrySample, srcDist []float64, gridStep float64) []ResampledPoint {
	if len(samples) == 0 || len(srcDist) == 0 {
		return nil
	}

	maxDist := srcDist[len(srcDist)-1]
	if maxDist <= 0 {
		return nil
	}

	numSteps := int(maxDist / gridStep)
	if numSteps <= 0 {
		return nil
	}

	result := make([]ResampledPoint, numSteps)

	srcIdx := 0
	for i := 0; i < numSteps; i++ {
		targetDist := float64(i) * gridStep

		for srcIdx < len(srcDist)-1 && srcDist[srcIdx+1] <= targetDist {
			srcIdx++
		}

		if srcIdx >= len(srcDist)-1 {
			last := samples[len(samples)-1]
			result[i] = ResampledPoint{
				Distance:   targetDist,
				RelTimeSec: last.RelTimeSec,
				Speed:      last.Speed,
				Throttle:   last.Throttle,
				Brake:      b2f(last.Brake),
				Gear:       float64(last.Gear),
				X:          last.X,
				Y:          last.Y,
			}
			continue
		}

		d0, d1 := srcDist[srcIdx], srcDist[srcIdx+1]
		t := 0.0
		if d1 > d0 {
			t = (targetDist - d0) / (d1 - d0)
		}

		s0, s1 := samples[srcIdx], samples[srcIdx+1]

		result[i] = ResampledPoint{
			Distance:   targetDist,
			RelTimeSec: lerp(s0.RelTimeSec, s1.RelTimeSec, t),
			Speed:      lerp(s0.Speed, s1.Speed, t),
			Throttle:   clamp(lerp(s0.Throttle, s1.Throttle, t), 0, 100),
			Brake:      clamp(lerp(b2f(s0.Brake), b2f(s1.Brake), t), 0, 1),
			Gear:       math.Round(lerp(float64(s0.Gear), float64(s1.Gear), t)),
			X:          lerp(s0.X, s1.X, t),
			Y:          lerp(s0.Y, s1.Y, t),
		}
	}

	return result
}

func lerp(a, b, t float64) float64 {
	return a + t*(b-a)
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func b2f(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
