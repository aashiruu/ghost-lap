package telemetry

import (
	"math/rand"
)

// FilterSamplesWithLoss drops samples based on a reproducible random seed rate (0.0 to 1.0)
func FilterSamplesWithLoss(samples []TelemetrySample, dropRate float64, seed int64) []TelemetrySample {
	if dropRate <= 0.0 {
		return samples
	}
	r := rand.New(rand.NewSource(seed))

	var filtered []TelemetrySample
	// Always retain first and last sample to anchor lap boundaries
	for i, s := range samples {
		if i == 0 || i == len(samples)-1 {
			filtered = append(filtered, s)
			continue
		}
		if r.Float64() >= dropRate {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
