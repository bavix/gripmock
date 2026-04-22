package deps

import (
	"context"
	"slices"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
	bufclient "github.com/bavix/gripmock/v3/internal/infra/bufclient"
	"github.com/bavix/gripmock/v3/internal/infra/build"
	"github.com/bavix/gripmock/v3/internal/infra/lifecycle"
	internalplugins "github.com/bavix/gripmock/v3/internal/infra/plugins"
	reflectclient "github.com/bavix/gripmock/v3/internal/infra/reflectclient"
	sourceclient "github.com/bavix/gripmock/v3/internal/infra/sourceclient"
	"github.com/bavix/gripmock/v3/internal/infra/storage"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/telemetry"
	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

type Option func(*Builder)

type Builder struct {
	config config.Config
	ender  *lifecycle.Manager

	promReg   *prometheus.Registry
	otelInstr *telemetry.Instruments

	budgerigar     *stuber.Budgerigar
	budgerigarOnce sync.Once

	historyStore     *history.MemoryStore
	historyStoreOnce sync.Once

	stubValidator     *validator.Validate
	stubValidatorOnce sync.Once

	descriptorRegistry     *descriptors.Registry
	descriptorRegistryOnce sync.Once

	bufClient     protosetdom.BSRClient
	bufClientOnce sync.Once

	reflectClient     protosetdom.RemoteClient
	reflectClientOnce sync.Once

	remoteClient     protosetdom.RemoteClient
	remoteClientOnce sync.Once

	extender     *storage.Extender
	extenderOnce sync.Once

	pluginPaths    []string
	pluginRegistry *internalplugins.Registry
	pluginOnce     sync.Once

	sessionGCOnce sync.Once
}

func NewBuilder(opts ...Option) *Builder {
	builder := &Builder{
		ender:   lifecycle.New(nil),
		promReg: prometheus.NewRegistry(),
	}
	for _, opt := range opts {
		opt(builder)
	}

	builder.promReg.MustRegister(collectors.NewGoCollector())
	builder.promReg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

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

// WithPlugins sets additional plugin paths (e.g. from CLI flags).
func WithPlugins(paths []string) Option {
	return func(builder *Builder) {
		builder.pluginPaths = slices.Clone(paths)
	}
}

func (b *Builder) LoadPlugins(ctx context.Context) {
	b.pluginOnce.Do(func() {
		reg := internalplugins.NewRegistry()
		internalplugins.RegisterBuiltins(reg)

		allPaths := slices.Concat(b.config.TemplatePluginPaths, b.pluginPaths)
		loader := internalplugins.NewLoader(allPaths)
		loader.Load(ctx, reg)
		b.pluginRegistry = reg
	})
}

// InitTelemetry initializes OpenTelemetry with fail-safe startup.
func (b *Builder) InitTelemetry(ctx context.Context) {
	b.otelInstr = telemetry.InitMetrics(ctx, build.Version, b.promReg)

	if b.config.OtelEnabled {
		cfg := telemetry.Config{
			Enabled:  b.config.OtelEnabled,
			Endpoint: b.config.OtelEndpoint,
			Insecure: b.config.OtelInsecure,
			Version:  build.Version,
		}
		shutdownFn := telemetry.InitTracing(ctx, cfg)
		b.ender.Add(shutdownFn)
	}
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
		var err error

		b.stubValidator, err = app.NewStubValidator()
		if err != nil {
			panic("stub validator init: " + err.Error())
		}
	})

	return b.stubValidator
}

func (b *Builder) DescriptorRegistry() *descriptors.Registry {
	b.descriptorRegistryOnce.Do(func() {
		b.descriptorRegistry = descriptors.NewRegistry()
	})

	return b.descriptorRegistry
}

//nolint:ireturn
func (b *Builder) BufClient() protosetdom.BSRClient {
	b.bufClientOnce.Do(func() {
		b.bufClient = bufclient.NewRouter(b.config.BSR)
	})

	return b.bufClient
}

//nolint:ireturn
func (b *Builder) ReflectClient() protosetdom.RemoteClient {
	b.reflectClientOnce.Do(func() {
		b.reflectClient = reflectclient.NewClient()
	})

	return b.reflectClient
}

//nolint:ireturn
func (b *Builder) RemoteClient() protosetdom.RemoteClient {
	b.remoteClientOnce.Do(func() {
		b.remoteClient = sourceclient.NewRouter(b.BufClient(), b.ReflectClient())
	})

	return b.remoteClient
}
