package config

import (
	"os"
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
	Endpoint string `env:"EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4317"`
	Enabled  bool   `env:"ENABLED"                envDefault:"false"`
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

	// Gateway serves both ConnectRPC and gRPC-web protocols on a single
	// HTTP port. Content-Type negotiation dispatches to the correct handler.
	//
	// The deprecated CONNECTRPC_PORT / CONNECTRPC_HOST / CONNECTRPC_ADDR /
	// CONNECTRPC_TLS_* env vars are still read as fallbacks (see
	// applyGatewayBackwardCompat).
	Gateway    ServerConfig `envPrefix:"GATEWAY_"`
	GatewayTLS TLSConfig    `envPrefix:"GATEWAY_TLS_"`

	// CORS configuration (applied to REST API).
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envDefault:"*"`
	CORSAllowedMethods []string `env:"CORS_ALLOWED_METHODS" envDefault:"GET,POST,DELETE,PATCH"`

	// OpenTelemetry configuration.
	OTel OTelConfig `envPrefix:"OTEL_"`

	// Files configuration.
	StubWatcherEnabled  bool          `env:"STUB_WATCHER_ENABLED"  envDefault:"true"`
	StubWatcherInterval time.Duration `env:"STUB_WATCHER_INTERVAL" envDefault:"1s"`
	StubWatcherType     watcherType   `env:"STUB_WATCHER_TYPE"     envDefault:"fsnotify"`

	// Proto message-to-map conversion limits.
	MaxNestingDepth uint32 `env:"MAX_NESTING_DEPTH" envDefault:"256"`

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

	applyGatewayBackwardCompat(&cfg)

	if cfg.GRPC.Addr == "" {
		cfg.GRPC.Addr = cfg.GRPC.Host + ":" + cfg.GRPC.Port
	}

	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = cfg.HTTP.Host + ":" + cfg.HTTP.Port
	}

	if cfg.Gateway.Addr == "" {
		cfg.Gateway.Addr = cfg.Gateway.Host + ":" + cfg.Gateway.Port
	}

	return cfg
}

// applyGatewayBackwardCompat maps deprecated CONNECTRPC_* env vars
// to the unified GATEWAY_* config when the latter is not set.
//
// Deprecated: CONNECTRPC_PORT, CONNECTRPC_HOST, CONNECTRPC_ADDR and
// CONNECTRPC_TLS_* are preserved for migration. Use GATEWAY_* instead.
func applyGatewayBackwardCompat(cfg *Config) { //nolint:cyclop
	if cfg.Gateway.Port == "" {
		if p := os.Getenv("CONNECTRPC_PORT"); p != "" {
			cfg.Gateway.Port = p
		}
	}

	if cfg.Gateway.Port == "" {
		cfg.Gateway.Port = "4769"
	}

	if cfg.Gateway.Host == "0.0.0.0" {
		if h := os.Getenv("CONNECTRPC_HOST"); h != "" {
			cfg.Gateway.Host = h
		}
	}

	if cfg.Gateway.Addr == "" {
		if a := os.Getenv("CONNECTRPC_ADDR"); a != "" {
			cfg.Gateway.Addr = a
		}
	}

	if cfg.GatewayTLS.CertFile == "" {
		if v := os.Getenv("CONNECTRPC_TLS_CERT_FILE"); v != "" {
			cfg.GatewayTLS.CertFile = v
		}
	}

	if cfg.GatewayTLS.KeyFile == "" {
		if v := os.Getenv("CONNECTRPC_TLS_KEY_FILE"); v != "" {
			cfg.GatewayTLS.KeyFile = v
		}
	}

	if !cfg.GatewayTLS.ClientAuth {
		if v := os.Getenv("CONNECTRPC_TLS_CLIENT_AUTH"); v != "" {
			cfg.GatewayTLS.ClientAuth = v == "true"
		}
	}

	if cfg.GatewayTLS.CAFile == "" {
		if v := os.Getenv("CONNECTRPC_TLS_CA_FILE"); v != "" {
			cfg.GatewayTLS.CAFile = v
		}
	}

	if cfg.GatewayTLS.MinVersion == "1.2" {
		if v := os.Getenv("CONNECTRPC_TLS_MIN_VERSION"); v != "" {
			cfg.GatewayTLS.MinVersion = v
		}
	}
}
