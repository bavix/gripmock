package types

// DataRule is a final single response rule for unary/client streaming.
type DataRule struct {
	Data   map[string]any `json:"data"`
	Status *GrpcStatus    `json:"status,omitempty"`
}

// DataResponse represents a structured data response.
type DataResponse struct {
	Content map[string]any    `json:"content"`
	Headers map[string]string `json:"headers,omitempty"`
}

// SequenceItem defines either a match-driven stream part or a final data rule.
type SequenceItem struct {
	Match    *Matcher     `json:"match,omitempty"`
	Stream   []StreamStep `json:"stream,omitempty"`
	SendEach *SendEach    `json:"sendEach,omitempty"`
	// Data endpoints the sequence for unary/client.
	Data   map[string]any `json:"data,omitempty"`
	Status *GrpcStatus    `json:"status,omitempty"`
}

// SequenceRule is a strict ordered set of inputs and responses.
type SequenceRule struct {
	Sequence []SequenceItem `json:"sequence"`
}

// OutputRule is a union of StreamRule, DataRule, and SequenceRule.
// Use only one variant in practice. This struct is a convenience holder.
type OutputRule struct {
	Stream   *StreamRule   `json:"-"`
	Data     *DataRule     `json:"-"`
	Sequence *SequenceRule `json:"-"`
}

// Output represents a modern output with better performance.
type Output struct {
	Data   map[string]any `json:"data,omitempty"`
	Stream []StreamStep   `json:"stream,omitempty"`
	Delay  string         `json:"delay,omitempty"`
	Status *GrpcStatus    `json:"status,omitempty"`
}

// OutputStrict represents a strictly typed output with no raw maps.
type OutputStrict struct {
	Data   *DataResponse      `json:"data,omitempty"`
	Stream []StreamStepStrict `json:"stream,omitempty"`
	Delay  *Delay             `json:"delay,omitempty"`
	Status *GrpcStatus        `json:"status,omitempty"`
}
