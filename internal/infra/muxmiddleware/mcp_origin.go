package muxmiddleware

import (
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

//nolint:gochecknoglobals
var loopbackHosts = map[string]struct{}{
	"localhost": {},
	"127.0.0.1": {},
	"::1":       {},
	"[::1]":     {},
}

func MCPOriginGuard(allowed []string) func(http.Handler) http.Handler {
	explicit := explicitOrigins(allowed)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" || originAllowed(origin, explicit) {
				next.ServeHTTP(w, r)

				return
			}

			http.Error(w, "origin not allowed", http.StatusForbidden)
		})
	}
}

func explicitOrigins(allowed []string) []string {
	explicit := make([]string, 0, len(allowed))

	for _, origin := range allowed {
		if trimmed := strings.ToLower(strings.TrimSpace(origin)); trimmed != "" && trimmed != "*" {
			explicit = append(explicit, trimmed)
		}
	}

	return explicit
}

func originAllowed(origin string, explicit []string) bool {
	if slices.Contains(explicit, strings.ToLower(strings.TrimSpace(origin))) {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := parsed.Hostname()
	if host == "" {
		host = parsed.Host
	}

	if _, ok := loopbackHosts[strings.ToLower(host)]; ok {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}

	return false
}
