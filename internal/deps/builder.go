package deps

import (
	"github.com/gripmock/environment"
	"github.com/gripmock/shutdown"
)

type Option func(*Builder)

type Builder struct {
	config environment.Config
	ender  *shutdown.Shutdown
}

func NewBuilder(opts ...Option) *Builder {
	builder := &Builder{ender: shutdown.New(nil)}
	for _, opt := range opts {
		opt(builder)
	}

	return builder
}

func WithDefaultConfig() Option {
	config, _ := environment.New()

	return WithConfig(config)
}

func WithConfig(config environment.Config) Option {
	return func(builder *Builder) {
		builder.config = config
	}
}

func WithEnder(ender *shutdown.Shutdown) Option {
	return func(builder *Builder) {
		builder.ender = ender
	}
}
