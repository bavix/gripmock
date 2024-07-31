package deps

import (
	"context"
	"log"

	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/pkg/zlogger"
)

func (b *Builder) Logger(ctx context.Context) context.Context {
	level, err := zerolog.ParseLevel(b.config.LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	return zlogger.Logger(ctx, level)
}
