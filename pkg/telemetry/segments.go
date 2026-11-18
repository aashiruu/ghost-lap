package telemetry

import "math"

type TrackSegment struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"` // "Corner Complex", "Braking Zone", "Straight"
	StartDist float64 `json:"start_dist"`
	EndDist   float64 `json:"end_dist"`
}

type SegmentResult struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	StartDist   float64 `json:"start_dist"`
	EndDist     float64 `json:"end_dist"`
	TimeRUS     float64 `json:"time_rus"`
	TimeHAM     float64 `json:"time_ham"`
	SegmentDiff float64 `json:"segment_diff"` // Positive = HAM lost time (RUS faster); Negative = HAM gained time
	Winner      string  `json:"winner"`
}

// SilverstoneSegments defines contiguous track segments covering 0 to 5856m
func SilverstoneSegments(maxDist float64) []TrackSegment {
	return []TrackSegment{
		{Name: "S1: Grid & Abbey / Farm (T1-T2)", Type: "High-Speed Corner", StartDist: 0, EndDist: 780},
		{Name: "S2: Village / The Loop / Aintree (T3-T5)", Type: "Heavy Braking & Technical", StartDist: 780, EndDist: 1560},
		{Name: "S3: Wellington Straight & Brooklands (T6)", Type: "Straight + Braking Zone", StartDist: 1560, EndDist: 2520},
		{Name: "S4: Luffield & Woodcote (T7-T8)", Type: "Low-Speed Traction", StartDist: 2520, EndDist: 3180},
		{Name: "S5: Old Pit Straight & Copse (T9)", Type: "High-Speed Corner", StartDist: 3180, EndDist: 3920},
		{Name: "S6: Maggotts / Becketts / Chapel (T10-T14)", Type: "Rapid Direction Change", StartDist: 3920, EndDist: 4780},
		{Name: "S7: Hangar Straight, Stowe & Vale/Club (T15-T18)", Type: "Braking Zone & Chicane", StartDist: 4780, EndDist: maxDist},
	}
}

// ComputeSegmentDeltas computes per-segment elapsed times and verifies telescoping sum conservation
func ComputeSegmentDeltas(rusGrid, hamGrid []ResampledPoint, rusTimes, hamTimes []float64, segments []TrackSegment) ([]SegmentResult, float64, float64) {
	maxIdx := len(rusTimes) - 1
	if len(hamTimes)-1 < maxIdx {
		maxIdx = len(hamTimes) - 1
	}

	results := make([]SegmentResult, len(segments))
	var sumSegmentDiff float64

	for i, seg := range segments {
		startIdx := int(math.Round(seg.StartDist))
		endIdx := int(math.Round(seg.EndDist))
		if startIdx < 0 {
			startIdx = 0
		}
		if endIdx > maxIdx {
			endIdx = maxIdx
		}

		dtRUS := rusTimes[endIdx] - rusTimes[startIdx]
		dtHAM := hamTimes[endIdx] - hamTimes[startIdx]
		diff := dtHAM - dtRUS
		sumSegmentDiff += diff

		winner := "RUS"
		if diff < 0 {
			winner = "HAM"
		}

		results[i] = SegmentResult{
			Name:        seg.Name,
			Type:        seg.Type,
			StartDist:   float64(startIdx),
			EndDist:     float64(endIdx),
			TimeRUS:     dtRUS,
			TimeHAM:     dtHAM,
			SegmentDiff: diff,
			Winner:      winner,
		}
	}

	totalDelta := hamTimes[maxIdx] - rusTimes[maxIdx]
	conservationErr := math.Abs(sumSegmentDiff - totalDelta)
	return results, sumSegmentDiff, conservationErr
}
