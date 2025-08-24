package types

import "time"

// StubAnalytics holds per-stub metrics for usage and performance.
type StubAnalytics struct {
	StubID                      string     `json:"stubId"`
	UsedCount                   int64      `json:"usedCount"`
	FirstUsedAt                 *time.Time `json:"firstUsedAt,omitempty"`
	LastUsedAt                  *time.Time `json:"lastUsedAt,omitempty"`
	TotalSendMessages           int64      `json:"totalSendMessages"`
	TotalDataResponses          int64      `json:"totalDataResponses"`
	StreamEndEvents             int64      `json:"streamEndEvents"`
	ErrorCount                  int64      `json:"errorCount"`
	TotalDurationMilliseconds   int64      `json:"totalDurationMilliseconds"`
	AverageDurationMilliseconds float64    `json:"averageDurationMilliseconds"`
}
