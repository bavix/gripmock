package dependencies

import (
	"context"
	"net"

	"github.com/gripmock/environment"
	"github.com/gripmock/shutdown"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func tracer(
	ctx context.Context,
	config environment.Config,
	shutdown *shutdown.Shutdown,
	appName string,
) (*trace.TracerProvider, error) {
	if config.OtlpRatio == 0 || config.OtlpHost == "" || config.OtlpPort == "" {
		return nil, nil
	}

	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(net.JoinHostPort(config.OtlpHost, config.OtlpPort)),
	}
	if !config.OtlpTLS {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return nil, errors.Wrap(err, "build trace exporter")
	}

	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.TraceIDRatioBased(config.OtlpRatio)),
		trace.WithSyncer(exporter),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(appName),
			),
		),
	)

	shutdown.Add(tp.Shutdown)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}
