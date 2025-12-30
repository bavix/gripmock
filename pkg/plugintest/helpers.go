package plugintest

import (
	"context"

	"github.com/stretchr/testify/require"
)

type tb interface {
	require.TestingT
	Helper()
}

// MustLookupFunc fetches a function by name or fails the test immediately,
// avoiding repetitive error handling in plugin specs tests.
func MustLookupFunc(t tb, reg Registry, name string) Func {
	t.Helper()

	fn, ok := LookupFunc(reg, name)
	require.Truef(t, ok && fn != nil, "func %q not found", name)

	return fn
}

// MustCall executes a Func and fails the test on error, letting assertions focus
// on return values rather than plumbing.
func MustCall(t tb, fn Func, args ...any) any {
	t.Helper()

	res, err := Call(context.Background(), fn, args...)
	require.NoError(t, err)

	return res
}
