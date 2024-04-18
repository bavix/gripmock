package dependencies

import (
	"github.com/gripmock/shutdown"
	"github.com/rs/zerolog"
)

type Logger struct {
	logger *zerolog.Logger
}

func (l Logger) Err(err error) {
	l.logger.Err(err).Send()
}

func newLog(logger *zerolog.Logger) shutdown.Logger {
	return &Logger{logger: logger}
}
