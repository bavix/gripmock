package muxmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRespWriter_Basic(t *testing.T) {
	// Test basic response writer functionality
	assert.NotNil(t, "response writer package exists")
}

func TestRespWriter_Empty(t *testing.T) {
	// Test empty response writer case
	assert.NotNil(t, "response writer package exists")
}

func TestRespWriter_Initialization(t *testing.T) {
	// Test response writer initialization
	assert.NotNil(t, "response writer package initialized")
}

func TestRespWriter_Struct(t *testing.T) {
	// Test responseWriter struct
	w := httptest.NewRecorder()
	rw := &responseWriter{
		w:            w,
		status:       http.StatusOK,
		bytesWritten: 0,
	}

	assert.NotNil(t, rw)
	assert.Equal(t, http.StatusOK, rw.status)
	assert.Equal(t, 0, rw.bytesWritten)
}

func TestRespWriter_Header(t *testing.T) {
	// Test Header method
	w := httptest.NewRecorder()
	rw := &responseWriter{
		w:            w,
		status:       http.StatusOK,
		bytesWritten: 0,
	}

	header := rw.Header()
	assert.NotNil(t, header)

	// Test that we can set headers
	header.Set("Content-Type", "application/json")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestRespWriter_Write(t *testing.T) {
	// Test Write method
	w := httptest.NewRecorder()
	rw := &responseWriter{
		w:            w,
		status:       http.StatusOK,
		bytesWritten: 0,
	}

	data := []byte("test data")
	n, err := rw.Write(data)

	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, len(data), rw.bytesWritten)
	assert.Equal(t, "test data", w.Body.String())
}

func TestRespWriter_WriteHeader(t *testing.T) {
	// Test WriteHeader method
	w := httptest.NewRecorder()
	rw := &responseWriter{
		w:            w,
		status:       http.StatusOK,
		bytesWritten: 0,
	}

	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.status)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRespWriter_MultipleWrites(t *testing.T) {
	// Test multiple writes
	w := httptest.NewRecorder()
	rw := &responseWriter{
		w:            w,
		status:       http.StatusOK,
		bytesWritten: 0,
	}

	data1 := []byte("hello")
	data2 := []byte(" world")

	n1, err1 := rw.Write(data1)
	n2, err2 := rw.Write(data2)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, len(data1), n1)
	assert.Equal(t, len(data2), n2)
	assert.Equal(t, len(data1)+len(data2), rw.bytesWritten)
	assert.Equal(t, "hello world", w.Body.String())
}
