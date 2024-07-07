package muxmiddleware

import (
	"net/http"
)

type responseWriter struct {
	w http.ResponseWriter

	status       int
	bytesWritten int
}

func (rw *responseWriter) Header() http.Header {
	return rw.w.Header()
}

func (rw *responseWriter) Write(bytes []byte) (int, error) {
	n, err := rw.w.Write(bytes)
	rw.bytesWritten += n

	return n, err
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
	rw.w.WriteHeader(statusCode)
}
