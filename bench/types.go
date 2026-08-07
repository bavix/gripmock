package main

type startupStats struct {
	Min  float64 `json:"min"`
	Avg  float64 `json:"avg"`
	Max  float64 `json:"max"`
	Runs int     `json:"runs"`
}

type meta struct {
	Image    string             `json:"image"`
	SizeMB   map[string]float64 `json:"size_mb"`
	Startup  startupStats       `json:"startup"`
	MemoryMB float64            `json:"memory_mb"`
}

type throughputPoint struct {
	Concurrency int     `json:"concurrency"`
	RPS         float64 `json:"rps"`
}

type benchReport struct {
	Run struct {
		StartedAt int64 `json:"started_at"`
		EndedAt   int64 `json:"ended_at"`
	} `json:"run"`
	OptionsResolved map[string]struct {
		Value string `json:"value"`
	} `json:"options_resolved"`
	Summary struct {
		Count       uint64  `json:"count"`
		OK          uint64  `json:"ok"`
		Errors      uint64  `json:"errors"`
		AverageNs   uint64  `json:"average_ns"`
		RPSObserved float64 `json:"rps_observed"`
	} `json:"summary"`
	LatencyDistribution []struct {
		Percentile float64 `json:"percentile"`
		LatencyNs  uint64  `json:"latency_ns"`
	} `json:"latency_distribution"`
}

func (r benchReport) latencyMs(percentile float64) float64 {
	for _, d := range r.LatencyDistribution {
		if d.Percentile == percentile {
			return float64(d.LatencyNs) / 1e6
		}
	}
	return 0
}
