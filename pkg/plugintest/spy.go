package plugintest

import (
	"context"

	"github.com/bavix/gripmock/v3/pkg/plugins"
)

// Invocation captures a single call to a spied function, including arguments and
// results, so tests can assert call order and data.
type Invocation struct {
	Args   []any
	Result any
	Err    error
}

// Spy records calls to a wrapped plugin Func without altering behavior, enabling
// verification of decorators or dynamic template helpers.
type Spy struct {
	Calls []Invocation
	fn    plugins.Func
}

// NewSpy wraps a Func and stores every invocation in memory for later
// inspection.
func NewSpy(fn plugins.Func) *Spy {
	return &Spy{fn: fn}
}

// Func returns the wrapped function that records invocations before delegating.
func (s *Spy) Func() plugins.Func {
	if s.fn == nil {
		return nil
	}

	return func(ctx context.Context, args ...any) (any, error) {
		res, err := s.fn(ctx, args...)
		s.Calls = append(s.Calls, Invocation{Args: append([]any{}, args...), Result: res, Err: err})

		return res, err
	}
}

// Decorator returns a decorator that records calls to the base function,
// convenient for testing decorator chaining.
func (s *Spy) Decorator() func(plugins.Func) plugins.Func {
	return func(base plugins.Func) plugins.Func {
		s.fn = base
		return s.Func()
	}
}

// Reset clears recorded calls, useful when multiple scenarios share the same spy.
func (s *Spy) Reset() {
	s.Calls = nil
}
