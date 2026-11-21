package telemetry

import "math"

type TrackSegment struct {
Name      string  `json:"name"`
StartDist float64 `json:"start_dist"`
EndDist   float64 `json:"end_dist"`
}

type SegmentResult struct {
Name        string  `json:"name"`
StartDist   float64 `json:"start_dist"`
EndDist     float64 `json:"end_dist"`
LengthM     float64 `json:"length_m"`
TimeRUS     float64 `json:"time_rus"`
TimeHAM     float64 `json:"time_ham"`
AvgSpeedRUS float64 `json:"avg_speed_rus_ms"`
AvgSpeedHAM float64 `json:"avg_speed_ham_ms"`
SegmentDiff float64 `json:"segment_diff"`
Winner      string  `json:"winner"`
}

func SilverstoneSegments(maxDist float64) []TrackSegment {
return []TrackSegment{
{Name: "Band 1 (0-780 m)", StartDist: 0, EndDist: 780},
{Name: "Band 2 (780-1560 m)", StartDist: 780, EndDist: 1560},
{Name: "Band 3 (1560-2520 m)", StartDist: 1560, EndDist: 2520},
{Name: "Band 4 (2520-3180 m)", StartDist: 2520, EndDist: 3180},
{Name: "Band 5 (3180-3920 m)", StartDist: 3180, EndDist: 3920},
{Name: "Band 6 (3920-4780 m)", StartDist: 3920, EndDist: 4780},
{Name: "Band 7 (4780-5823 m)", StartDist: 4780, EndDist: maxDist},
}
}

func ComputeSegmentDeltas(rusGrid, hamGrid []ResampledPoint, segments []TrackSegment) ([]SegmentResult, float64, float64) {
maxIdx := len(rusGrid) - 1
if len(hamGrid)-1 < maxIdx {
maxIdx = len(hamGrid) - 1
}

scale := 1.0
if len(segments) > 0 && segments[len(segments)-1].EndDist > 0 {
scale = float64(maxIdx) / segments[len(segments)-1].EndDist
}

results := make([]SegmentResult, len(segments))
var sumSegmentDiff float64

for i, seg := range segments {
startIdx := int(math.Round(seg.StartDist * scale))
endIdx := int(math.Round(seg.EndDist * scale))
if startIdx < 0 {
startIdx = 0
}
if startIdx > maxIdx {
startIdx = maxIdx
}
if endIdx < startIdx {
endIdx = startIdx
}
if endIdx > maxIdx {
endIdx = maxIdx
}

lengthM := float64(endIdx - startIdx)
dtRUS := rusGrid[endIdx].RelTimeSec - rusGrid[startIdx].RelTimeSec
dtHAM := hamGrid[endIdx].RelTimeSec - hamGrid[startIdx].RelTimeSec
diff := dtHAM - dtRUS
sumSegmentDiff += diff

avgSpdRUS := 0.0
if dtRUS > 0 {
avgSpdRUS = lengthM / dtRUS
}
avgSpdHAM := 0.0
if dtHAM > 0 {
avgSpdHAM = lengthM / dtHAM
}

winner := "RUS"
if diff < 0 {
winner = "HAM"
}

results[i] = SegmentResult{
Name:        seg.Name,
StartDist:   float64(startIdx),
EndDist:     float64(endIdx),
LengthM:     lengthM,
TimeRUS:     dtRUS,
TimeHAM:     dtHAM,
AvgSpeedRUS: avgSpdRUS,
AvgSpeedHAM: avgSpdHAM,
SegmentDiff: diff,
Winner:      winner,
}
}

totalDelta := hamGrid[maxIdx].RelTimeSec - rusGrid[maxIdx].RelTimeSec
conservationErr := math.Abs(sumSegmentDiff - totalDelta)
return results, sumSegmentDiff, conservationErr
}
