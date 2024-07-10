package muxmiddleware

import (
	"net"
	"net/http"
	"strings"
)

// getIP returns the IP address from the request headers.
// It returns the IP address from the X-Forwarded-For header if it exists,
// otherwise it returns the IP address from the RemoteAddr field.
func getIP(r *http.Request) (net.IP, error) {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[len(ips)-1])
			if parsedIP := net.ParseIP(ip); parsedIP != nil {
				return parsedIP, nil
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, err
	}

	return net.ParseIP(host), nil
}
