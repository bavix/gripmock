package deps

import (
	"context"
	"sync"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/infra/errors"
	"github.com/bavix/gripmock/v3/internal/infra/grpcservice"
	"github.com/bavix/gripmock/v3/internal/infra/lifecycle"
	internalplugins "github.com/bavix/gripmock/v3/internal/infra/plugins"
	"github.com/bavix/gripmock/v3/internal/infra/runtime"
	"github.com/bavix/gripmock/v3/internal/infra/storage"
	"github.com/bavix/gripmock/v3/internal/infra/store/memory"
	localstuber "github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/pkg/plugins"
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

	analytics     *memory.InMemoryAnalytics
	analyticsOnce sync.Once

	pluginPaths []string

	errorFormatter     *app.ErrorFormatter
	errorFormatterOnce sync.Once

	messageConverter     *app.MessageConverter
	messageConverterOnce sync.Once

	stubNotFoundFormatter     *errors.StubNotFoundFormatter
	stubNotFoundFormatterOnce sync.Once

	pluginRegistry *internalplugins.Registry

	pluginOnce sync.Once
}

func NewBuilder(opts ...Option) *Builder {
	builder := &Builder{
		ender:       lifecycle.New(nil),
		pluginPaths: nil,
	}
	for _, opt := range opts {
		opt(builder)
	}

	return builder
}

func WithDefaultConfig() Option {
	return WithConfig(config.Load())
}

func WithConfig(cfg config.Config) Option {
	return func(builder *Builder) {
		builder.config = cfg
	}
}

// WithPlugins sets additional plugin paths (e.g. from CLI flags).
// Paths are copied to avoid external mutation.
func WithPlugins(paths []string) Option {
	return func(builder *Builder) {
		if len(paths) == 0 {
			return
		}

		builder.pluginPaths = append(make([]string, 0, len(paths)), paths...)
	}
}

func (b *Builder) LoadPlugins(ctx context.Context) {
	b.pluginOnce.Do(func() {
		b.loadPlugins(ctx)
	})
}

func (b *Builder) loadPlugins(ctx context.Context) {
	all := make([]string, 0, len(b.config.TemplatePluginPaths)+len(b.pluginPaths))
	all = append(all, b.config.TemplatePluginPaths...)
	all = append(all, b.pluginPaths...)

	b.pluginRegistry = internalplugins.NewRegistry(internalplugins.WithContext(ctx))
	internalplugins.RegisterBuiltins(b.pluginRegistry)

	loader := internalplugins.NewLoader(all)
	loader.Load(ctx, b.pluginRegistry)

	// Wire hook consumers
	localstuber.SetMatcherHooks(b.pluginRegistry)
	runtime.SetHookRegistry(b.pluginRegistry)
}

func (b *Builder) TemplateRegistry(ctx context.Context) *internalplugins.Registry {
	b.LoadPlugins(ctx)
	return b.pluginRegistry
}

func (b *Builder) PluginInfos(ctx context.Context) []plugins.PluginWithFuncs {
	b.LoadPlugins(ctx)
	return b.pluginRegistry.Groups()
}

func (b *Builder) PluginMeta(ctx context.Context) []plugins.PluginInfo {
	b.LoadPlugins(ctx)
	return b.pluginRegistry.Plugins()
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
