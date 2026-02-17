package deps

import (
	"context"
	"slices"
	"sync"

	"github.com/go-playground/validator/v10"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
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

	historyStore     *history.MemoryStore
	historyStoreOnce sync.Once

	stubValidator     *validator.Validate
	stubValidatorOnce sync.Once

	descriptorRegistry     *descriptors.Registry
	descriptorRegistryOnce sync.Once

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
		builder.pluginPaths = slices.Clone(paths)
	}
}

func (b *Builder) LoadPlugins(ctx context.Context) {
	b.pluginOnce.Do(func() {
		reg := internalplugins.NewRegistry()
		allPaths := slices.Concat(b.config.TemplatePluginPaths, b.pluginPaths)
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

// HistoryStore returns the shared in-memory history store when HistoryEnabled.
// Returns nil when history is disabled.
func (b *Builder) HistoryStore() *history.MemoryStore {
	if !b.config.HistoryEnabled {
		return nil
	}

	b.historyStoreOnce.Do(func() {
		opts := []history.MemoryStoreOption{
			history.WithMessageMaxBytes(b.config.HistoryMessageMaxBytes),
		}
		if len(b.config.HistoryRedactKeys) > 0 {
			opts = append(opts, history.WithRedactKeys(b.config.HistoryRedactKeys))
		}

		b.historyStore = history.NewMemoryStore(b.config.HistoryLimit.Int64(), opts...)
	})

	return b.historyStore
}

// StubValidator returns the shared stub validator (created once per Builder).
func (b *Builder) StubValidator() *validator.Validate {
	b.stubValidatorOnce.Do(func() {
		b.stubValidator = app.NewStubValidator()
	})

	return b.stubValidator
}

func (b *Builder) DescriptorRegistry() *descriptors.Registry {
	b.descriptorRegistryOnce.Do(func() {
		b.descriptorRegistry = descriptors.NewRegistry()
	})

	return b.descriptorRegistry
}
