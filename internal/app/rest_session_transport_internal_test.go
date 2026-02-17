package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/pkg/session"
)

type nopExtender struct{}

func (nopExtender) Wait(context.Context) {}

func TestRestAddStub_SessionFromHeaderOnly(t *testing.T) {
	t.Parallel()

	// Arrange
	b := stuber.NewBudgerigar(features.New())
	srv, err := NewRestServer(t.Context(), b, nopExtender{}, nil, nil, nil)
	require.NoError(t, err)

	body := []byte(`[
		{"service":"svc.Greeter","method":"SayHello","session":"BODY","input":{"equals":{"name":"Bob"}},"output":{"data":{"message":"ok"}}}
	]`)
	req := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(session.HeaderName, "HEADER")

	w := httptest.NewRecorder()

	// Act
	srv.AddStub(w, req)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)

	all := b.All()
	require.Len(t, all, 1)
	require.Equal(t, "HEADER", all[0].Session)
}

func TestRestAddStub_WithoutHeaderUsesGlobal(t *testing.T) {
	t.Parallel()

	// Arrange
	b := stuber.NewBudgerigar(features.New())
	srv, err := NewRestServer(t.Context(), b, nopExtender{}, nil, nil, nil)
	require.NoError(t, err)

	body := []byte(`[
		{"service":"svc.Greeter","method":"SayHello","session":"BODY","input":{"equals":{"name":"Bob"}},"output":{"data":{"message":"ok"}}}
	]`)
	req := httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Act
	srv.AddStub(w, req)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)

	all := b.All()
	require.Len(t, all, 1)
	require.Empty(t, all[0].Session)
}
