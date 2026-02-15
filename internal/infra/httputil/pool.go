package httputil

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
)

var errInvalidPoolType = errors.New("invalid buffer pool type")

type bodyKey struct{}

// ContextWithBody stores body bytes in context. Used by middleware that pre-reads body.
func ContextWithBody(ctx context.Context, body []byte) context.Context {
	return context.WithValue(ctx, bodyKey{}, body)
}

// BodyFromContext returns body bytes if set by middleware; nil otherwise.
func BodyFromContext(ctx context.Context) []byte {
	b, _ := ctx.Value(bodyKey{}).([]byte)

	return b
}

// RequestBody returns body bytes: from context if set, otherwise reads from r.Body.
func RequestBody(r *http.Request) ([]byte, error) {
	if b := BodyFromContext(r.Context()); b != nil {
		return b, nil
	}

	return ReadBody(r)
}

const (
	defaultBufferSize = 32 << 10
	maxRequestBody    = 4 << 20 // 4MB
)

var bufferPool = sync.Pool{ //nolint:gochecknoglobals
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, defaultBufferSize))
	},
}

// MaxBodyBytes returns the maximum allowed request body size in bytes.
func MaxBodyBytes() int64 {
	return maxRequestBody
}

// MaxBodySize limits request body size.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil || r.Body == http.NoBody {
				next.ServeHTTP(w, r)

				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// ReadBody reads the full body using a pooled buffer. Returns a copy; caller owns the result.
func ReadBody(r *http.Request) ([]byte, error) {
	if r.Body == nil || r.Body == http.NoBody {
		return nil, nil
	}

	buf, ok := bufferPool.Get().(*bytes.Buffer)
	if !ok {
		return nil, errInvalidPoolType
	}

	buf.Reset()

	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	_, err := buf.ReadFrom(io.LimitReader(r.Body, maxRequestBody))
	_ = r.Body.Close()

	if err != nil {
		return nil, err
	}

	body := buf.Bytes()
	result := make([]byte, len(body))
	copy(result, body)

	return result, nil
}
