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

// TrimToOfficialLapWindow interpolates exact boundary samples at t=0.0 and t=officialLapTime
// from the buffered OpenF1 export so distance and time are bounded strictly to the flying lap.
func TrimToOfficialLapWindow(samples []TelemetrySample, officialLapTime float64) []TelemetrySample {
if len(samples) < 2 {
return samples
}

interpAt := func(targetT float64) TelemetrySample {
if targetT <= samples[0].RelTimeSec {
s := samples[0]
s.RelTimeSec = targetT
return s
}
if targetT >= samples[len(samples)-1].RelTimeSec {
s := samples[len(samples)-1]
s.RelTimeSec = targetT
return s
}
for i := 0; i < len(samples)-1; i++ {
s0, s1 := samples[i], samples[i+1]
if targetT >= s0.RelTimeSec && targetT <= s1.RelTimeSec {
alpha := 0.0
if s1.RelTimeSec > s0.RelTimeSec {
alpha = (targetT - s0.RelTimeSec) / (s1.RelTimeSec - s0.RelTimeSec)
}
return TelemetrySample{
RelTimeSec: targetT,
Date:       s0.Date,
Speed:      lerp(s0.Speed, s1.Speed, alpha),
Throttle:   lerp(s0.Throttle, s1.Throttle, alpha),
Brake:      s0.Brake,
Gear:       s0.Gear,
X:          lerp(s0.X, s1.X, alpha),
Y:          lerp(s0.Y, s1.Y, alpha),
Z:          lerp(s0.Z, s1.Z, alpha),
}
}
}
return samples[0]
}

trimmed := make([]TelemetrySample, 0, len(samples))
trimmed = append(trimmed, interpAt(0.0))
for _, s := range samples {
if s.RelTimeSec > 0.0 && s.RelTimeSec < officialLapTime {
trimmed = append(trimmed, s)
}
}
trimmed = append(trimmed, interpAt(officialLapTime))
return trimmed
}

// ComputeSpeedIntegratedDistance integrates trapezoidal speed (km/h -> m/s) over sample dt.
func ComputeSpeedIntegratedDistance(samples []TelemetrySample) []float64 {
dist := make([]float64, len(samples))
for i := 1; i < len(samples); i++ {
dt := samples[i].RelTimeSec - samples[i-1].RelTimeSec
if dt < 0 {
dt = 0
}
avgSpeedMS := ((samples[i-1].Speed + samples[i].Speed) / 2.0) / 3.6
dist[i] = dist[i-1] + avgSpeedMS*dt
}
return dist
}

// ComputeArcLengthDistance computes cumulative 2D Euclidean arc length in meters
// from OpenF1 location coordinates (which are reported in decimeters, 1 unit = 0.1 m).
func ComputeArcLengthDistance(samples []TelemetrySample) []float64 {
dist := make([]float64, len(samples))
for i := 1; i < len(samples); i++ {
dx := samples[i].X - samples[i-1].X
dy := samples[i].Y - samples[i-1].Y
stepMeters := math.Hypot(dx, dy) / 10.0
dist[i] = dist[i-1] + stepMeters
}
return dist
}

// ResampleLinear maps non-uniform samples onto a uniform spatial distance grid (gridStep in meters).
// RelTimeSec is interpolated directly from each sample's real timestamp and arc distance (no calibration).
func ResampleLinear(samples []TelemetrySample, srcDist []float64, gridStep float64) []ResampledPoint {
if len(samples) < 2 || len(srcDist) != len(samples) {
return nil
}

maxDist := srcDist[len(srcDist)-1]
if maxDist <= 0 {
return nil
}

numSteps := int(math.Floor(maxDist/gridStep)) + 1
result := make([]ResampledPoint, numSteps)

srcIdx := 0
for i := 0; i < numSteps; i++ {
targetDist := float64(i) * gridStep
for srcIdx < len(srcDist)-2 && srcDist[srcIdx+1] < targetDist {
srcIdx++
}

d0, d1 := srcDist[srcIdx], srcDist[srcIdx+1]
t := 0.0
if d1 > d0 {
t = clamp((targetDist-d0)/(d1-d0), 0.0, 1.0)
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
