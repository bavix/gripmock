package app

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	internalplugins "github.com/bavix/gripmock/v3/internal/infra/plugins"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
	"github.com/bavix/gripmock/v3/internal/infra/types"
	"github.com/bavix/gripmock/v3/pkg/plugins"
)

const regressiveDelay types.Delay = "{{ duration (sub 3000 (mul 500 (sub .AttemptNumber 1))) }}"

func TestResolveDelay(t *testing.T) {
	t.Parallel()

	engine := template.New(t.Context(), nil)

	tests := []struct {
		name          string
		delay         types.Delay
		attemptNumber int
		expected      time.Duration
		wantCode      codes.Code
	}{
		{name: "static", delay: "1s", expected: time.Second},
		{name: "no delay", expected: 0},
		{name: "first match", delay: regressiveDelay, attemptNumber: 1, expected: 3 * time.Second},
		{name: "third match", delay: regressiveDelay, attemptNumber: 3, expected: 2 * time.Second},
		{name: "floor", delay: regressiveDelay, attemptNumber: 42, expected: 0},
		{name: "empty render", delay: `{{ "" }}`, expected: 0},
		{name: "broken template", delay: "{{ nope }}ms", wantCode: codes.Internal},
		{name: "not a duration", delay: "{{ 5 }} parrots", wantCode: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := newTemplateData(nil, nil, 0, time.Now(), nil, nil, tt.attemptNumber)

			got, err := resolveDelay(engine, tt.delay, data)
			if tt.wantCode != codes.OK {
				require.Equal(t, tt.wantCode, status.Code(err))

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, time.Duration(got))
		})
	}
}

func TestElementDelayOverrides(t *testing.T) {
	t.Parallel()

	require.Equal(t, regressiveDelay, elementDelay(regressiveDelay, stuber.GripMockElement{}))
	require.Equal(t, types.Delay("1m"),
		elementDelay(regressiveDelay, stuber.GripMockElement{HasDelay: true, Delay: "1m"}))
}

func TestValidateStubRejectsBrokenDelayTemplate(t *testing.T) {
	t.Parallel()

	v, err := NewStubValidator()
	require.NoError(t, err)

	server := &RestServer{
		validator:      v,
		templateEngine: template.New(t.Context(), nil),
	}

	stub := &stuber.Stub{
		Service: "svc",
		Method:  "method",
		Input:   stuber.InputData{Equals: map[string]any{"a": "b"}},
		Output:  stuber.Output{Delay: "{{ nope }}", Data: map[string]any{"a": "b"}},
	}

	require.ErrorAs(t, server.validateStub(stub), new(*ValidationError))

	stub.Output.Delay = "banana"
	require.ErrorAs(t, server.validateStub(stub), new(*ValidationError))

	stub.Output.Delay = "{{ duration 10 }}"
	require.NoError(t, server.validateStub(stub))

	stub.Output.Delay = "150ms"
	require.NoError(t, server.validateStub(stub))
}

func TestResolveDelayConcurrent(t *testing.T) {
	t.Parallel()

	engine := template.New(t.Context(), nil)
	data := newTemplateData(nil, nil, 0, time.Now(), nil, nil, 3)

	const workers = 64

	var (
		wg   sync.WaitGroup
		got  [workers]types.Duration
		errs [workers]error
	)

	for i := range workers {
		wg.Go(func() {
			got[i], errs[i] = resolveDelay(engine, regressiveDelay, data)
		})
	}

	wg.Wait()

	for i := range workers {
		require.NoError(t, errs[i])
		require.Equal(t, 2*time.Second, time.Duration(got[i]))
	}
}

func BenchmarkResolveDelay(b *testing.B) {
	engine := template.New(b.Context(), nil)
	data := newTemplateData(nil, nil, 0, time.Now(), nil, nil, 1)

	b.Run("static", func(b *testing.B) {
		for b.Loop() {
			_, _ = resolveDelay(engine, "100ms", data)
		}
	})

	b.Run("template", func(b *testing.B) {
		for b.Loop() {
			_, _ = resolveDelay(engine, regressiveDelay, data)
		}
	})

	b.Run("helper", func(b *testing.B) {
		for b.Loop() {
			_, _ = resolveDelay(engine, `{{ regressive .AttemptNumber "3s" "500ms" }}`, data)
		}
	})
}

func TestPluginExtendsDelayAlgorithms(t *testing.T) {
	t.Parallel()

	registry := internalplugins.NewRegistry()
	internalplugins.RegisterBuiltins(registry)

	registry.AddPlugin(
		plugins.PluginInfo{Name: "curves", Kind: "external"},
		[]plugins.SpecProvider{plugins.Specs(
			plugins.FuncSpec{Name: "linear", Fn: func(attempt, step any) string {
				return fmt.Sprintf("%dms", mustInt(attempt)*mustInt(step))
			}},
			plugins.FuncSpec{
				Name:      "jitter",
				Decorates: "@gripmock/jitter",
				Fn: func(base plugins.Func) plugins.Func {
					return func(ctx context.Context, args ...any) (any, error) {
						return "1s", nil
					}
				},
			},
		)},
	)

	engine := template.New(t.Context(), registry)
	data := newTemplateData(nil, nil, 0, time.Now(), nil, nil, 3)

	linear, err := resolveDelay(engine, `{{ linear .AttemptNumber 40 }}`, data)
	require.NoError(t, err)
	require.Equal(t, 120*time.Millisecond, time.Duration(linear))

	decorated, err := resolveDelay(engine, `{{ jitter "50ms" "60ms" }}`, data)
	require.NoError(t, err)
	require.Equal(t, time.Second, time.Duration(decorated))
}

func mustInt(v any) int {
	switch value := v.(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}
