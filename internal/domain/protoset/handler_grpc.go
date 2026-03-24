package protoset

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

const defaultReflectTimeout = 5 * time.Second

var (
	errGRPCSourceMissingHost = errors.New("gRPC source must include host:port")
	errGRPCSourceHasPath     = errors.New("gRPC source must not include path")
	errUnsupportedScheme     = errors.New("unsupported gRPC scheme")
)

type GRPCHandler struct{}

func (h *GRPCHandler) CanHandle(raw string) bool {
	return strings.HasPrefix(raw, "grpc://") || strings.HasPrefix(raw, "grpcs://")
}

func (h *GRPCHandler) Parse(raw string) (*Source, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, errors.Wrap(err, "invalid gRPC source")
	}

	if parsed.Scheme != "grpc" && parsed.Scheme != "grpcs" {
		return nil, errors.Wrap(errUnsupportedScheme, parsed.Scheme)
	}

	if parsed.Host == "" {
		return nil, errGRPCSourceMissingHost
	}

	if parsed.Path != "" && parsed.Path != "/" {
		return nil, errGRPCSourceHasPath
	}

	timeout := defaultReflectTimeout
	if rawTimeout := parsed.Query().Get("timeout"); rawTimeout != "" {
		timeout, err = time.ParseDuration(rawTimeout)
		if err != nil {
			return nil, errors.Wrap(err, "invalid timeout")
		}
	}

	return &Source{
		Type:              SourceReflect,
		Raw:               raw,
		ReflectAddress:    parsed.Host,
		ReflectTLS:        parsed.Scheme == "grpcs",
		ReflectServerName: parsed.Query().Get("serverName"),
		ReflectBearer:     parsed.Query().Get("bearer"),
		ReflectTimeout:    timeout,
	}, nil
}

func (h *GRPCHandler) Process(ctx context.Context, source *Source, processor SourceProcessor) error {
	reflectProcessor, ok := processor.(interface {
		ProcessReflect(ctx context.Context, source *Source) error
	})

	if !ok {
		return nil
	}

	return reflectProcessor.ProcessReflect(ctx, source)
}
