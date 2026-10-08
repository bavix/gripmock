package protoset

import (
	"net/url"
	"strings"
)

const redactedValue = "REDACTED"

const redactedSource = "[unparsable source redacted]"

//nolint:gochecknoglobals
var credentialQueryKeys = map[string]struct{}{
	"auth":          {},
	"authorization": {},
	"bearer":        {},
	"passwd":        {},
	"password":      {},
	"pwd":           {},
	"sig":           {},
	"signature":     {},
}

//nolint:gochecknoglobals
var credentialQueryHints = []string{
	"apikey",
	"api_key",
	"credential",
	"password",
	"secret",
	"token",
}

// RedactURL masks credential-carrying parts of a source or proxy URL.
func RedactURL(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return redactedSource
	}

	if parsed.RawQuery == "" && parsed.User == nil {
		return raw
	}

	values, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		parsed.RawQuery = redactedValue
	} else if redactQueryValues(values) {
		parsed.RawQuery = values.Encode()
	}

	return parsed.Redacted()
}

// RedactURLs applies RedactURL to every entry of a source list.
func RedactURLs(raws []string) []string {
	if len(raws) == 0 {
		return nil
	}

	out := make([]string, len(raws))
	for i, raw := range raws {
		out[i] = RedactURL(raw)
	}

	return out
}

func redactQueryValues(values url.Values) bool {
	redacted := false

	for key, vals := range values {
		if !isCredentialParam(key) {
			continue
		}

		for i := range vals {
			vals[i] = redactedValue
		}

		redacted = true
	}

	return redacted
}

func isCredentialParam(key string) bool {
	lower := strings.ToLower(key)

	if _, ok := credentialQueryKeys[lower]; ok {
		return true
	}

	for _, hint := range credentialQueryHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}

	return false
}
