package deps

import (
	"context"
	"sync"

	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/infra/lifecycle"
	internalplugins "github.com/bavix/gripmock/v3/internal/infra/plugins"
	"github.com/bavix/gripmock/v3/internal/infra/storage"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

type Option func(*Builder)

type Builder struct {
	config config.Config
	ender  *lifecycle.Manager

	budgerigar     *stuber.Budgerigar
	budgerigarOnce sync.Once

	extender     *storage.Extender
	extenderOnce sync.Once

	pluginPaths    []string
	pluginRegistry *internalplugins.Registry
	pluginOnce     sync.Once
}

func NewBuilder(opts ...Option) *Builder {
	builder := &Builder{ender: lifecycle.New(nil)}
	for _, opt := range opts {
		opt(builder)
	}

	return builder
}

func WithDefaultConfig() Option {
	cfg := config.Load()

	return WithConfig(cfg)
}

func WithConfig(config config.Config) Option {
	return func(builder *Builder) {
		builder.config = config
	}
}

func WithEnder(ender *lifecycle.Manager) Option {
	return func(builder *Builder) {
		builder.ender = ender
	}
}

// WithPlugins sets additional plugin paths (e.g. from CLI flags).
func WithPlugins(paths []string) Option {
	return func(builder *Builder) {
		builder.pluginPaths = append(make([]string, 0, len(paths)), paths...)
	}
}

func (b *Builder) LoadPlugins(ctx context.Context) {
	b.pluginOnce.Do(func() {
		reg := internalplugins.NewRegistry()

		allPaths := make([]string, 0, len(b.config.TemplatePluginPaths)+len(b.pluginPaths))
		allPaths = append(allPaths, b.config.TemplatePluginPaths...)
		allPaths = append(allPaths, b.pluginPaths...)
		loader := internalplugins.NewLoader(allPaths)
		loader.Load(ctx, reg)
		b.pluginRegistry = reg
	})
}

func (b *Builder) PluginInfos(ctx context.Context) []pkgplugins.PluginWithFuncs {
	b.LoadPlugins(ctx)

	if b.pluginRegistry == nil {
		return nil
	}

	return b.pluginRegistry.Groups(ctx)
}
