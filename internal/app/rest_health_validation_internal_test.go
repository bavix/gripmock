package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestRestServerAddStubAllowsProtectedHealthServiceDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
	}{
		{
			name: "equals matcher",
			payload: `[{"service":"grpc.health.v1.Health","method":"Check","input":{"equals":{"service":"gripmock"}},` +
				`"output":{"data":{"status":"NOT_SERVING"}}}]`,
		},
		{
			name: "contains matcher",
			payload: `[{"service":"grpc.health.v1.Health","method":"Watch","input":{"contains":{"service":"gripmock"}},` +
				`"output":{"stream":[{"status":"SERVING"}]}}]`,
		},
		{
			name: "matches matcher",
			payload: `[{"service":"grpc.health.v1.Health","method":"Check","input":{"matches":{"service":"grip.*"}},` +
				`"output":{"data":{"status":"NOT_SERVING"}}}]`,
		},
		{
			name: "matches wildcard",
			payload: `[{"service":"grpc.health.v1.Health","method":"Check","input":{"matches":{"service":".*"}},` +
				`"output":{"data":{"status":"NOT_SERVING"}}}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			server, err := NewRestServer(
				t.Context(),
				stuber.NewBudgerigar(),
				&mockExtender{},
				nil,
				nil,
				nil,
			)
			require.NoError(t, err)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/stubs", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			// Act
			server.AddStub(w, req)

			// Assert
			require.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestRestServerAddStubAllowsCustomHealthService(t *testing.T) {
	t.Parallel()

	// Arrange
	server, err := NewRestServer(
		t.Context(),
		stuber.NewBudgerigar(),
		&mockExtender{},
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)

	payload := `[{"service":"grpc.health.v1.Health","method":"Check","input":{"equals":{"service":"orders.v1.OrderService"}},` +
		`"output":{"data":{"status":"NOT_SERVING"}}}]`
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/stubs", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Act
	server.AddStub(w, req)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)
}
