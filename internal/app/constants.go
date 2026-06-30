package app

const (
	HealthServiceFullName = "grpc.health.v1.Health"
	HealthServiceName     = "gripmock"
)

const (
	LogFieldService     = "service"
	LogFieldMethod      = "method"
	LogFieldPeerAddress = "peer.address"
	LogFieldComponent   = "grpc.component"
)

//nolint:gochecknoglobals
var ExcludedHeaders = []string{
	":authority",
	"content-type",
	"grpc-accept-encoding",
	"user-agent",
	"accept-encoding",
}
