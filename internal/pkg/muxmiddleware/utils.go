package muxmiddleware

import (
	"net"
	"net/http"
	"strings"
)

func getIP(r *http.Request) (net.IP, error) {
	ips := r.Header.Get("X-Forwarded-For")
	splitIps := strings.Split(ips, ",")

	if len(splitIps) > 0 {
		netIP := net.ParseIP(splitIps[len(splitIps)-1])

		if netIP != nil {
			return netIP, nil
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, err
	}

	return net.ParseIP(ip), nil
}
