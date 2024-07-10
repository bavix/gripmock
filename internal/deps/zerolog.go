package deps

import (
	"context"
	"log"
	"time"

	"github.com/rs/zerolog"
)

func (b *Builder) Logger(ctx context.Context) context.Context {
	level, err := zerolog.ParseLevel(b.config.LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	return zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.TimeFormat = time.RFC3339Nano
	})).
		Level(level).
		With().
		Timestamp().
		Logger().
		WithContext(ctx)
}
