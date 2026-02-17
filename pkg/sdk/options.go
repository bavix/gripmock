package sdk

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func (o *options) appendDescriptorFiles(files []*descriptorpb.FileDescriptorProto) {
	seen := make(map[string]bool, len(o.descriptorFiles))
	for _, f := range o.descriptorFiles {
		seen[f.GetName()] = true
	}
	for _, f := range files {
		if !seen[f.GetName()] {
			seen[f.GetName()] = true
			o.descriptorFiles = append(o.descriptorFiles, f)
		}
	}
}

type options struct {
	descriptorFiles []*descriptorpb.FileDescriptorProto // accumulated via append
	mockFromAddr    string
	remoteAddr      string // gRPC address for remote mode
	remoteRestURL   string // REST base URL (e.g. "http://localhost:4771") for remote mode
	httpClient      *http.Client
	session         string // X-Gripmock-Session for isolation (remote mode)
	sessionTTL      time.Duration
	grpcTimeout     time.Duration
	listenNetwork   string // "tcp" for real port
	listenAddr      string // ":0" for real port
	healthyTimeout  time.Duration
}

const defaultHealthyTimeout = 10 * time.Second
const defaultSessionTTL = 60 * time.Second

// Option configures Run behavior.
type Option func(*options)

// WithDescriptors appends files from the FileDescriptorSet to the mock server (skips duplicates by name).
func WithDescriptors(fds *descriptorpb.FileDescriptorSet) Option {
	return func(o *options) {
		o.appendDescriptorFiles(fds.GetFile())
	}
}

// WithFileDescriptor appends a generated protoreflect.FileDescriptor (e.g. helloworld.File_service_proto).
func WithFileDescriptor(fd protoreflect.FileDescriptor) Option {
	return func(o *options) {
		fdp := protodesc.ToFileDescriptorProto(fd)
		o.appendDescriptorFiles([]*descriptorpb.FileDescriptorProto{fdp})
	}
}

// WithListenAddr sets network and address for real port listening.
func WithListenAddr(network, addr string) Option {
	return func(o *options) {
		o.listenNetwork = network
		o.listenAddr = addr
	}
}

// WithHealthCheckTimeout sets timeout for readiness/health check.
func WithHealthCheckTimeout(d time.Duration) Option {
	return func(o *options) {
		o.healthyTimeout = d
	}
}

// MockFrom configures the mock to use gRPC reflection from the given address (phase 0.5).
func MockFrom(addr string) Option {
	return func(o *options) {
		o.mockFromAddr = addr
	}
}

// WithRemote configures the mock to connect to an external gripmock process.
// grpcAddr is the gRPC server address (e.g. "localhost:4770").
// restURL is the REST base URL used for management operations (e.g. "http://localhost:4771").
func WithRemote(grpcAddr string, restURL string) Option {
	return func(o *options) {
		o.remoteAddr = normalizeRemoteAddr(grpcAddr)
		o.remoteRestURL = normalizeRemoteRestURL(restURL)
	}
}

// WithHTTPClient overrides the HTTP client used by WithRemote mode for REST API calls.
// If not set, SDK uses a default client with 10s timeout.
func WithHTTPClient(client *http.Client) Option {
	return func(o *options) {
		o.httpClient = client
	}
}

// WithSession sets the session ID for isolation (remote mode only).
// Stubs and history are partitioned by session; use with t.Parallel() when sharing one gripmock.
func WithSession(sessionID string) Option {
	return func(o *options) {
		o.session = strings.TrimSpace(sessionID)
	}
}

// WithSessionTTL configures automatic cleanup time for session-scoped remote resources.
// Only applies to WithRemote mode.
func WithSessionTTL(d time.Duration) Option {
	return func(o *options) {
		o.sessionTTL = d
	}
}

// WithGRPCTimeout sets default per-RPC timeout for remote gRPC calls.
// Applied only when call context has no deadline.
func WithGRPCTimeout(d time.Duration) Option {
	return func(o *options) {
		o.grpcTimeout = d
	}
}

func deriveRestURLFromGrpcAddr(grpcAddr string) string {
	host := extractHost(normalizeRemoteAddr(grpcAddr))
	if host == "" {
		host = "127.0.0.1"
	}

	return (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, "4771")}).String()
}

func normalizeRemoteAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.Contains(addr, "://") {
		if parsed, err := url.Parse(addr); err == nil && parsed.Host != "" {
			addr = parsed.Host
		}
	}

	addr = strings.TrimSuffix(addr, "/")

	return addr
}

func normalizeRemoteRestURL(restURL string) string {
	restURL = strings.TrimSpace(restURL)
	if restURL == "" {
		return ""
	}

	if !strings.Contains(restURL, "://") {
		restURL = "http://" + restURL
	}

	parsed, err := url.Parse(restURL)
	if err != nil {
		return strings.TrimRight(restURL, "/")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")

	return parsed.String()
}

func extractHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host
	}

	if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") {
		return strings.Trim(addr, "[]")
	}

	return strings.TrimSuffix(addr, "/")
}
