package config

type App struct {
	Name     string `envconfig:"APP_NAME" default:"gripmock"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
}
