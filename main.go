package main

import (
	"context"
	"github.com/bavix/gripmock/cmd"
	"github.com/bavix/gripmock/internal/config"
	"github.com/bavix/gripmock/pkg/trace"
	"github.com/bavix/gripmock/pkg/utils"
	"github.com/rs/zerolog"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	conf, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	logLevel, err := zerolog.ParseLevel(conf.App.LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	logger := utils.NewLogger(logLevel)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	ctx = logger.WithContext(ctx)

	if err := trace.InitTracer(ctx, "gripmock", conf.OTLPTrace); err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("connect to tracer")
	}

	cmd.Execute(ctx, conf)
}
