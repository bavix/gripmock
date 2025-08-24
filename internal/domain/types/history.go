package types

import "time"

// HistoryRecord is a single RPC session log.
type HistoryRecord struct {
	ID                   string              `json:"id"`
	Timestamp            time.Time           `json:"timestamp"`
	Service              string              `json:"service"`
	Method               string              `json:"method"`
	RPCType              string              `json:"rpcType"`
	StubID               string              `json:"stubId"`
	FormatVersion        string              `json:"formatVersion"`
	RuleKind             string              `json:"ruleKind"`
	RequestHeaders       map[string][]string `json:"requestHeaders,omitempty"`
	ResponseHeaders      map[string][]string `json:"responseHeaders,omitempty"`
	Requests             []map[string]any    `json:"requests,omitempty"`
	Responses            []map[string]any    `json:"responses,omitempty"`
	EndStatus            map[string]any      `json:"endStatus,omitempty"`
	DurationMilliseconds int64               `json:"durationMilliseconds"`
	BytesIn              int64               `json:"bytesIn"`
	BytesOut             int64               `json:"bytesOut"`
	IsTruncated          bool                `json:"isTruncated"`
}
