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
	// Create a new zerolog logger
	logger := zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		// Set the time format of the console writer to RFC3339Nano
		w.TimeFormat = time.RFC3339Nano
	})).
		Level(level). // Set the log level of the logger
		With().       // Start a new log entry with no fields
		Timestamp()   // Add a timestamp to the log entry

	// Wrap the logger in the context and return it
	return logger.Logger().WithContext(ctx)
}
