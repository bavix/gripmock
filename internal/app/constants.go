package app

// Constants for gRPC server configuration.
const (
	// ServiceReflection is the name of the gRPC reflection service.
	ServiceReflection = "grpc.reflection.v1.ServerReflection"

	// DefaultTimeout is the default timeout for operations.
	DefaultTimeout = 30

	// HealthServiceName is the name of the health check service.
	HealthServiceName = "gripmock"
)

// ExcludedHeaders contains headers that should be excluded from stub matching.
var ExcludedHeaders = []string{
	":authority",
	"content-type",
	"grpc-accept-encoding",
	"user-agent",
	"accept-encoding",
}

// Error messages.
const (
	ErrMsgFailedToSendResponse     = "failed to send response"
	ErrMsgFailedToReceiveMessage   = "failed to receive message"
	ErrMsgFailedToSetHeaders       = "failed to set headers"
	ErrMsgFailedToConvertResponse  = "failed to convert response to dynamic message"
	ErrMsgFailedToProcessMessage   = "failed to process bidirectional message"
	ErrMsgFailedToInitializeStream = "failed to initialize bidirectional streaming session"
	ErrMsgFailedToBuildDescriptors = "failed to build descriptors"
	ErrMsgFailedToFindStub         = "failed to find stub"
	ErrMsgFailedToMarshalData      = "failed to marshal expect data"
)

// Logging constants.
const (
	LogFieldService     = "service"
	LogFieldMethod      = "method"
	LogFieldPeerAddress = "peer.address"
	LogFieldProtocol    = "protocol"
	LogFieldTimeMs      = "grpc.time_ms"
	LogFieldCode        = "grpc.code"
	LogFieldComponent   = "grpc.component"
	LogFieldMetadata    = "grpc.metadata"
	LogFieldRequest     = "grpc.request.content"
	LogFieldResponse    = "grpc.response.content"
	LogFieldMethodType  = "grpc.method_type"
)
