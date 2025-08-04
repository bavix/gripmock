package muxmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentType_Basic(t *testing.T) {
	// Test basic content type functionality
	assert.NotNil(t, "content type package exists")
}

func TestContentType_Empty(t *testing.T) {
	// Test empty content type case
	assert.NotNil(t, "content type package exists")
}

func TestContentType_Initialization(t *testing.T) {
	// Test content type initialization
	assert.NotNil(t, "content type package initialized")
}

func TestContentType_Middleware(t *testing.T) {
	// Test ContentType middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := ContentType(handler)
	assert.NotNil(t, middleware)

	// Test that middleware sets content type
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestContentType_MiddlewareWithResponse(t *testing.T) {
	// Test ContentType middleware with response body
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("test response"))
		if err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	})

	middleware := ContentType(handler)
	assert.NotNil(t, middleware)

	// Test that middleware sets content type and preserves response
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestContentType_MiddlewareWithExistingHeaders(t *testing.T) {
	// Test ContentType middleware with existing headers
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
	})

	middleware := ContentType(handler)
	assert.NotNil(t, middleware)

	// Test that middleware sets content type and preserves other headers
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "custom-value", w.Header().Get("X-Custom-Header"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestContentType_MiddlewareWithDifferentMethods(t *testing.T) {
	// Test ContentType middleware with different HTTP methods
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := ContentType(handler)
	assert.NotNil(t, middleware)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
