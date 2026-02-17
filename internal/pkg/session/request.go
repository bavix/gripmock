package session

import (
	"net/http"
	"strings"
)

const (
	HeaderName = "X-Gripmock-Session"
)

func FromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	if v := strings.TrimSpace(r.Header.Get(HeaderName)); v != "" {
		return v
	}

	return ""
}
