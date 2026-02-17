package session

import (
	"context"
	"net/http"
	"strings"
)

const (
	HeaderName = "X-Gripmock-Session"
)

type contextKey struct{}

// WithContext stores transport session in context for internal propagation.
func WithContext(ctx context.Context, sessionID string) context.Context {
	if strings.TrimSpace(sessionID) == "" {
		return ctx
	}

	return context.WithValue(ctx, contextKey{}, strings.TrimSpace(sessionID))
}

// ConsumeRequest moves session from transport header into request context and removes the header.
func ConsumeRequest(r *http.Request) *http.Request {
	if r == nil {
		return nil
	}

	v := strings.TrimSpace(r.Header.Get(HeaderName))
	if v == "" {
		return r
	}

	Touch(v)

	r.Header.Del(HeaderName)

	return r.WithContext(WithContext(r.Context(), v))
}

func FromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	if v, ok := r.Context().Value(contextKey{}).(string); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}

	if v := strings.TrimSpace(r.Header.Get(HeaderName)); v != "" {
		return v
	}

	return ""
}
