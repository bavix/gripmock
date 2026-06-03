package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestConnectRPCGateway_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/TestService/TestMethod", nil)

	gateway.ServeHTTP(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestConnectRPCGateway_MethodNotFound(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/UnknownService/UnknownMethod", nil)

	gateway.ServeHTTP(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestConnectRPCGateway_StubNotFoundWithoutDescriptor(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/UnknownService/UnknownMethod", bytes.NewReader([]byte(`{}`)))
	r.Header.Set("Content-Type", "application/json")

	gateway.ServeHTTP(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "not found")
}

func TestConnectRPCGateway_InvalidJSON(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/TestService/TestMethod", bytes.NewReader([]byte("invalid json")))
	r.Header.Set("Content-Type", "application/json")

	gateway.ServeHTTP(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestConnectRPCGateway_WriteError(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()

	gateway.writeError(w, codes.Unimplemented, "streaming not supported")

	require.Equal(t, http.StatusNotImplemented, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp map[string]string

	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "unimplemented", resp["code"])
	require.Equal(t, "streaming not supported", resp["message"])
}

func TestConnectRPCGateway_IsJSONContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ct  string
		yes bool
	}{
		{"application/json", true},
		{"application/connect+json", true},
		{"application/proto", false},
		{"application/grpc", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.ct, func(t *testing.T) {
			t.Parallel()

			got := isJSONContentType(tc.ct)
			require.Equal(t, tc.yes, got)
		})
	}
}

func TestConnectRPCGateway_NewConnectRPCGateway(t *testing.T) {
	t.Parallel()

	gateway := NewConnectRPCGateway(nil, nil, nil, nil, nil, nil)
	require.NotNil(t, gateway)
}

func TestHttpStreamAdapter_SendMsg_NonProtoMessage(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	adapter := &httpStreamAdapter{
		ctx:        nil,
		req:        nil,
		w:          rec,
		sentHeader: false,
	}

	err := adapter.SendMsg("not a proto")
	require.NoError(t, err)
	require.True(t, adapter.sentHeader)
}

func TestHttpStreamAdapter_RecvMsg_EmptyBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader([]byte{}))
	adapter := &httpStreamAdapter{
		ctx: req.Context(),
		req: req,
		w:   nil,
	}

	_ = adapter.RecvMsg(nil)
}
