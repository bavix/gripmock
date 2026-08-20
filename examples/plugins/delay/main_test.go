package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

func TestFibonacciRegistered(t *testing.T) {
	t.Parallel()

	reg := plugintest.NewRegistry()
	Register(reg)

	fn, ok := plugintest.LookupFunc(reg, "fibonacci")
	require.True(t, ok, "fibonacci not registered")

	tests := []struct {
		name    string
		attempt any
		step    any
		want    string
	}{
		{name: "first attempt", attempt: 1, step: "50ms", want: "50ms"},
		{name: "second attempt", attempt: 2, step: "50ms", want: "50ms"},
		{name: "third attempt", attempt: 3, step: "50ms", want: "100ms"},
		{name: "fifth attempt", attempt: 5, step: "50ms", want: "250ms"},
		{name: "float attempt", attempt: 5.0, step: "50ms", want: "250ms"},
		{name: "zero attempt", attempt: 0, step: "50ms", want: "50ms"},
		{name: "negative attempt", attempt: -3, step: "50ms", want: "50ms"},
		{name: "unparsable step", attempt: 1, step: "nope", want: "0s"},
		{name: "step is not a string", attempt: 1, step: 50, want: "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := plugintest.Call(t.Context(), fn, tt.attempt, tt.step)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
