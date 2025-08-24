package types

// GrpcStatus represents a final status for unary/client or end step for streaming.
type GrpcStatus struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
