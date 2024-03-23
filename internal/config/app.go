package config

type App struct {
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
	Simpler  bool   `envconfig:"SERVICE_SIMPLER" default:"true"`
}
