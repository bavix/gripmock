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

	timeout, insecure, recordDelay, err := parseProxyQuery(parsed.Query())
	if err != nil {
		return nil, err
	}

	tlsFiles, err := parseUpstreamTLSFiles(parsed.Query(), tlsEnabled)
	if err != nil {
		return nil, err
	}

	return &Source{
		Type:              SourceProxy,
		Raw:               raw,
		ReflectAddress:    parsed.Host,
		ReflectTLS:        tlsEnabled,
		ReflectServerName: parsed.Query().Get("serverName"),
		ReflectBearer:     parsed.Query().Get("bearer"),
		ReflectTimeout:    timeout,
		ReflectInsecure:   insecure,
		ReflectClientCert: tlsFiles.ClientCert,
		ReflectClientKey:  tlsFiles.ClientKey,
		ReflectCAFile:     tlsFiles.CAFile,
		ProxyMode:         proxyMode,
		RecordDelay:       recordDelay,
	}, nil
}

func (h *ProxyHandler) Process(_ context.Context, _ *Source, _ SourceProcessor) error {
	return nil
}

// parseProxyQuery reads the switches a proxy URL may carry.
func parseProxyQuery(query url.Values) (time.Duration, bool, bool, error) {
	timeout := defaultReflectTimeout

	if raw := query.Get("timeout"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return 0, false, false, errors.Wrap(err, "invalid timeout")
		}

		timeout = parsed
	}

	insecure := false

	if raw := query.Get("insecureSkipVerify"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return 0, false, false, errProxySourceInvalidTLS
		}

		insecure = parsed
	}

	recordDelay := false

	if raw := query.Get("recordDelay"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return 0, false, false, errors.Wrap(err, "invalid recordDelay")
		}

		recordDelay = parsed
	}

	return timeout, insecure, recordDelay, nil
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
