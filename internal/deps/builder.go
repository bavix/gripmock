package deps

import (
	"context"
	"log"
	"slices"
	"sync"
	"sync/atomic"

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
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
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

	// eagerly initialized
	stubValidator      *validator.Validate
	errorFormatter     *app.ErrorFormatter
	descriptorRegistry *descriptors.Registry

	// lazy stateful
	budgerigar     *stuber.Budgerigar
	budgerigarOnce sync.Once

	historyStore     *history.MemoryStore
	historyStoreOnce sync.Once

	remoteClient     protosetdom.RemoteClient
	remoteClientOnce sync.Once

	extender     *storage.Extender
	extenderOnce sync.Once

	pluginPaths    []string
	pluginRegistry *internalplugins.Registry
	pluginOnce     sync.Once

	gateway     *app.MultiProtocolGateway
	proxyRoutes atomic.Pointer[proxyroutes.Registry]
}

func newStubValidator() *validator.Validate {
	v, err := app.NewStubValidator()
	if err != nil {
		log.Printf("[gripmock] stub validator init failed: %v; using fallback", err)

		return validator.New()
	}

	return v
}

func NewBuilder(opts ...Option) *Builder {
	b := &Builder{
		ender:   lifecycle.New(nil),
		promReg: prometheus.NewRegistry(),
	}
	for _, opt := range opts {
		opt(b)
	}

	b.stubValidator = newStubValidator()
	b.errorFormatter = app.NewErrorFormatter()
	b.descriptorRegistry = descriptors.NewRegistry()

	b.promReg.MustRegister(collectors.NewGoCollector())
	b.promReg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	return b
}

func WithDefaultConfig() Option {
	cfg := config.Load()

	return WithConfig(cfg)
}

func WithConfig(cfg config.Config) Option {
	return func(b *Builder) {
		b.config = cfg
	}
}

func WithPlugins(paths []string) Option {
	return func(b *Builder) {
		b.pluginPaths = slices.Clone(paths)
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

func (b *Builder) InitTelemetry(ctx context.Context) {
	b.otelInstr = telemetry.InitMetrics(ctx, build.Version, b.promReg)

	if b.config.OTel.Enabled {
		cfg := telemetry.Config{
			Enabled:  b.config.OTel.Enabled,
			Endpoint: b.config.OTel.Endpoint,
			Insecure: b.config.OTel.Insecure,
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

func (b *Builder) StubValidator() *validator.Validate {
	return b.stubValidator
}

func (b *Builder) ErrorFormatter() *app.ErrorFormatter {
	return b.errorFormatter
}

func (b *Builder) DescriptorRegistry() *descriptors.Registry {
	return b.descriptorRegistry
}

//nolint:ireturn
func (b *Builder) RemoteClient() protosetdom.RemoteClient {
	b.remoteClientOnce.Do(func() {
		b.remoteClient = sourceclient.NewRouter(
			bufclient.NewRouter(b.config.BSR),
			reflectclient.NewClient(),
		)
	})

	return b.remoteClient
}

func (b *Builder) SetProxyRoutes(r *proxyroutes.Registry) {
	b.proxyRoutes.Store(r)
}

func (b *Builder) ProxyRoutes() *proxyroutes.Registry {
	return b.proxyRoutes.Load()
}

func (b *Builder) SetGateway(g *app.MultiProtocolGateway) {
	b.gateway = g
}

func (b *Builder) Gateway() *app.MultiProtocolGateway {
	return b.gateway
}
