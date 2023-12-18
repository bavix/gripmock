package muxmiddleware

import "net/http"

type responseWriter struct {
	w http.ResponseWriter

	status int
	bytes  int
}

func (r *responseWriter) Header() http.Header {
	return r.w.Header()
}

func (r *responseWriter) Write(bytes []byte) (int, error) {
	n, err := r.w.Write(bytes)

	r.bytes += n

	return n, err
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.w.WriteHeader(statusCode)

	r.status = statusCode
}
