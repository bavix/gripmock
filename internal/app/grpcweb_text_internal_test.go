package app

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsGRPCWebTextContentType(t *testing.T) {
	t.Parallel()

	for ct, want := range map[string]bool{
		grpcwebContentTypeText:      true,
		grpcwebContentTypeTextProto: true,
		grpcwebContentTypeTextJSON:  true,
		grpcwebContentTypeProto:     false,
		grpcwebContentTypeJSON:      false,
		contentTypeConnectProto:     false,
	} {
		require.Equal(t, want, isGRPCWebTextContentType(ct), ct)
	}
}

func TestDecodeGRPCWebTextAcceptsEveryPadding(t *testing.T) {
	t.Parallel()

	payload := []byte{0x00, 0x00, 0x00, 0x00, 0x02, 0x08, 0x2a}

	t.Run("padded", func(t *testing.T) {
		t.Parallel()

		got, err := decodeGRPCWebText([]byte(base64.StdEncoding.EncodeToString(payload)))
		require.NoError(t, err)
		require.Equal(t, payload, got)
	})

	t.Run("unpadded", func(t *testing.T) {
		t.Parallel()

		got, err := decodeGRPCWebText([]byte(base64.RawStdEncoding.EncodeToString(payload)))
		require.NoError(t, err)
		require.Equal(t, payload, got)
	})

	t.Run("concatenated frames", func(t *testing.T) {
		t.Parallel()

		// A streaming client encodes each frame on its own, so the body is a
		// run of independently padded chunks rather than one base64 string.
		first := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x41}
		second := []byte{0x80, 0x00, 0x00, 0x00, 0x01, 0x42}
		body := base64.StdEncoding.EncodeToString(first) + base64.StdEncoding.EncodeToString(second)

		got, err := decodeGRPCWebText([]byte(body))
		require.NoError(t, err)
		require.Equal(t, append(append([]byte{}, first...), second...), got)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		got, err := decodeGRPCWebText([]byte("  "))
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("malformed", func(t *testing.T) {
		t.Parallel()

		_, err := decodeGRPCWebText([]byte("!!!not base64!!!"))
		require.Error(t, err)
	})
}

func TestBase64StreamWriterEncodesWholeBody(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	w := newBase64StreamWriter(rec)

	// A frame reaches the writer as a header write followed by a payload
	// write; encoding each separately would corrupt the stream.
	header := []byte{0x00, 0x00, 0x00, 0x00, 0x01}
	payload := []byte{0x41}
	trailers := []byte{0x80, 0x00, 0x00, 0x00, 0x01, 0x42}

	for _, chunk := range [][]byte{header, payload, trailers} {
		_, err := w.Write(chunk)
		require.NoError(t, err)
	}

	require.NoError(t, w.Close())

	want := append(append(append([]byte{}, header...), payload...), trailers...)
	decoded, err := base64.StdEncoding.DecodeString(rec.Body.String())
	require.NoError(t, err, "the body must decode in a single base64 pass")
	require.Equal(t, want, decoded)
}

func TestSetGRPCWebContentTypeEchoesTextMode(t *testing.T) {
	t.Parallel()

	for reqCT, wantCT := range map[string]string{
		grpcwebContentTypeTextProto: grpcwebContentTypeTextProto,
		grpcwebContentTypeTextJSON:  grpcwebContentTypeTextJSON,
		grpcwebContentTypeText:      grpcwebContentTypeTextProto,
		grpcwebContentTypeJSON:      grpcwebContentTypeJSON,
		grpcwebContentTypeProto:     grpcwebContentTypeProto,
	} {
		rec := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/s/m", nil)
		r.Header.Set(headerContentType, reqCT)

		setGRPCWebContentType(rec, r)
		require.Equal(t, wantCT, rec.Header().Get(headerContentType), "request %s", reqCT)
	}
}
