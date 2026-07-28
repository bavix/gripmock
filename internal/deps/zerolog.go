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

	return newLogger(ctx, level)
}

func newLogger(ctx context.Context, level zerolog.Level) context.Context {
	logger := zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.TimeFormat = time.RFC3339Nano
	})).
		Level(level).
		With().
		Timestamp()

	return logger.Logger().WithContext(ctx)
}
