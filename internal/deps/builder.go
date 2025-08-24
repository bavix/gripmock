package deps

import (
	"sync"

	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/infra/lifecycle"
	"github.com/bavix/gripmock/v3/internal/infra/storage"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type Option func(*Builder)

type Builder struct {
	config config.Config
	ender  *lifecycle.Manager

	budgerigar     *localstuber.Budgerigar
	budgerigarOnce sync.Once

	extender     *storage.Extender
	extenderOnce sync.Once

	serviceManager     *grpcservice.Manager
	serviceManagerOnce sync.Once

	analytics     port.AnalyticsRepository
	analyticsOnce sync.Once

	errorFormatter     *app.ErrorFormatter
	errorFormatterOnce sync.Once

	messageConverter     *app.MessageConverter
	messageConverterOnce sync.Once

	stubNotFoundFormatter     *errors.StubNotFoundFormatter
	stubNotFoundFormatterOnce sync.Once
}

func NewBuilder(opts ...Option) *Builder {
	builder := &Builder{ender: lifecycle.New(nil)}
	for _, opt := range opts {
		opt(builder)
	}

	return builder
}

func WithDefaultConfig() Option {
	cfg, _ := config.New()

	return WithConfig(cfg)
}

func WithConfig(config config.Config) Option {
	return func(builder *Builder) {
		builder.config = cfg
	}
}

func WithEnder(ender *lifecycle.Manager) Option {
	return func(builder *Builder) {
		builder.ender = ender
	}
}

func (b *Builder) ErrorFormatter() *app.ErrorFormatter {
	b.errorFormatterOnce.Do(func() {
		b.errorFormatter = app.NewErrorFormatter()
	})

	return b.errorFormatter
}

func (b *Builder) MessageConverter() *app.MessageConverter {
	b.messageConverterOnce.Do(func() {
		b.messageConverter = app.NewMessageConverter()
	})

	return b.messageConverter
}

func (b *Builder) StubNotFoundFormatter() *errors.StubNotFoundFormatter {
	b.stubNotFoundFormatterOnce.Do(func() {
		b.stubNotFoundFormatter = errors.NewStubNotFoundFormatter()
	})

	return b.stubNotFoundFormatter
}
