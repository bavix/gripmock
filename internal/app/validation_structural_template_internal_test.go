package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestValidateStructuralTemplateOutput(t *testing.T) {
	t.Parallel()

	validator, err := NewStubValidator()
	require.NoError(t, err)

	tests := []struct {
		name    string
		output  stuber.Output
		wantErr bool
	}{
		{name: "data template", output: stuber.Output{DataTemplate: "value: {{ .Request.value }}"}},
		{name: "stream template", output: stuber.Output{StreamTemplate: "[]"}},
		{name: "data and data template", output: stuber.Output{Data: map[string]any{"ok": true}, DataTemplate: "ok: true"}, wantErr: true},
		{
			name:    "stream and stream template",
			output:  stuber.Output{Stream: []any{map[string]any{"ok": true}}, StreamTemplate: "[]"},
			wantErr: true,
		},
		{name: "data and stream templates", output: stuber.Output{DataTemplate: "ok: true", StreamTemplate: "[]"}, wantErr: true},
		{name: "blank template", output: stuber.Output{DataTemplate: "  \n"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &stuber.Stub{
				Service: "example.Service",
				Method:  "Call",
				Input:   stuber.InputData{Equals: map[string]any{"id": "1"}},
				Output:  tt.output,
			}

			err := validator.Struct(stub)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}
