package types

// MethodInfo represents information about a gRPC method including its streaming type.
type MethodInfo struct {
	Service        string `json:"service"`
	Method         string `json:"method"`
	IsUnary        bool   `json:"isUnary"`
	IsClientStream bool   `json:"isClientStream"`
	IsServerStream bool   `json:"isServerStream"`
	IsBidiStream   bool   `json:"isBidiStream"`
}

// MethodRegistry stores information about all registered gRPC methods.
type MethodRegistry struct {
	methods map[string]MethodInfo // key: "service/method"
}

// NewMethodRegistry creates a new MethodRegistry.
func NewMethodRegistry() *MethodRegistry {
	return &MethodRegistry{
		methods: make(map[string]MethodInfo),
	}
}

// RegisterMethod adds a method to the registry.
func (r *MethodRegistry) RegisterMethod(service, method string, isClientStream, isServerStream bool) {
	key := service + "/" + method
	r.methods[key] = MethodInfo{
		Service:        service,
		Method:         method,
		IsUnary:        !isClientStream && !isServerStream,
		IsClientStream: isClientStream,
		IsServerStream: isServerStream,
		IsBidiStream:   isClientStream && isServerStream,
	}
}

// GetMethodInfo retrieves method information from the registry.
func (r *MethodRegistry) GetMethodInfo(service, method string) (MethodInfo, bool) {
	key := service + "/" + method
	info, exists := r.methods[key]

	return info, exists
}

// GetAllMethods returns all registered methods.
func (r *MethodRegistry) GetAllMethods() []MethodInfo {
	methods := make([]MethodInfo, 0, len(r.methods))
	for _, method := range r.methods {
		methods = append(methods, method)
	}

	return methods
}

// IsUnary checks if a method is unary.
func (r *MethodRegistry) IsUnary(service, method string) bool {
	info, exists := r.GetMethodInfo(service, method)

	return exists && info.IsUnary
}

// IsClientStream checks if a method is client streaming.
func (r *MethodRegistry) IsClientStream(service, method string) bool {
	info, exists := r.GetMethodInfo(service, method)

	return exists && info.IsClientStream
}

// IsServerStream checks if a method is server streaming.
func (r *MethodRegistry) IsServerStream(service, method string) bool {
	info, exists := r.GetMethodInfo(service, method)

	return exists && info.IsServerStream
}

// IsBidiStream checks if a method is bidirectional streaming.
func (r *MethodRegistry) IsBidiStream(service, method string) bool {
	info, exists := r.GetMethodInfo(service, method)

	return exists && info.IsBidiStream
}

// StubType represents the type of gRPC stub.
type StubType string

const (
	// StubTypeUnary represents unary RPC stubs (single request, single response).
	StubTypeUnary StubType = "unary"
	// StubTypeClientStream represents client streaming RPC stubs (multiple requests, single response).
	StubTypeClientStream StubType = "client_stream"
	// StubTypeServerStream represents server streaming RPC stubs (single request, multiple responses).
	StubTypeServerStream StubType = "server_stream"
	// StubTypeBidirectional represents bidirectional streaming RPC stubs (multiple requests, multiple responses).
	StubTypeBidirectional StubType = "bidirectional"
)

// Stub is the top-level stub entity.
type Stub struct {
	ID               string            `json:"id,omitempty"`
	Service          string            `json:"service"`
	Method           string            `json:"method"`
	Priority         int               `json:"priority,omitempty"`
	Headers          *Matcher          `json:"headers,omitempty"`
	Inputs           []Matcher         `json:"inputs,omitempty"`
	OutputsRaw       []map[string]any  `json:"outputs"`
	ResponseHeaders  map[string]string `json:"responseHeaders,omitempty"`
	ResponseTrailers map[string]string `json:"responseTrailers,omitempty"`
	Times            int               `json:"times,omitempty"`

	// StubType is a computed field that is determined by infrastructure layer
	// based on gRPC method characteristics. It should never be serialized to/from JSON
	// or stored in files, as it's derived information.
	StubType StubType `json:"-"`
}

// StubModern represents a modern stub with improved performance.
type StubModern struct {
	Service          string            `json:"service"`
	Method           string            `json:"method"`
	Priority         int               `json:"priority,omitempty"`
	Times            int               `json:"times,omitempty"`
	Inputs           []Matcher         `json:"inputs,omitempty"`
	Headers          *Matcher          `json:"headers,omitempty"`
	Outputs          []Output          `json:"outputs"`
	ResponseHeaders  map[string]string `json:"responseHeaders,omitempty"`
	ResponseTrailers map[string]string `json:"responseTrailers,omitempty"`
	ID               string            `json:"id,omitempty"`
}
