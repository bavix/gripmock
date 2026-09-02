package app

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestResponseFrameEncoding(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		requestHeaders  map[string]string
		responseHeaders map[string]string
		want            string
	}{
		"no accept header": {
			want: encodingIdentity,
		},
		"connect accepts gzip": {
			requestHeaders: map[string]string{headerConnectAcceptEncoding: "gzip"},
			want:           encodingGzip,
		},
		"grpc-web accepts gzip": {
			requestHeaders: map[string]string{headerGRPCAcceptEncoding: "identity, gzip"},
			want:           encodingGzip,
		},
		"unsupported codec": {
			requestHeaders: map[string]string{headerConnectAcceptEncoding: "br"},
			want:           encodingIdentity,
		},
		"http layer already compressing": {
			requestHeaders:  map[string]string{headerConnectAcceptEncoding: "gzip"},
			responseHeaders: map[string]string{"Content-Encoding": "gzip"},
			want:            encodingIdentity,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/svc/Method", nil)
			for k, v := range tc.requestHeaders {
				r.Header.Set(k, v)
			}

			rec := httptest.NewRecorder()
			for k, v := range tc.responseHeaders {
				rec.Header().Set(k, v)
			}

			require.Equal(t, tc.want, responseFrameEncoding(rec, r))
		})
	}
}

func TestWriteConnectFrameEncodedRoundTrip(t *testing.T) {
	t.Parallel()

	payload := bytes.Repeat([]byte(`{"message":"hello"}`), 20)

	var buf bytes.Buffer
	require.NoError(t, writeConnectFrameEncoded(&buf, payload, false, encodingGzip))

	body := buf.Bytes()
	require.Equal(t, byte(connectEnvelopeFlagCompressed), body[0])
	require.Equal(t, len(body)-ConnectEnvelopeHeaderSize,
		int(binary.BigEndian.Uint32(body[1:5])))
	require.Less(t, len(body), len(payload), "gzip must actually shrink a repetitive payload")

	hdr := http.Header{}
	hdr.Set(headerConnectContentEncoding, encodingGzip)

	decoded, err := decodeFramePayload(body[0], body[ConnectEnvelopeHeaderSize:], hdr)
	require.NoError(t, err)
	require.Equal(t, payload, decoded)
}

func TestWriteConnectFrameEncodedKeepsEndStreamFlag(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writeConnectFrameEncoded(&buf, []byte("{}"), true, encodingGzip))

	flags := buf.Bytes()[0]
	require.NotZero(t, flags&connectEnvelopeFlagEndStream)
	require.NotZero(t, flags&connectEnvelopeFlagCompressed)
}

func TestWriteConnectFrameEncodedIdentityMatchesPlain(t *testing.T) {
	t.Parallel()

	var encoded, plain bytes.Buffer
	require.NoError(t, writeConnectFrameEncoded(&encoded, []byte("payload"), false, encodingIdentity))
	require.NoError(t, writeConnectFrame(&plain, []byte("payload"), false))

	require.Equal(t, plain.Bytes(), encoded.Bytes())
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)

	return len(p), nil
}

func gzipZeros(t *testing.T, n int64) []byte {
	t.Helper()

	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)
	_, err := io.CopyN(zw, zeroReader{}, n)
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	return buf.Bytes()
}

func TestDecompressFrameRejectsDecompressionBomb(t *testing.T) {
	t.Parallel()

	frame := gzipZeros(t, connectEnvelopeMaxFrameSize+1)
	require.Less(t, len(frame), connectEnvelopeMaxFrameSize,
		"the bomb must stay small on the wire")

	hdr := http.Header{}
	hdr.Set(headerGRPCEncoding, encodingGzip)

	payload, err := decodeFramePayload(connectEnvelopeFlagCompressed, frame, hdr)
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Zero(t, len(payload), "an oversized frame must yield no payload") //nolint:testifylint
}

func TestDecompressFrameAcceptsPayloadAtTheLimit(t *testing.T) {
	t.Parallel()

	frame := gzipZeros(t, connectEnvelopeMaxFrameSize)

	hdr := http.Header{}
	hdr.Set(headerConnectContentEncoding, encodingGzip)

	payload, err := decodeFramePayload(connectEnvelopeFlagCompressed, frame, hdr)
	require.NoError(t, err)
	require.Len(t, payload, connectEnvelopeMaxFrameSize)
}
