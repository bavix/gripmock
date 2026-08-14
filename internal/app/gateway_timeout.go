package app

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	headerConnectTimeoutMs = "Connect-Timeout-Ms"
	headerGRPCTimeout      = "Grpc-Timeout"

	connectTimeoutMaxDigits = 10
	grpcTimeoutMaxDigits    = 8

	// A gRPC timeout is at least one digit plus the unit suffix.
	grpcTimeoutMinLen = 2
)

var errTimeoutMalformed = errors.New("malformed timeout header")

//nolint:gochecknoglobals
var grpcTimeoutUnits = map[byte]time.Duration{
	'H': time.Hour,
	'M': time.Minute,
	'S': time.Second,
	'm': time.Millisecond,
	'u': time.Microsecond,
	'n': time.Nanosecond,
}

func requestTimeout(hdr http.Header) (time.Duration, bool, error) {
	if v := strings.TrimSpace(hdr.Get(headerConnectTimeoutMs)); v != "" {
		d, err := parseConnectTimeout(v)

		return d, err == nil, err
	}

	if v := strings.TrimSpace(hdr.Get(headerGRPCTimeout)); v != "" {
		d, err := parseGRPCTimeout(v)

		return d, err == nil, err
	}

	return 0, false, nil
}

func parseConnectTimeout(raw string) (time.Duration, error) {
	if len(raw) > connectTimeoutMaxDigits {
		return 0, errTimeoutMalformed
	}

	ms, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || ms < 0 {
		return 0, errTimeoutMalformed
	}

	return time.Duration(ms) * time.Millisecond, nil
}

func parseGRPCTimeout(raw string) (time.Duration, error) {
	if len(raw) < grpcTimeoutMinLen {
		return 0, errTimeoutMalformed
	}

	unit, ok := grpcTimeoutUnits[raw[len(raw)-1]]
	if !ok {
		return 0, errTimeoutMalformed
	}

	digits := raw[:len(raw)-1]
	if len(digits) > grpcTimeoutMaxDigits {
		return 0, errTimeoutMalformed
	}

	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || n < 0 {
		return 0, errTimeoutMalformed
	}

	return time.Duration(n) * unit, nil
}
