package app

import (
	"bytes"
	"context"
	"crypto/tls"
	"runtime"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	protoloc "github.com/bavix/gripmock/v3/internal/domain/proto"
	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

// excludedHeaders contains headers that should be excluded from stub matching.
// Map for O(1) lookup in hot path.
//
//nolint:gochecknoglobals
var excludedHeaders = map[string]struct{}{
	":authority":           {},
	"content-type":         {},
	"grpc-accept-encoding": {},
	"user-agent":           {},
	"accept-encoding":      {},
}

const (
	sessionHeaderKey = "x-gripmock-session" // gRPC metadata keys are lowercase
	unknownValue     = "unknown"

	// High-load gRPC server tuning.
	keepaliveMaxIdle     = 5 * time.Minute
	keepaliveMaxAge      = 30 * time.Minute
	keepaliveMaxAgeGrace = 5 * time.Second
	keepaliveTime        = 30 * time.Second
	keepaliveTimeout     = 10 * time.Second
	keepaliveMinTime     = 10 * time.Second
	maxConcurrentStreams = 100
	maxLoggingStreamMsgs = 32
	maxHistoryStreamMsgs = 100
	minStreamWorkers     = 4
)

const (
	jsonBufferInitialCap            = 4096
	bidiRecordingStreamInitCap      = 16
	bidiRecordingStreamResponsesCap = 16
)

//nolint:gochecknoglobals
var (
	runtimeNumStreamWorkers = max(runtime.NumCPU(), minStreamWorkers)
	jsonBufferPool          = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 0, jsonBufferInitialCap))
		},
	}
)

const serviceReflection = "grpc.reflection.v1.ServerReflection"

type GRPCServer struct {
	network         string
	address         string
	params          *protoloc.Arguments
	budgerigar      *stuber.Budgerigar
	healthState     stuber.Aliveness
	waiter          Extender
	recorder        history.Recorder
	descriptors     *descriptors.Registry
	remoteClient    protosetdom.RemoteClient
	tlsConfig       *tls.Config
	proxies         *proxyroutes.Registry
	otelEnabled     bool
	maxNestingDepth uint32
	validator       *validator.Validate
	errorFormatter  *ErrorFormatter
}

type grpcMocker struct {
	budgerigar         *stuber.Budgerigar
	templateEngine     *template.Engine
	errorFormatter     *ErrorFormatter
	recorder           history.Recorder
	descriptorResolver protodesc.Resolver
	proxies            *proxyroutes.Registry
	validator          *validator.Validate

	inputDesc  protoreflect.MessageDescriptor
	outputDesc protoreflect.MessageDescriptor

	fullServiceName string
	serviceName     string
	methodName      string
	fullMethod      string

	serverStream bool
	clientStream bool

	strictServiceMatch bool

	maxNestingDepth uint32
}

const defaultConvertDepth = 256

// newOutputMessage converts stub data (a map, scalar, or nil) into a dynamicpb.Message
// for the response descriptor. Map payloads have numeric values converted to
// json.Number so int64 fields survive the JSON round trip; scalar payloads (e.g. a
// well-known type whose JSON encoding is a primitive: string for Timestamp, number
// for wrappers, object for Struct) are JSON-marshaled as-is and fed to protojson,
// which natively understands the canonical JSON form for every WKT.

const clientMessagesInitCap = 16

func NewGRPCServer(
	network, address string,
	params *protoloc.Arguments,
	budgerigar *stuber.Budgerigar,
	waiter Extender,
	recorder history.Recorder,
	descriptorRegistry *descriptors.Registry,
	tlsConfig *tls.Config,
	remoteClient protosetdom.RemoteClient,
	otelEnabled bool,
	maxNestingDepth uint32,
	stubValidator *validator.Validate,
	errorFormatter *ErrorFormatter,
) *GRPCServer {
	registry := descriptorRegistry
	if registry == nil {
		registry = descriptors.NewRegistry()
	}

	v := stubValidator
	if v == nil {
		v = mustNewStubValidator()
	}

	e := errorFormatter
	if e == nil {
		e = NewErrorFormatter()
	}

	var healthState stuber.Aliveness
	if budgerigar != nil {
		healthState = budgerigar
	}

	return &GRPCServer{
		network:         network,
		address:         address,
		params:          params,
		budgerigar:      budgerigar,
		healthState:     healthState,
		waiter:          waiter,
		recorder:        recorder,
		descriptors:     registry,
		remoteClient:    remoteClient,
		tlsConfig:       tlsConfig,
		otelEnabled:     otelEnabled,
		maxNestingDepth: maxNestingDepth,
		validator:       v,
		errorFormatter:  e,
	}
}

func (s *GRPCServer) Proxies() *proxyroutes.Registry {
	return s.proxies
}

//nolint:cyclop
func (s *GRPCServer) Build(ctx context.Context) (*grpc.Server, error) {
	var err error

	imports := []string{}
	protoPaths := []string{}
	sources := []string{}

	var descriptors []*descriptorpb.FileDescriptorSet

	if s.params != nil {
		imports = s.params.Imports()
		protoPaths = s.params.ProtoPath()
		sources = s.params.Sources()
	}

	if s.params != nil && s.params.HasProxyBindings() {
		descriptors, s.proxies, err = s.buildProxiesWithBindings(ctx, imports)
	} else {
		descriptors, s.proxies, err = s.buildProxiesFromSources(ctx, imports, protoPaths, sources)
	}

	if err != nil {
		return nil, err
	}

	if s.proxies != nil {
		s.startProxyCleanup(ctx)
		s.registerProxyDescriptors(ctx)
	}

	if len(protoPaths) > 0 {
		nonProxyDescriptors, err := protosetdom.Build(ctx, imports, protoPaths, s.remoteClient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build descriptors")
		}

		descriptors = append(descriptors, nonProxyDescriptors...)
	}

	if s.proxies != nil {
		proxyFiles := s.proxies.Files()
		if len(proxyFiles) > 0 {
			descriptors = append(descriptors, proxyFiles...)
		}
	}

	if s.waiter != nil {
		s.waiter.Wait(ctx)
	}

	server := s.createServer(ctx)
	s.setupHealthCheck(server, nil)
	s.registerServices(ctx, server, descriptors, nil)
	s.markServerReady(ctx)

	return server, nil
}

// BuildFromDescriptorSet creates a gRPC server from a pre-built FileDescriptorSet.
// Used by the SDK for embedded mode. Does not use GlobalFiles.
// If recorder is non-nil, all gRPC calls are recorded for History/Verify.

// methodFilesLister abstracts a descriptor registry that supports iteration
// over file descriptors. Implemented by *protoregistry.Files and
// *descriptors.Registry.
