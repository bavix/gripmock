package app

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
)

const (
	grpcwebContentTypeText      = "application/grpc-web-text"
	grpcwebContentTypeTextProto = "application/grpc-web-text+proto"
	grpcwebContentTypeTextJSON  = "application/grpc-web-text+json"
)

func isGRPCWebTextContentType(ct string) bool {
	return strings.HasPrefix(normalizeContentType(ct), grpcwebContentTypeText)
}

func grpcwebTextResponseContentType(ct string) string {
	if normalizeContentType(ct) == grpcwebContentTypeTextJSON {
		return grpcwebContentTypeTextJSON
	}

	return grpcwebContentTypeTextProto
}

type base64StreamWriter struct {
	http.ResponseWriter

	encoder io.WriteCloser
}

func newBase64StreamWriter(w http.ResponseWriter) *base64StreamWriter {
	return &base64StreamWriter{
		ResponseWriter: w,
		encoder:        base64.NewEncoder(base64.StdEncoding, w),
	}
}

func (w *base64StreamWriter) Write(p []byte) (int, error) {
	return w.encoder.Write(p) //nolint:wrapcheck
}

func (w *base64StreamWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *base64StreamWriter) Close() error {
	return w.encoder.Close() //nolint:wrapcheck
}

func decodeGRPCWebText(raw []byte) ([]byte, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err == nil {
		return decoded, nil
	}

	decoded, err = base64.RawStdEncoding.DecodeString(trimmed)
	if err == nil {
		return decoded, nil
	}

	return decodeConcatenatedBase64(trimmed)
}

func decodeConcatenatedBase64(s string) ([]byte, error) {
	var out []byte

	start := 0

	for i := 0; i < len(s); i++ {
		if s[i] != '=' {
			continue
		}

		end := i + 1
		for end < len(s) && s[end] == '=' {
			end++
		}

		decoded, err := base64.StdEncoding.DecodeString(s[start:end])
		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		out = append(out, decoded...)
		start = end
		i = end - 1
	}

	if start < len(s) {
		decoded, err := base64.RawStdEncoding.DecodeString(s[start:])
		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		out = append(out, decoded...)
	}

	return out, nil
}
