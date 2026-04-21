package config

import (
	"time"

	env "github.com/caarlos0/env/v11"

	infraTypes "github.com/bavix/gripmock/v3/internal/infra/types"
)

// ByteSize is kept for backward compatibility; defined in internal/infra/types.
type ByteSize = infraTypes.ByteSize

// Config holds environment-derived configuration values.
type Config struct {
	// Application logging level.
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	// GRPC server configuration.
	GRPCNetwork string `env:"GRPC_NETWORK" envDefault:"tcp"`
	GRPCHost    string `env:"GRPC_HOST"    envDefault:"0.0.0.0"`
	GRPCPort    string `env:"GRPC_PORT"    envDefault:"4770"`
	GRPCAddr    string `env:",expand"      envDefault:"$GRPC_HOST:$GRPC_PORT"`

	// HTTP server configuration.
	HTTPHost string `env:"HTTP_HOST" envDefault:"0.0.0.0"`
	HTTPPort string `env:"HTTP_PORT" envDefault:"4771"`
	HTTPAddr string `env:",expand"   envDefault:"$HTTP_HOST:$HTTP_PORT"`

	// Files configuration.
	StubWatcherEnabled  bool          `env:"STUB_WATCHER_ENABLED"  envDefault:"true"`
	StubWatcherInterval time.Duration `env:"STUB_WATCHER_INTERVAL" envDefault:"1s"`
	StubWatcherType     watcherType   `env:"STUB_WATCHER_TYPE"     envDefault:"fsnotify"`

	// History configuration.
	HistoryEnabled         bool     `env:"HISTORY_ENABLED"           envDefault:"true"`
	HistoryLimit           ByteSize `env:"HISTORY_LIMIT"             envDefault:"64M"`
	HistoryMessageMaxBytes int64    `env:"HISTORY_MESSAGE_MAX_BYTES" envDefault:"262144"`
	HistoryRedactKeys      []string `env:"HISTORY_REDACT_KEYS"`

	// Session GC configuration.
	SessionGCInterval time.Duration `env:"SESSION_GC_INTERVAL" envDefault:"30s"`
	SessionGCTTL      time.Duration `env:"SESSION_GC_TTL"      envDefault:"60s"`

	// Plugins configuration.
	TemplatePluginPaths []string `env:"TEMPLATE_PLUGIN_PATHS"`

	// gRPC TLS configuration.
	GRPCTLSCertFile   string `env:"GRPC_TLS_CERT_FILE"`
	GRPCTLSKeyFile    string `env:"GRPC_TLS_KEY_FILE"`
	GRPCTLSClientAuth bool   `env:"GRPC_TLS_CLIENT_AUTH" envDefault:"false"`
	GRPCTLSCAFile     string `env:"GRPC_TLS_CA_FILE"`
	GRPCTLSMinVersion string `env:"GRPC_TLS_MIN_VERSION" envDefault:"1.2"`

	// HTTP TLS configuration.
	HTTPTLSCertFile   string `env:"HTTP_TLS_CERT_FILE"`
	HTTPTLSKeyFile    string `env:"HTTP_TLS_KEY_FILE"`
	HTTPTLSClientAuth bool   `env:"HTTP_TLS_CLIENT_AUTH" envDefault:"false"`
	HTTPTLSCAFile     string `env:"HTTP_TLS_CA_FILE"`

	// OpenTelemetry configuration.
	OtelEnabled  bool   `env:"OTEL_ENABLED"                envDefault:"false"`
	OtelEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4317"`
	OtelInsecure bool   `env:"OTEL_EXPORTER_OTLP_INSECURE" envDefault:"true"`

	// Buf Schema Registry configuration.
	BSR BSRConfig `envPrefix:"BSR_"`
}

// Load returns configuration from environment with sensible defaults.
func Load() Config {
	var cfg Config

	_ = env.Parse(&cfg)

	return cfg
}
