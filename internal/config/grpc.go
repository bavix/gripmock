package config

type GRPC struct {
	Network string `envconfig:"GRPC_NETWORK" default:"tcp"`
	Host    string `envconfig:"GRPC_HOST" default:"0.0.0.0"`
	Port    string `envconfig:"GRPC_PORT" default:"4770"`
}
