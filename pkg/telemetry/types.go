package telemetry

type TelemetrySample struct {
	DriverCode  string  `json:"driver_code"`
	RelTimeSec  float64 `json:"rel_time_sec"`
	Date        string  `json:"date"`
	Speed       float64 `json:"speed"`
	Throttle    float64 `json:"throttle"`
	Brake       bool    `json:"brake"`
	Gear        int     `json:"gear"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Z           float64 `json:"z"`
	Distance    float64 `json:"distance"`
}

type LapExport struct {
	OfficialLapTime float64           `json:"official_lap_time"`
	SampleCount     int               `json:"sample_count"`
	Samples         []TelemetrySample `json:"samples"`
}
