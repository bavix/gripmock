package types

// StreamStep describes a unit of streaming work: send, delay or end.
type StreamStep struct {
	Send  map[string]any `json:"send,omitempty"`
	Delay string         `json:"delay,omitempty"`
	End   *GrpcStatus    `json:"end,omitempty"`
}

// StreamStepStrict represents a strictly typed stream step.
type StreamStepStrict struct {
	Send  *SendStep `json:"send,omitempty"`
	Delay *Delay    `json:"delay,omitempty"`
	End   *EndStep  `json:"end,omitempty"`
}

// SendStep represents a structured send operation.
type SendStep struct {
	Data    map[string]any    `json:"data"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Delay represents a structured delay operation.
type Delay struct {
	Duration string `json:"duration"` // e.g., "100ms", "2s"
}

// EndStep represents a structured end operation.
type EndStep struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// SendEach describes a shortcut for sending multiple messages derived from a template.
type SendEach struct {
	Items   string         `json:"items"`
	As      string         `json:"as"`
	Message map[string]any `json:"message"`
	Delay   string         `json:"delay,omitempty"`
}

// StreamRule is a reactive rule for server/bidi methods.
type StreamRule struct {
	Match    *Matcher     `json:"match,omitempty"`
	Stream   []StreamStep `json:"stream,omitempty"`
	SendEach *SendEach    `json:"sendEach,omitempty"`
	Delay    string       `json:"delay,omitempty"`
}
