package types

import (
	"encoding"
	"fmt"
	"strconv"
	"strings"
)

const (
	kibibyte = int64(1024)
	mebibyte = kibibyte * 1024
	gibibyte = mebibyte * 1024
)

// ByteSize is an env-decodable size with K|M|G suffix support (decimal kilobytes/mebibytes/gibibytes).
// Examples: "128K", "64M", "1G", or plain integer bytes like "262144".
type ByteSize struct {
	Bytes int64
}

// UnmarshalText implements encoding.TextUnmarshaler for ByteSize.
func (b *ByteSize) UnmarshalText(text []byte) error {
	// Support both integer bytes and values with K/M/G suffixes.
	raw := strings.TrimSpace(strings.ToUpper(string(text)))
	if raw == "" {
		b.Bytes = 0

		return nil
	}

	mult := int64(1)

	switch {
	case strings.HasSuffix(raw, "K"):
		mult = kibibyte
		raw = strings.TrimSuffix(raw, "K")
	case strings.HasSuffix(raw, "M"):
		mult = mebibyte
		raw = strings.TrimSuffix(raw, "M")
	case strings.HasSuffix(raw, "G"):
		mult = gibibyte
		raw = strings.TrimSuffix(raw, "G")
	}

	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid byte size: %w", err)
	}

	b.Bytes = n * mult

	return nil
}

// Int64 returns the value in bytes.
func (b *ByteSize) Int64() int64 { return b.Bytes }

var _ encoding.TextUnmarshaler = (*ByteSize)(nil)

