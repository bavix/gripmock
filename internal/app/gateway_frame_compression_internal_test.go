package app

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
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
