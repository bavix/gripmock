package telemetry

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

type Config struct {
	Enabled  bool
	Endpoint string
	Insecure bool
	Version  string
}

// InitMetrics sets up Prometheus metrics via OTel MeterProvider.
// Always initializes regardless of OTEL_ENABLED.
func InitMetrics(ctx context.Context, version string, reg prometheus.Registerer) *Instruments {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("gripmock"),
			semconv.ServiceVersion(version),
		),
	)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("failed to create OTEL resource for metrics")

		return nil
	}

	promExporter, err := promexporter.New(
		promexporter.WithRegisterer(reg),
	)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("failed to create Prometheus exporter")

		return nil
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	meter := otel.Meter("gripmock")

	stubsCounter, _ := meter.Int64UpDownCounter(
		"gripmock.stub.count",
		metric.WithDescription("Current number of registered stubs"),
		metric.WithUnit("{stub}"),
	)

	matchDuration, _ := meter.Float64Histogram(
		"gripmock.stub.match.duration",
		metric.WithDescription("Duration of stub matching"),
		metric.WithUnit("ms"),
	)

	zerolog.Ctx(ctx).Info().Msg("Prometheus metrics endpoint enabled at /metrics")

	return &Instruments{
		StubsCounter:  stubsCounter,
		MatchDuration: matchDuration,
	}
}

// InitTracing sets up OpenTelemetry tracing (OTLP gRPC exporter).
// Only call when OTEL_ENABLED=true.
//
//nolint:funlen
func InitTracing(ctx context.Context, cfg Config) func(context.Context) error {
	if !cfg.Enabled {
		return func(_ context.Context) error { return nil }
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("gripmock"),
			semconv.ServiceVersion(cfg.Version),
		),
	)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("failed to create OTEL resource for tracing")

		return func(_ context.Context) error { return nil }
	}

	dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second) //nolint:mnd
	defer cancel()

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(dialCtx, opts...)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("failed to connect to OTEL collector, tracing will be disabled")

		return func(_ context.Context) error { return nil }
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	zerolog.Ctx(ctx).Info().
		Str("endpoint", cfg.Endpoint).
		Bool("insecure", cfg.Insecure).
		Msg("OpenTelemetry tracer provider initialized")

	return func(shutdownCtx context.Context) error {
		if err := tp.Shutdown(shutdownCtx); err != nil {
			zerolog.Ctx(shutdownCtx).Err(err).Msg("failed to shutdown Otel tracer provider")

			return err
		}

		return nil
	}
}

type Instruments struct {
	StubsCounter  metric.Int64UpDownCounter
	MatchDuration metric.Float64Histogram
}

func MetricsHandler(gatherer prometheus.Gatherer) http.Handler {
	return promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
}
