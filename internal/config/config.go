package config

import (
	"net"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	App  App
	GRPC GRPC
	HTTP HTTP
}

func Load() (Config, error) {
	cnf := Config{} //nolint:exhaustruct

	if err := godotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return cnf, errors.Wrap(err, "read .env file")
	}

	if err := envconfig.Process("", &cnf); err != nil {
		return cnf, errors.Wrap(err, "read environment")
	}

	return cnf, nil
}

func (c *Config) GRPCAddr() string {
	return net.JoinHostPort(c.GRPC.Host, c.GRPC.Port)
}

func (c *Config) HTTPAddr() string {
	return net.JoinHostPort(c.HTTP.Host, c.HTTP.Port)
}
