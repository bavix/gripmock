package zlogger

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// Logger creates a new zerolog logger with the given log level and wraps it in a context.
// The logger will use a console writer with the time format set to RFC3339Nano.
// The logger will also embed the timestamp in all log entries.
//
// Parameters:
// - ctx: The context to wrap the logger in.
// - level: The log level to use for the logger.
//
// Returns:
// - context.Context: The context with the logger embedded in it.
func Logger(ctx context.Context, level zerolog.Level) context.Context {
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
