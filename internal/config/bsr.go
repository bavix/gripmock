package config

import (
	"net/url"
	"time"
)

type BSRProfile struct {
	BaseURL *url.URL      `env:"BASE_URL"`
	Token   string        `env:"TOKEN"`
	Timeout time.Duration `env:"TIMEOUT"  envDefault:"5s"`
}

type BSRConfig struct {
	Buf  BSRProfile `envPrefix:"BUF_"`
	Self BSRProfile `envPrefix:"SELF_"`
}
