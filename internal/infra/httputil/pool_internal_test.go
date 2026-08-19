package httputil

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var errPreRead = errors.New("pre-read failed")

func postWithBody(t *testing.T, body []byte) *http.Request {
	t.Helper()

	return httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/stubs", bytes.NewReader(body))
}

func TestReadBodyAtTheLimit(t *testing.T) {
	t.Parallel()

	body := bytes.Repeat([]byte("a"), maxRequestBody)

	got, err := ReadBody(postWithBody(t, body))

	require.NoError(t, err)
	require.Len(t, got, maxRequestBody)
}

func TestReadBodyRefusesOversizedBody(t *testing.T) {
	t.Parallel()

	body := bytes.Repeat([]byte("a"), maxRequestBody+1)

	_, err := ReadBody(postWithBody(t, body))

	require.ErrorIs(t, err, ErrBodyTooLarge)
}

func TestReadBodyWithoutBody(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/stubs", nil)

	got, err := ReadBody(request)

	require.NoError(t, err)
	require.Nil(t, got)
}

func TestRequestBodyPrefersTheContextCopy(t *testing.T) {
	t.Parallel()

	request := postWithBody(t, []byte("from the reader"))
	request = request.WithContext(ContextWithBody(request.Context(), []byte("from the context")))

	got, err := RequestBody(request)

	require.NoError(t, err)
	require.Equal(t, "from the context", string(got))
}

func TestRequestBodyReportsWhyMiddlewareCouldNotRead(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/stubs",
		io.NopCloser(strings.NewReader("")))
	request = request.WithContext(ContextWithBodyError(request.Context(), errPreRead))

	_, err := RequestBody(request)

	require.ErrorIs(t, err, errPreRead)
}

func TestBodyErrorFromContextWithoutOne(t *testing.T) {
	t.Parallel()

	require.NoError(t, BodyErrorFromContext(context.Background()))
}
