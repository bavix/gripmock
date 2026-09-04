package app

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	connectEnvelopeFlagCompressed = 0b00000001

	headerConnectContentEncoding = "Connect-Content-Encoding"
	headerConnectAcceptEncoding  = "Connect-Accept-Encoding"
	headerGRPCEncoding           = "Grpc-Encoding"

	encodingIdentity = "identity"
	encodingGzip     = "gzip"
)

func frameEncoding(hdr http.Header) string {
	for _, key := range []string{headerConnectContentEncoding, headerGRPCEncoding} {
		if v := strings.TrimSpace(hdr.Get(key)); v != "" {
			return strings.ToLower(v)
		}
	}

	return encodingIdentity
}

func decompressFrame(data []byte, encoding string) ([]byte, error) {
	switch encoding {
	case "", encodingIdentity:
		return nil, status.Error(codes.Internal,
			"frame is marked compressed but no compression was negotiated")
	case encodingGzip:
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "malformed gzip frame")
		}

		defer reader.Close() //nolint:errcheck

		out, err := io.ReadAll(io.LimitReader(reader, connectEnvelopeMaxFrameSize+1))
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "malformed gzip frame")
		}

		if len(out) > connectEnvelopeMaxFrameSize {
			return nil, status.Errorf(codes.ResourceExhausted,
				"decompressed frame exceeds the %d byte limit", connectEnvelopeMaxFrameSize)
		}

		return out, nil
	default:
		return nil, status.Errorf(codes.Unimplemented, "unsupported frame compression %q", encoding)
	}
}

func decodeFramePayload(flags byte, data []byte, hdr http.Header) ([]byte, error) {
	if flags&connectEnvelopeFlagCompressed == 0 {
		return data, nil
	}

	return decompressFrame(data, frameEncoding(hdr))
}

const headerGRPCAcceptEncoding = "Grpc-Accept-Encoding"

// responseFrameEncoding picks the per-frame encoding for a streaming response.
// Whole-body gzip is handled a layer up by handlers.CompressHandler, which sets
// Content-Encoding before the handler runs; compressing frames on top of that
// would gzip the same bytes twice.
func responseFrameEncoding(w http.ResponseWriter, r *http.Request) string {
	if w.Header().Get("Content-Encoding") != "" {
		return encodingIdentity
	}

	accept := r.Header.Get(headerConnectAcceptEncoding)
	if accept == "" {
		accept = r.Header.Get(headerGRPCAcceptEncoding)
	}

	for candidate := range strings.SplitSeq(accept, ",") {
		if strings.EqualFold(strings.TrimSpace(candidate), encodingGzip) {
			return encodingGzip
		}
	}

	return encodingIdentity
}

func compressFrame(data []byte, encoding string) ([]byte, byte, error) {
	if encoding != encodingGzip || len(data) == 0 {
		return data, 0, nil
	}

	var buf bytes.Buffer

	writer := gzip.NewWriter(&buf)

	_, err := writer.Write(data)
	if err != nil {
		return nil, 0, err
	}

	err = writer.Close()
	if err != nil {
		return nil, 0, err
	}

	return buf.Bytes(), connectEnvelopeFlagCompressed, nil
}
