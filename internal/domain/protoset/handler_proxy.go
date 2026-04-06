package protoset

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

var errProxySourceInvalidTLS = errors.New("proxy source insecureSkipVerify must be true or false")

const (
	proxySchemeParts = 2
	transportGRPC    = "grpc"
	transportGRPCS   = "grpcs"
)

type ProxyHandler struct{}

func (h *ProxyHandler) CanHandle(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}

	_, _, parseErr := parseProxyScheme(parsed.Scheme)

	return parseErr == nil
}

//nolint:cyclop
func (h *ProxyHandler) Parse(raw string) (*Source, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, errors.Wrap(err, "invalid proxy source")
	}

	proxyMode, tlsEnabled, err := parseProxyScheme(parsed.Scheme)
	if err != nil {
		return nil, err
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

	insecure := false
	if rawInsecure := parsed.Query().Get("insecureSkipVerify"); rawInsecure != "" {
		insecure, err = strconv.ParseBool(rawInsecure)
		if err != nil {
			return nil, errProxySourceInvalidTLS
		}
	}

	recordDelay := false
	if rawRecordDelay := parsed.Query().Get("recordDelay"); rawRecordDelay != "" {
		recordDelay, err = strconv.ParseBool(rawRecordDelay)
		if err != nil {
			return nil, errors.Wrap(err, "invalid recordDelay")
		}
	}

	return &Source{
		Type:              SourceReflect,
		Raw:               raw,
		ReflectAddress:    parsed.Host,
		ReflectTLS:        tlsEnabled,
		ReflectServerName: parsed.Query().Get("serverName"),
		ReflectBearer:     parsed.Query().Get("bearer"),
		ReflectTimeout:    timeout,
		ReflectInsecure:   insecure,
		ProxyMode:         proxyMode,
		RecordDelay:       recordDelay,
	}, nil
}

func (h *ProxyHandler) Process(ctx context.Context, source *Source, processor SourceProcessor) error {
	reflectProcessor, ok := processor.(interface {
		ProcessReflect(ctx context.Context, source *Source) error
	})

	if !ok {
		return nil
	}

	return reflectProcessor.ProcessReflect(ctx, source)
}

func parseProxyScheme(scheme string) (string, bool, error) {
	parts := strings.Split(scheme, "+")
	if len(parts) != proxySchemeParts {
		return "", false, errors.Wrap(errUnsupportedScheme, scheme)
	}

	transport := parts[0]
	mode := parts[1]

	var tlsEnabled bool

	switch transport {
	case transportGRPC:
		tlsEnabled = false
	case transportGRPCS:
		tlsEnabled = true
	default:
		return "", false, errors.Wrap(errUnsupportedScheme, scheme)
	}

	switch mode {
	case "proxy":
		return "proxy", tlsEnabled, nil
	case "replay":
		return "replay", tlsEnabled, nil
	case "capture":
		return "capture", tlsEnabled, nil
	default:
		return "", false, errors.Wrap(errUnsupportedScheme, scheme)
	}
}
