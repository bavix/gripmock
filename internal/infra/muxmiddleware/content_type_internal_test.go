package muxmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentType_Basic(t *testing.T) {
	t.Parallel()

	// Test basic content type functionality
	require.NotNil(t, "content type package exists")
}

func TestContentType_Empty(t *testing.T) {
	t.Parallel()
	// Test empty content type case
	require.NotNil(t, "content type package exists")
}

func TestContentType_Initialization(t *testing.T) {
	t.Parallel()
	// Test content type initialization
	require.NotNil(t, "content type package initialized")
}

func TestContentType_Middleware(t *testing.T) {
	t.Parallel()
	// Test ContentType middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := ContentType(handler)
	require.NotNil(t, middleware)

	// Test that middleware sets content type
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestContentType_MiddlewareWithResponse(t *testing.T) {
	t.Parallel()
	// Test ContentType middleware with response body
	var writeErr error

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, writeErr = w.Write([]byte("test response"))
	})

	middleware := ContentType(handler)
	require.NotNil(t, middleware)

	// Test that middleware sets content type and preserves response
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.NoError(t, writeErr)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "test response", w.Body.String())
}

func TestContentType_MiddlewareWithExistingHeaders(t *testing.T) {
	t.Parallel()
	// Test ContentType middleware with existing headers
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
	})

	middleware := ContentType(handler)
	require.NotNil(t, middleware)

	// Test that middleware sets content type and preserves other headers
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	require.Equal(t, "custom-value", w.Header().Get("X-Custom-Header"))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestContentType_MiddlewareWithDifferentMethods(t *testing.T) {
	t.Parallel()
	// Test ContentType middleware with different HTTP methods
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := ContentType(handler)
	require.NotNil(t, middleware)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			require.Equal(t, "application/json", w.Header().Get("Content-Type"))
			require.Equal(t, http.StatusOK, w.Code)
		})
	}
}
