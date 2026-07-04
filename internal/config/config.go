package config

import (
	"time"

	env "github.com/caarlos0/env/v11"

	infraTypes "github.com/bavix/gripmock/v3/internal/infra/types"
)

// ByteSize is kept for backward compatibility; defined in internal/infra/types.
type ByteSize = infraTypes.ByteSize

// TLSConfig holds TLS settings shared across servers.
type TLSConfig struct {
	CertFile   string `env:"CERT_FILE"`
	KeyFile    string `env:"KEY_FILE"`
	ClientAuth bool   `env:"CLIENT_AUTH" envDefault:"false"`
	CAFile     string `env:"CA_FILE"`
	MinVersion string `env:"MIN_VERSION" envDefault:"1.2"`
}

// ServerConfig holds address configuration for a server.
type ServerConfig struct {
	Host string `env:"HOST" envDefault:"0.0.0.0"`
	Port string `env:"PORT"`
	Addr string `env:"ADDR"`
}

// OTelConfig holds OpenTelemetry configuration.
type OTelConfig struct {
	Enabled  bool   `env:"ENABLED"                envDefault:"false"`
	Endpoint string `env:"EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4317"`
	Insecure bool   `env:"EXPORTER_OTLP_INSECURE" envDefault:"true"`
}

// Config holds environment-derived configuration values.
type Config struct {
	// Application logging level.
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	// GRPC server configuration.
	GRPCNetwork string       `env:"GRPC_NETWORK"    envDefault:"tcp"`
	GRPC        ServerConfig `envPrefix:"GRPC_"`
	GRPCTLS     TLSConfig    `envPrefix:"GRPC_TLS_"`

	// HTTP server configuration.
	HTTP    ServerConfig `envPrefix:"HTTP_"`
	HTTPTLS TLSConfig    `envPrefix:"HTTP_TLS_"`

	// ConnectRPC server configuration.
	Connect    ServerConfig `envPrefix:"CONNECTRPC_"`
	ConnectTLS TLSConfig    `envPrefix:"CONNECTRPC_TLS_"`

	// OpenTelemetry configuration.
	OTel OTelConfig `envPrefix:"OTEL_"`

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

	// Buf Schema Registry configuration.
	BSR BSRConfig `envPrefix:"BSR_"`
}

// Load returns configuration from environment with sensible defaults.
func Load() Config {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		panic("config: failed to parse environment: " + err.Error())
	}

	if cfg.GRPC.Port == "" {
		cfg.GRPC.Port = "4770"
	}

	if cfg.HTTP.Port == "" {
		cfg.HTTP.Port = "4771"
	}

	if cfg.Connect.Port == "" {
		cfg.Connect.Port = "4769"
	}

	if cfg.GRPC.Addr == "" {
		cfg.GRPC.Addr = cfg.GRPC.Host + ":" + cfg.GRPC.Port
	}

	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = cfg.HTTP.Host + ":" + cfg.HTTP.Port
	}

	if cfg.Connect.Addr == "" {
		cfg.Connect.Addr = cfg.Connect.Host + ":" + cfg.Connect.Port
	}

	return cfg
}
