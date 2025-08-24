package deps

import (
	"sync"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/app/port"
	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/infra/errors"
	"github.com/bavix/gripmock/v3/internal/infra/grpcservice"
	localshutdown "github.com/bavix/gripmock/v3/internal/infra/shutdown"
	"github.com/bavix/gripmock/v3/internal/infra/storage"
	localstuber "github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type Option func(*Builder)

type Builder struct {
	config config.AppConfig
	ender  *localshutdown.Shutdown

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
	builder := &Builder{ender: localshutdown.New(nil)}
	for _, opt := range opts {
		opt(builder)
	}

	return builder
}

func WithDefaultConfig() Option { return WithConfig(config.Load()) }

func WithConfig(cfg config.AppConfig) Option {
	return func(builder *Builder) {
		builder.config = cfg
	}
}

func WithEnder(ender *localshutdown.Shutdown) Option {
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
