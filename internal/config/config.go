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

// GRPCLimitsConfig holds gRPC message size limits and keepalive settings.
type GRPCLimitsConfig struct {
	MaxRecvMsgSize ByteSize `env:"MAX_RECV_MSG_SIZE" envDefault:"4M"`
	MaxSendMsgSize ByteSize `env:"MAX_SEND_MSG_SIZE" envDefault:"0"`

	KeepaliveTime              time.Duration `env:"KEEPALIVE_TIME"                envDefault:"30s"`
	KeepaliveTimeout           time.Duration `env:"KEEPALIVE_TIMEOUT"             envDefault:"10s"`
	KeepaliveMaxConnectionIdle time.Duration `env:"KEEPALIVE_MAX_CONNECTION_IDLE" envDefault:"5m"`
	KeepaliveMaxConnectionAge  time.Duration `env:"KEEPALIVE_MAX_CONNECTION_AGE"  envDefault:"30m"`
}

// OTelConfig holds OpenTelemetry configuration.
type OTelConfig struct {
	Endpoint string `env:"EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4317"`
	Enabled  bool   `env:"ENABLED"                envDefault:"false"`
	Insecure bool   `env:"EXPORTER_OTLP_INSECURE" envDefault:"true"`
}

// Config holds environment-derived configuration values.
type Config struct {
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	GRPCNetwork string           `env:"GRPC_NETWORK"    envDefault:"tcp"`
	GRPC        ServerConfig     `envPrefix:"GRPC_"`
	GRPCTLS     TLSConfig        `envPrefix:"GRPC_TLS_"`
	GRPCLimits  GRPCLimitsConfig `envPrefix:"GRPC_"`

	HTTP    ServerConfig `envPrefix:"HTTP_"`
	HTTPTLS TLSConfig    `envPrefix:"HTTP_TLS_"`

	Gateway    ServerConfig `envPrefix:"GATEWAY_"`
	GatewayTLS TLSConfig    `envPrefix:"GATEWAY_TLS_"`

	ConnectRequireProtocolVersion bool `env:"CONNECT_REQUIRE_PROTOCOL_VERSION" envDefault:"false"`

	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envDefault:"*"`
	CORSAllowedMethods []string `env:"CORS_ALLOWED_METHODS" envDefault:"GET,POST,DELETE,PATCH"`

	OTel OTelConfig `envPrefix:"OTEL_"`

	StubWatcherEnabled  bool          `env:"STUB_WATCHER_ENABLED"  envDefault:"true"`
	StubWatcherInterval time.Duration `env:"STUB_WATCHER_INTERVAL" envDefault:"1s"`
	StubWatcherType     watcherType   `env:"STUB_WATCHER_TYPE"     envDefault:"fsnotify"`

	MaxNestingDepth uint32 `env:"MAX_NESTING_DEPTH" envDefault:"256"`

	HistoryEnabled         bool     `env:"HISTORY_ENABLED"           envDefault:"true"`
	HistoryLimit           ByteSize `env:"HISTORY_LIMIT"             envDefault:"64M"`
	HistoryMessageMaxBytes int64    `env:"HISTORY_MESSAGE_MAX_BYTES" envDefault:"262144"`
	HistoryRedactKeys      []string `env:"HISTORY_REDACT_KEYS"`

	SessionGCInterval time.Duration `env:"SESSION_GC_INTERVAL" envDefault:"30s"`
	SessionGCTTL      time.Duration `env:"SESSION_GC_TTL"      envDefault:"60s"`

	TemplatePluginPaths []string `env:"TEMPLATE_PLUGIN_PATHS"`

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

	if cfg.Gateway.Port == "" {
		cfg.Gateway.Port = "4769"
	}

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
