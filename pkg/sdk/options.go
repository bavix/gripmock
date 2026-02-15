package sdk

import (
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
	session         string // X-Gripmock-Session for isolation (remote mode)
	listenNetwork   string // "tcp" for real port
	listenAddr      string // ":0" for real port
	healthyTimeout  time.Duration
}

const defaultHealthyTimeout = 10 * time.Second

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

// WithHealthyTimeout sets the timeout for health check.
func WithHealthyTimeout(d time.Duration) Option {
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

// Remote configures the mock to connect to an external gripmock process (phase 0.5).
// grpcAddr is the gRPC server address (e.g. "localhost:4770").
// Stubs are added via REST API; if restURL is empty, it defaults to "http://{host}:4771".
func Remote(grpcAddr string, restURL ...string) Option {
	return func(o *options) {
		o.remoteAddr = grpcAddr
		if len(restURL) > 0 && restURL[0] != "" {
			o.remoteRestURL = restURL[0]
		} else {
			o.remoteRestURL = deriveRestURLFromGrpcAddr(grpcAddr)
		}
	}
}

// WithSession sets the session ID for isolation (remote mode only).
// Stubs and history are partitioned by session; use with t.Parallel() when sharing one gripmock.
func WithSession(sessionID string) Option {
	return func(o *options) {
		o.session = sessionID
	}
}

func deriveRestURLFromGrpcAddr(grpcAddr string) string {
	host, _, _ := splitHostPort(grpcAddr)
	if host == "" {
		host = "127.0.0.1"
	}
	// IPv6 addresses must be wrapped in brackets in URLs
	if strings.Contains(host, ":") {
		return "http://[" + host + "]:4771"
	}
	return "http://" + host + ":4771"
}

func splitHostPort(addr string) (host, port string, err error) {
	// Handle "[::1]:4770" format
	if len(addr) > 0 && addr[0] == '[' {
		end := strings.Index(addr, "]")
		if end == -1 {
			return addr, "", nil
		}
		host = addr[1:end]
		if end+1 < len(addr) && addr[end+1] == ':' {
			port = addr[end+2:]
		}
		return host, port, nil
	}
	idx := strings.LastIndex(addr, ":")
	if idx == -1 {
		return addr, "", nil
	}
	return addr[:idx], addr[idx+1:], nil
}
