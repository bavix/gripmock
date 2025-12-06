package config

import (
	"time"

	env "github.com/caarlos0/env/v11"
)

type watcherType string

const (
	WatcherFSNotify watcherType = "fsnotify"
	WatcherTimer    watcherType = "timer"
)

type Config struct {
	// Application logging level.
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
	// Deprecated.
	// Strict mode for checking the name of services and methods.
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
}

func New() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
