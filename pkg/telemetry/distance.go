package telemetry

import (
	"math"
)

// ComputeIntegratedDistance calculates distance by integrating speed over time step dt.
// speed is in km/h, relTimeSec in seconds.
func ComputeIntegratedDistance(samples []TelemetrySample) []float64 {
	distances := make([]float64, len(samples))
	if len(samples) == 0 {
		return distances
	}

	distances[0] = 0.0
	for i := 1; i < len(samples); i++ {
		dt := samples[i].RelTimeSec - samples[i-1].RelTimeSec
		// Convert speed from km/h to m/s
		vMS := (samples[i-1].Speed + samples[i].Speed) / 2.0 / 3.6
		distances[i] = distances[i-1] + (vMS * dt)
	}
	return distances
}

// ComputeProjectedDistance maps (X,Y) coordinates onto a reference lap spatial line.
func ComputeProjectedDistance(driverSamples []TelemetrySample, refSamples []TelemetrySample) []float64 {
	distances := make([]float64, len(driverSamples))
	if len(driverSamples) == 0 || len(refSamples) == 0 {
		return distances
	}

	for i, ds := range driverSamples {
		minDistSq := math.MaxFloat64
		bestIdx := 0

		// Find nearest reference track point (spatial projection)
		for rIdx, rs := range refSamples {
			dx := ds.X - rs.X
			dy := ds.Y - rs.Y
			distSq := dx*dx + dy*dy
			if distSq < minDistSq {
				minDistSq = distSq
				bestIdx = rIdx
			}
		}
		distances[i] = refSamples[bestIdx].Distance
	}
	return distances
}
