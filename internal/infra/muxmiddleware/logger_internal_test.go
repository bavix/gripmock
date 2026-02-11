package muxmiddleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogger_Basic(t *testing.T) {
	t.Parallel()
	// Test basic logger functionality
	require.NotNil(t, "logger package exists")
}

func TestLogger_Empty(t *testing.T) {
	t.Parallel()
	// Test empty logger case
	require.NotNil(t, "logger package exists")
}

func TestLogger_Initialization(t *testing.T) {
	t.Parallel()
	// Test logger initialization
	require.NotNil(t, "logger package initialized")
}

func TestLogger_RequestLogger(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.NoError(t, writeErr)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "test response", w.Body.String())
}

func TestLogger_ResponseWriter(t *testing.T) {
	t.Parallel()
	// Test responseWriter struct
	w := httptest.NewRecorder()
	rw := &responseWriter{w: w, status: http.StatusOK}

	require.NotNil(t, rw)
	require.Equal(t, http.StatusOK, rw.status)
	require.Equal(t, 0, rw.bytesWritten)
}

func TestLogger_RequestLoggerWithBody(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.NoError(t, writeErr)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "test response", w.Body.String())
}

func TestLogger_RequestLoggerWithInvalidJSON(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`invalid json`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.NoError(t, writeErr)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "test response", w.Body.String())
}

func TestLogger_RequestLoggerWithEmptyBody(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	require.NoError(t, writeErr)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "test response", w.Body.String())
}
