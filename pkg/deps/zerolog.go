package deps

import (
	"time"

	"github.com/gripmock/environment"
	"github.com/rs/zerolog"
)

func withTimeFormat(format string) func(w *zerolog.ConsoleWriter) {
	return func(w *zerolog.ConsoleWriter) {
		w.TimeFormat = format
	}
}

func NewZerolog(config environment.Config) (*zerolog.Logger, error) {
	level, err := zerolog.ParseLevel(config.LogLevel)
	if err != nil {
		return nil, err
	}

	logger := zerolog.New(zerolog.NewConsoleWriter(withTimeFormat(time.RFC3339Nano))).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &logger, nil
}
