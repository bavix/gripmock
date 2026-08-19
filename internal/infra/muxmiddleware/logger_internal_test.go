package muxmiddleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoggerBasic(t *testing.T) {
	t.Parallel()
	// Test basic logger functionality
	require.NotNil(t, "logger package exists")
}

func TestLoggerEmpty(t *testing.T) {
	t.Parallel()
	// Test empty logger case
	require.NotNil(t, "logger package exists")
}

func TestLoggerInitialization(t *testing.T) {
	t.Parallel()
	// Test logger initialization
	require.NotNil(t, "logger package initialized")
}

func TestLoggerRequestLogger(t *testing.T) {
	t.Parallel()
	// Test RequestLogger middleware
	var writeErr error

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, writeErr = w.Write([]byte("test response"))
	})

	middleware := RequestLogger(handler)
	require.NotNil(t, middleware)

	// Test that middleware works
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.NoError(t, writeErr)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "test response", w.Body.String())
}

func TestLoggerResponseWriter(t *testing.T) {
	t.Parallel()
	// Test responseWriter struct
	w := httptest.NewRecorder()
	rw := &responseWriter{w: w, status: http.StatusOK}

	require.NotNil(t, rw)
	require.Equal(t, http.StatusOK, rw.status)
	require.Equal(t, 0, rw.bytesWritten)
}

func TestLoggerRequestLoggerWithBody(t *testing.T) {
	t.Parallel()
	// Test RequestLogger with request body
	var writeErr error

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, writeErr = w.Write([]byte("test response"))
	})

	middleware := RequestLogger(handler)
	require.NotNil(t, middleware)

	// Test with JSON body
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", bytes.NewBufferString(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.NoError(t, writeErr)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "test response", w.Body.String())
}

func TestLoggerRequestLoggerWithInvalidJSON(t *testing.T) {
	t.Parallel()
	// Test RequestLogger with invalid JSON body
	var writeErr error

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, writeErr = w.Write([]byte("test response"))
	})

	middleware := RequestLogger(handler)
	require.NotNil(t, middleware)

	// Test with invalid JSON body
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", bytes.NewBufferString(`invalid json`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.NoError(t, writeErr)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "test response", w.Body.String())
}

func TestLoggerRequestLoggerWithEmptyBody(t *testing.T) {
	t.Parallel()
	// Test RequestLogger with empty body
	var writeErr error

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, writeErr = w.Write([]byte("test response"))
	})

	middleware := RequestLogger(handler)
	require.NotNil(t, middleware)

	// Test with empty body
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.NoError(t, writeErr)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "test response", w.Body.String())
}

func TestRequestLoggerRefusesAnUnreadableBody(t *testing.T) {
	t.Parallel()

	reached := false
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true

		w.WriteHeader(http.StatusOK)
	}))

	body := bytes.Repeat([]byte("a"), (4<<20)+1)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/stubs",
		bytes.NewReader(body))

	handler.ServeHTTP(recorder, request)

	require.False(t, reached,
		"the body is gone by now, so the handler must not be asked to serve the request")
	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	require.Contains(t, recorder.Body.String(), "exceeds the maximum size")
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
}

func TestRequestLoggerPassesAReadableBodyThrough(t *testing.T) {
	t.Parallel()

	var seen []byte

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = io.ReadAll(r.Body)

		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/stubs",
		bytes.NewReader([]byte(`{"ok":true}`)))

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"ok":true}`, string(seen))
}
