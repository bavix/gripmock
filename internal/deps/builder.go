package deps

import (
	"sync"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/infra/errors"
	"github.com/bavix/gripmock/v3/internal/infra/grpcservice"
	"github.com/bavix/gripmock/v3/internal/infra/lifecycle"
	internalplugins "github.com/bavix/gripmock/v3/internal/infra/plugins"
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

	errorFormatter     *app.ErrorFormatter
	errorFormatterOnce sync.Once

	messageConverter     *app.MessageConverter
	messageConverterOnce sync.Once

	stubNotFoundFormatter     *errors.StubNotFoundFormatter
	stubNotFoundFormatterOnce sync.Once

	pluginRegistry *internalplugins.Registry
}

func NewBuilder(opts ...Option) *Builder {
	builder := &Builder{
		ender:          lifecycle.New(nil),
		pluginRegistry: internalplugins.NewRegistry(),
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

// InitTemplatePlugins wires template plugin registry and loads plugins.
// extraPaths are additional plugin paths from CLI.
func (b *Builder) InitTemplatePlugins(extraPaths []string) {
	all := make([]string, 0, len(b.config.TemplatePluginPaths)+len(extraPaths))
	all = append(all, b.config.TemplatePluginPaths...)
	all = append(all, extraPaths...)

	b.pluginRegistry = internalplugins.NewRegistry()
	internalplugins.RegisterBuiltins(b.pluginRegistry)

	loader := internalplugins.NewLoader(all)
	loader.Load(b.pluginRegistry)
}

func (b *Builder) TemplateRegistry() *internalplugins.Registry {
	return b.pluginRegistry
}

func (b *Builder) PluginInfos() []plugins.PluginWithFuncs {
	return b.pluginRegistry.Groups()
}

func (b *Builder) PluginMeta() []plugins.PluginInfo {
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
