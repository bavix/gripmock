package stuber //nolint:testpackage

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/features"
)

func TestQuery_RequestInternal(t *testing.T) {
	t.Parallel()

	q := Query{
		toggles: features.New(),
	}
	require.False(t, q.RequestInternal())

	q = Query{
		toggles: features.New(RequestInternalFlag),
	}
	require.True(t, q.RequestInternal())
}

func TestToggles(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	togglesResult := toggles(req)
	require.False(t, togglesResult.Has(RequestInternalFlag))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	req.Header.Set("X-Gripmock-Requestinternal", "true")
	togglesResult = toggles(req)
	require.True(t, togglesResult.Has(RequestInternalFlag))
}

func TestNewQuery_WithBody(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"service": "TestService",
		"method":  "TestMethod",
		"data":    map[string]any{"key": "value"},
		"headers": map[string]any{"header": "value"},
	}

	body, err := json.Marshal(data)
	require.NoError(t, err)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", bytes.NewBuffer(body))

	q, err := NewQuery(req)
	require.NoError(t, err)
	require.Equal(t, "TestService", q.Service)
	require.Equal(t, "TestMethod", q.Method)
	require.Equal(t, map[string]any{"key": "value"}, q.Data)
	require.Equal(t, map[string]any{"header": "value"}, q.Headers)
}

func TestNewQuery_WithID(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	data := map[string]any{
		"id":      id.String(),
		"service": "TestService",
		"method":  "TestMethod",
	}

	body, err := json.Marshal(data)
	require.NoError(t, err)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", bytes.NewBuffer(body))

	q, err := NewQuery(req)
	require.NoError(t, err)
	require.Equal(t, id, *q.ID)
	require.Equal(t, "TestService", q.Service)
	require.Equal(t, "TestMethod", q.Method)
}

func TestNewQuery_InvalidJSON(t *testing.T) {
	t.Parallel()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", bytes.NewBufferString("invalid json"))

	_, err := NewQuery(req)
	require.Error(t, err)
}

func TestNewQueryBidi(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"service":"svc","method":"mthd","headers":{"h":"v"}}`))
	req.Header.Set("Content-Type", "application/json")
	q, err := NewQueryBidi(req)
	require.NoError(t, err)
	require.Equal(t, "svc", q.Service)
	require.Equal(t, "mthd", q.Method)
	require.Equal(t, "v", q.Headers["h"])
}

func TestRequestInternalBidi(t *testing.T) {
	t.Parallel()

	q := QueryBidi{
		Service: "svc",
		Method:  "mthd",
		Headers: map[string]any{"h": "v"},
	}
	require.False(t, q.RequestInternal())
}

func TestRequestInternalV2(t *testing.T) {
	t.Parallel()

	q := QueryV2{
		Service: "svc",
		Method:  "mthd",
		Headers: map[string]any{"h": "v"},
		Input:   []map[string]any{{"key": "value"}},
	}
	require.False(t, q.RequestInternal())
}
