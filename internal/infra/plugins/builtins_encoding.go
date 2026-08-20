package plugins

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func encodingFuncs() map[string]any {
	conv := conversionFuncs{}
	b64 := base64Funcs{}
	u := uuidHelper{}

	return map[string]any{
		"bytes":         conv.StringToBytes,
		"string2base64": b64.StringToBase64,
		"bytes2base64":  b64.BytesToBase64,
		"uuid2base64":   u.UUIDToBase64,
		"uuid2bytes":    u.UUIDToBytes,
		"uuid2int64":    u.UUIDToInt64,
	}
}

type conversionFuncs struct{}

func (conversionFuncs) StringToBytes(s string) []byte { return []byte(s) }

type base64Funcs struct{}

func (base64Funcs) StringToBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
func (base64Funcs) BytesToBase64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

type uuidHelper struct{}

func (uuidHelper) UUIDToBase64(id string) (string, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(parsed[:]), nil
}

func (uuidHelper) UUIDToBytes(id string) ([]byte, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return parsed[:], nil
}

func (uuidHelper) UUIDToInt64(id string) (string, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return "", err
	}

	bytes := parsed[:]

	//nolint:gosec
	high := int64(binary.LittleEndian.Uint64(bytes[:8]))
	//nolint:gosec
	low := int64(binary.LittleEndian.Uint64(bytes[8:]))

	return fmt.Sprintf(`{"high":%d,"low":%d}`, high, low), nil
}

func timeFuncs() map[string]any {
	return map[string]any{
		"now":    time.Now,
		"unix":   time.Time.Unix,
		"format": time.Time.Format,
		"duration": func(v any, unit ...any) string {
			f, ok := convertToFloat64(v)
			if !ok {
				return "0s"
			}

			suffix := "ms"

			if len(unit) > 0 {
				s, ok := unit[0].(string)
				if !ok {
					return "0s"
				}

				suffix = s
			}

			scale := durationUnit(suffix)
			if scale == 0 {
				return "0s"
			}

			return time.Duration(f * float64(scale)).String()
		},
	}
}

func durationUnit(suffix string) time.Duration {
	switch suffix {
	case "ns":
		return time.Nanosecond
	case "us", "µs":
		return time.Microsecond
	case "ms":
		return time.Millisecond
	case "s":
		return time.Second
	case "m":
		return time.Minute
	case "h":
		return time.Hour
	default:
		return 0
	}
}

func uuidFuncMap() map[string]any {
	return map[string]any{
		"uuid": func() string {
			return uuid.New().String()
		},
	}
}
