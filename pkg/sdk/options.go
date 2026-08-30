package sdk

import (
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
		if f == nil {
			continue
		}

		seen[f.GetName()] = true
	}

	for _, f := range files {
		if f == nil {
			continue
		}

		if !seen[f.GetName()] {
			seen[f.GetName()] = true
			o.descriptorFiles = append(o.descriptorFiles, f)
		}
	}
}

type options struct {
	descriptorFiles []*descriptorpb.FileDescriptorProto
	protoPaths      []string
	batchMode       bool
	mockFromAddr    string
	remoteAddr      string
	remoteRestURL   string
	httpClient      *http.Client
	session         string
	sessionTTL      time.Duration
	grpcTimeout     time.Duration
	listenNetwork   string
	listenAddr      string
	healthyTimeout  time.Duration
}

const (
	defaultHealthyTimeout = 10 * time.Second
	defaultSessionTTL     = 60 * time.Second
)

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

// WithListenAddr sets network and address for real port listening. The default is
// a loopback port (127.0.0.1:0), which is also the address the server reports.
func WithListenAddr(network, addr string) Option {
	return func(o *options) {
		o.listenNetwork = network
		o.listenAddr = addr
	}
}

func WithHealthCheckTimeout(d time.Duration) Option {
	return func(o *options) {
		o.healthyTimeout = d
	}
}

// WithRemote configures the mock to connect to an external gripmock process.
func WithRemote(grpcAddr string, restURL string) Option {
	return func(o *options) {
		o.remoteAddr = normalizeRemoteAddr(grpcAddr)
		o.remoteRestURL = normalizeRemoteRestURL(restURL)
	}
}

// WithHTTPClient overrides the HTTP client used by WithRemote mode for REST API calls.
func WithHTTPClient(client *http.Client) Option {
	return func(o *options) {
		o.httpClient = client
	}
}

// WithSession sets the session ID for isolation.
func WithSession(sessionID string) Option {
	return func(o *options) {
		o.session = strings.TrimSpace(sessionID)
	}
}

// WithSessionTTL configures automatic cleanup time for session-scoped remote resources.
func WithSessionTTL(d time.Duration) Option {
	return func(o *options) {
		o.sessionTTL = d
	}
}

// WithGRPCTimeout sets default per-RPC timeout for remote gRPC calls.
func WithGRPCTimeout(d time.Duration) Option {
	return func(o *options) {
		o.grpcTimeout = d
	}
}

func WithProtoFiles(paths ...string) Option {
	return func(o *options) {
		o.protoPaths = append(o.protoPaths, paths...)
	}
}

// WithBatch enables batch mode for remote stub registration.
func WithBatch() Option {
	return func(o *options) {
		o.batchMode = true
	}
}

// WithReflection resolves descriptors from a running gRPC server via reflection.
func WithReflection(addr string) Option {
	return func(o *options) {
		o.mockFromAddr = addr
	}
}

func normalizeRemoteAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.Contains(addr, "://") {
		parsed, err := url.Parse(addr)
		if err == nil && parsed.Host != "" {
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
