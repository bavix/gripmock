package httputil

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/snappy"
	"github.com/klauspost/compress/zstd"
)

// ErrUnsupportedEncoding is returned by decompressReader when the request
// advertises a Content-Encoding the middleware does not understand.
var ErrUnsupportedEncoding = errors.New("unsupported encoding")

// GzipRequestMiddleware decompresses request bodies whose Content-Encoding
// indicates a supported compression algorithm. Supported encodings:
//   - gzip     (klauspost/compress/gzip)
//   - deflate  (klauspost/compress/flate, raw deflate RFC 1951)
//   - zstd     (klauspost/compress/zstd)
//   - snappy   (klauspost/compress/snappy)
//   - br       (andybalholm/brotli)
//
// The original request is replaced with an uncompressed body and the
// Content-Encoding header is removed so downstream handlers see plain bytes.
//
// Bodies with no Content-Encoding or an unsupported encoding are passed
// through untouched.
//
// Outbound compression is provided by gorilla/handlers.CompressHandler,
// which is wired in deps/rest_server.go and deps/connect_server.go.
func GzipRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil || r.Body == http.NoBody {
			next.ServeHTTP(w, r)

			return
		}

		enc := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Encoding")))
		if enc == "" {
			next.ServeHTTP(w, r)

			return
		}

		rc, err := decompressReader(enc, r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid %s body: %v", enc, err), http.StatusBadRequest)

			return
		}

		r.Body = &decompressingBody{reader: rc, source: r.Body}
		r.Header.Del("Content-Encoding")
		r.ContentLength = -1

		next.ServeHTTP(w, r)
	})
}

// decompressReader wraps src in a reader for the given encoding. The
// returned ReadCloser MUST be closed by the caller.
func decompressReader(enc string, src io.Reader) (io.ReadCloser, error) {
	switch enc {
	case "gzip":
		return gzip.NewReader(src)
	case "deflate":
		return flate.NewReader(src), nil
	case "zstd":
		d, err := zstd.NewReader(src)
		if err != nil {
			return nil, err
		}

		return d.IOReadCloser(), nil
	case "snappy":
		return io.NopCloser(snappy.NewReader(src)), nil
	case "br":
		return io.NopCloser(brotli.NewReader(src)), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedEncoding, enc)
	}
}

// decompressingBody pairs the decompressed reader with the original request
// body so callers that close the body release the underlying connection AND
// the decompressor resources.
type decompressingBody struct {
	reader io.ReadCloser
	source io.ReadCloser
}

func (d *decompressingBody) Read(p []byte) (int, error) { return d.reader.Read(p) }

func (d *decompressingBody) Close() error {
	err1 := d.reader.Close()
	err2 := d.source.Close()

	return errors.Join(err1, err2)
}
