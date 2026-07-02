package httputil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureBody returns a handler that records the request body and encoding
// header into the supplied pointers.
func captureBody(got *[]byte, gotEncoding *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*gotEncoding = r.Header.Get("Content-Encoding")

		buf, _ := io.ReadAll(r.Body)

		*got = buf

		w.WriteHeader(http.StatusOK)
	}
}

// encodeFunc compresses payload into buf.
type encodeFunc func(buf *bytes.Buffer, payload []byte) error

// encodeGzip / encodeDeflate / encodeZstd / encodeSnappy / encodeBrotli are
// the encoder factories used in the table-driven decompress tests.
func encodeGzip(buf *bytes.Buffer, payload []byte) error {
	w := gzip.NewWriter(buf)
	if _, err := w.Write(payload); err != nil {
		return err
	}

	return w.Close()
}

func encodeDeflate(buf *bytes.Buffer, payload []byte) error {
	w, err := flate.NewWriter(buf, flate.DefaultCompression)
	if err != nil {
		return err
	}

	if _, err := w.Write(payload); err != nil {
		return err
	}

	return w.Close()
}

func encodeZstd(buf *bytes.Buffer, payload []byte) error {
	w, err := zstd.NewWriter(buf)
	if err != nil {
		return err
	}

	if _, err := w.Write(payload); err != nil {
		return err
	}

	return w.Close()
}

func encodeSnappy(buf *bytes.Buffer, payload []byte) error {
	w := snappy.NewBufferedWriter(buf)
	if _, err := w.Write(payload); err != nil {
		return err
	}

	return w.Close()
}

func encodeBrotli(buf *bytes.Buffer, payload []byte) error {
	w := brotli.NewWriter(buf)
	if _, err := w.Write(payload); err != nil {
		return err
	}

	return w.Close()
}

func TestGzipRequestMiddleware_Plain(t *testing.T) {
	t.Parallel()

	body := []byte("hello world")
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	var (
		got         []byte
		gotEncoding string
	)
	GzipRequestMiddleware(captureBody(&got, &gotEncoding)).ServeHTTP(rec, r)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, gotEncoding)
	assert.Equal(t, body, got)
}

func TestGzipRequestMiddleware_Decompress(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"hello":"world","items":[1,2,3]}`)

	tests := []struct {
		name     string
		encoding string
		encode   encodeFunc
	}{
		{"gzip", "gzip", encodeGzip},
		{"gzip case-insensitive", "GZIP", encodeGzip},
		{"deflate", "deflate", encodeDeflate},
		{"zstd", "zstd", encodeZstd},
		{"snappy", "snappy", encodeSnappy},
		{"brotli", "br", encodeBrotli},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			require.NoError(t, tc.encode(&buf, payload))

			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader(buf.Bytes()))
			r.Header.Set("Content-Encoding", tc.encoding)

			rec := httptest.NewRecorder()

			var (
				got         []byte
				gotEncoding string
			)
			GzipRequestMiddleware(captureBody(&got, &gotEncoding)).ServeHTTP(rec, r)

			require.Equal(t, http.StatusOK, rec.Code)
			assert.Empty(t, gotEncoding, "Content-Encoding should be removed after decompression")
			assert.Equal(t, payload, got)
		})
	}
}

func TestGzipRequestMiddleware_InvalidGzip(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader([]byte("not a gzip stream")))
	r.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	called := false

	GzipRequestMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true

		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, r)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, called, "handler should not be called for invalid gzip body")
}

func TestGzipRequestMiddleware_UnknownEncoding(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader([]byte("body")))
	r.Header.Set("Content-Encoding", "lzma") // unknown - not supported

	rec := httptest.NewRecorder()
	GzipRequestMiddleware(captureBody(nil, nil)).ServeHTTP(rec, r)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGzipRequestMiddleware_NilBody(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	called := false

	GzipRequestMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true

		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, r)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called)
}
