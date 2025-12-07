package config

import (
	"encoding"
	"fmt"
	"strconv"
	"strings"
	"time"

	env "github.com/caarlos0/env/v11"
)

const (
	kibibyte = int64(1024)
	mebibyte = kibibyte * 1024
	gibibyte = mebibyte * 1024
)

// ByteSize is an env-decodable size with K|M|G suffix support (decimal kilobytes/mebibytes/gibibytes).
// Examples: "128K", "64M", "1G", or plain integer bytes like "262144".
type ByteSize struct {
	Bytes int64
}

// UnmarshalText implements encoding.TextUnmarshaler for ByteSize.
func (b *ByteSize) UnmarshalText(text []byte) error {
	// Support both integer bytes and values with K/M/G suffixes
	raw := strings.TrimSpace(strings.ToUpper(string(text)))
	if raw == "" {
		b.Bytes = 0

		return nil
	}

	mult := int64(1)

	switch {
	case strings.HasSuffix(raw, "K"):
		mult = kibibyte
		raw = strings.TrimSuffix(raw, "K")
	case strings.HasSuffix(raw, "M"):
		mult = mebibyte
		raw = strings.TrimSuffix(raw, "M")
	case strings.HasSuffix(raw, "G"):
		mult = gibibyte
		raw = strings.TrimSuffix(raw, "G")
	}

	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid byte size: %w", err)
	}

	b.Bytes = n * mult

	return nil
}

// Int64 returns the value in bytes.
func (b *ByteSize) Int64() int64 { return b.Bytes }

var _ encoding.TextUnmarshaler = (*ByteSize)(nil)

// Config holds environment-derived configuration values.
type Config struct {
	// Application logging level.
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	// Deprecated: Strict mode for checking the name of services and methods.
	StrictMethodTitle bool `env:"STRICT_METHOD_TITLE" envDefault:"false"`

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

	// Plugins configuration.
	TemplatePluginPaths []string `env:"TEMPLATE_PLUGIN_PATHS"`
}

// Load returns configuration from environment with sensible defaults.
func Load() Config {
	var cfg Config

	_ = env.Parse(&cfg)

	return cfg
}
