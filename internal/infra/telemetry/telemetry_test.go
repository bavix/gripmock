package telemetry_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/bavix/gripmock/v3/internal/infra/telemetry"
)

func newRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

func TestInitMetricsCreatesInstruments(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	reg := newRegistry()

	instr := telemetry.InitMetrics(ctx, "test", reg)
	require.NotNil(t, instr)
	require.NotNil(t, instr.StubsCounter)
	require.NotNil(t, instr.MatchDuration)
}

func TestInitMetricsReturnsPrometheusOutput(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	reg := newRegistry()

	instr := telemetry.InitMetrics(ctx, "test", reg)
	require.NotNil(t, instr)

	instr.StubsCounter.Add(ctx, 1)

	handler := telemetry.MetricsHandler(reg)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, string(body), "gripmock_stub_count")
}

func TestStubsCounterRecordsMetrics(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	reg := newRegistry()

	instr := telemetry.InitMetrics(ctx, "test", reg)
	require.NotNil(t, instr)

	instr.StubsCounter.Add(ctx, 5, metric.WithAttributes(attribute.String("service", "test.Service")))
	instr.StubsCounter.Add(ctx, -2, metric.WithAttributes(attribute.String("service", "test.Service")))

	handler := telemetry.MetricsHandler(reg)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body, err := io.ReadAll(rec.Result().Body)
	require.NoError(t, err)

	bodyStr := string(body)
	require.Contains(t, bodyStr, "gripmock_stub_count")
	require.Contains(t, bodyStr, "service=\"test.Service\"")
}

func TestMatchDurationRecordsMetrics(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	reg := newRegistry()

	instr := telemetry.InitMetrics(ctx, "test", reg)
	require.NotNil(t, instr)

	instr.MatchDuration.Record(ctx, 12.5, metric.WithAttributes(attribute.String("method", "SayHello")))
	instr.MatchDuration.Record(ctx, 3.2, metric.WithAttributes(attribute.String("method", "SayHello")))

	handler := telemetry.MetricsHandler(reg)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body, err := io.ReadAll(rec.Result().Body)
	require.NoError(t, err)

	require.Contains(t, string(body), "gripmock_stub_match_duration")
}

func TestTracingDisabled(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cfg := telemetry.Config{Enabled: false}
	shutdown := telemetry.InitTracing(ctx, cfg)
	require.NotNil(t, shutdown)

	err := shutdown(ctx)
	require.NoError(t, err)
}

func TestTracingUnreachableCollector(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cfg := telemetry.Config{
		Enabled:  true,
		Endpoint: "localhost:19999",
		Insecure: true,
		Version:  "test",
	}

	start := time.Now()
	shutdown := telemetry.InitTracing(ctx, cfg)
	elapsed := time.Since(start)

	require.Less(t, elapsed, 5*time.Second)

	if shutdown != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		_ = shutdown(shutdownCtx)
	}
}

func TestTracingInvalidEndpoint(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cfg := telemetry.Config{
		Enabled:  true,
		Endpoint: "not-a-valid-endpoint:99999",
		Insecure: true,
		Version:  "test",
	}

	start := time.Now()
	shutdown := telemetry.InitTracing(ctx, cfg)
	elapsed := time.Since(start)

	require.Less(t, elapsed, 5*time.Second)
	require.NotNil(t, shutdown)

	shutdownCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_ = shutdown(shutdownCtx)
}

func TestTracerProviderSetAfterInit(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cfg := telemetry.Config{
		Enabled:  true,
		Endpoint: "localhost:19999",
		Insecure: true,
		Version:  "test",
	}

	_ = telemetry.InitTracing(ctx, cfg)

	tracer := otel.Tracer("test")
	require.NotNil(t, tracer)

	_, span := tracer.Start(ctx, "test-span")
	require.NotNil(t, span)
	span.End()
}

func TestTextMapPropagatorSetAfterInit(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cfg := telemetry.Config{
		Enabled:  true,
		Endpoint: "localhost:19999",
		Insecure: true,
		Version:  "test",
	}

	_ = telemetry.InitTracing(ctx, cfg)

	propagator := otel.GetTextMapPropagator()
	require.NotNil(t, propagator)
}
