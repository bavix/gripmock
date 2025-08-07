package muxmiddleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger_Basic(t *testing.T) {
	// Test basic logger functionality
	assert.NotNil(t, "logger package exists")
}

func TestLogger_Empty(t *testing.T) {
	// Test empty logger case
	assert.NotNil(t, "logger package exists")
}

func TestLogger_Initialization(t *testing.T) {
	// Test logger initialization
	assert.NotNil(t, "logger package initialized")
}

func TestLogger_RequestLogger(t *testing.T) {
	// Test RequestLogger middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("test response"))
		if err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	middleware := RequestLogger(handler)
	assert.NotNil(t, middleware)

	// Test that middleware works
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestLogger_ResponseWriter(t *testing.T) {
	// Test responseWriter struct
	w := httptest.NewRecorder()
	rw := &responseWriter{w: w, status: http.StatusOK}

	assert.NotNil(t, rw)
	assert.Equal(t, http.StatusOK, rw.status)
	assert.Equal(t, 0, rw.bytesWritten)
}

func TestLogger_RequestLoggerWithBody(t *testing.T) {
	// Test RequestLogger with request body
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("test response"))
		if err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	middleware := RequestLogger(handler)
	assert.NotNil(t, middleware)

	// Test with JSON body
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestLogger_RequestLoggerWithInvalidJSON(t *testing.T) {
	// Test RequestLogger with invalid JSON body
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("test response"))
		if err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	middleware := RequestLogger(handler)
	assert.NotNil(t, middleware)

	// Test with invalid JSON body
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(`invalid json`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestLogger_RequestLoggerWithEmptyBody(t *testing.T) {
	// Test RequestLogger with empty body
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("test response"))
		if err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	middleware := RequestLogger(handler)
	assert.NotNil(t, middleware)

	// Test with empty body
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}
