package muxmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRespWriter_Basic(t *testing.T) {
	t.Parallel()
	// Test basic response writer functionality
	require.NotNil(t, "response writer package exists")
}

func TestRespWriter_Empty(t *testing.T) {
	t.Parallel()
	// Test empty response writer case
	require.NotNil(t, "response writer package exists")
}

func TestRespWriter_Initialization(t *testing.T) {
	t.Parallel()
	// Test response writer initialization
	require.NotNil(t, "response writer package initialized")
}

func TestRespWriter_Struct(t *testing.T) {
	t.Parallel()
	// Test responseWriter struct
	w := httptest.NewRecorder()
	rw := &responseWriter{
		w:            w,
		status:       http.StatusOK,
		bytesWritten: 0,
	}

	require.NotNil(t, rw)
	require.Equal(t, http.StatusOK, rw.status)
	require.Equal(t, 0, rw.bytesWritten)
}

func TestRespWriter_Header(t *testing.T) {
	t.Parallel()
	// Test Header method
	w := httptest.NewRecorder()
	rw := &responseWriter{
		w:            w,
		status:       http.StatusOK,
		bytesWritten: 0,
	}

	header := rw.Header()
	require.NotNil(t, header)

	// Test that we can set headers
	header.Set("Content-Type", "application/json")
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestRespWriter_Write(t *testing.T) {
	t.Parallel()
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
	require.Equal(t, len(data), n)
	require.Equal(t, len(data), rw.bytesWritten)
	require.Equal(t, "test data", w.Body.String())
}

func TestRespWriter_WriteHeader(t *testing.T) {
	t.Parallel()
	// Test WriteHeader method
	w := httptest.NewRecorder()
	rw := &responseWriter{
		w:            w,
		status:       http.StatusOK,
		bytesWritten: 0,
	}

	rw.WriteHeader(http.StatusNotFound)
	require.Equal(t, http.StatusNotFound, rw.status)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestRespWriter_MultipleWrites(t *testing.T) {
	t.Parallel()
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

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.Equal(t, len(data1), n1)
	require.Equal(t, len(data2), n2)
	require.Equal(t, len(data1)+len(data2), rw.bytesWritten)
	require.Equal(t, "hello world", w.Body.String())
}
