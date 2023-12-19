package utils

import (
	"time"

	"github.com/rs/zerolog"
)

func NewLogger(level zerolog.Level) zerolog.Logger {
	return zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.TimeFormat = time.RFC3339Nano
	})).
		Level(level).
		With().
		Timestamp().
		Logger()
}
