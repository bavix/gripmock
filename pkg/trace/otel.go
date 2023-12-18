package trace

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func InitTracer(ctx context.Context, appName string, otlpTrace OTLPTrace) error {
	if !otlpTrace.UseTrace() {
		return nil
	}

	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(
			net.JoinHostPort(otlpTrace.Host, otlpTrace.Port)),
	}
	if !otlpTrace.TLS {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return errors.Wrap(err, "build trace exporter")
	}

	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.TraceIDRatioBased(otlpTrace.SampleRatio)),
		trace.WithBatcher(exporter),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(appName),
			),
		),
	)

	//tp.Shutdown

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return nil
}
