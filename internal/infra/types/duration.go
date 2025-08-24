// Package types provides custom JSON types for Gripmock.
package types

import (
	"encoding/json"
	"time"
)

// Duration is a custom type alias for time.Duration that provides
// JSON marshaling/unmarshaling support for string values like "100ms".
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler interface.
func (d *Duration) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string

	err := json.Unmarshal(data, &s)
	if err == nil {
		duration, err := time.ParseDuration(s)
		if err != nil {
			return err
		}

		*d = Duration(duration)

		return nil
	}

	return json.Unmarshal(data, (*time.Duration)(d))
}

// MarshalJSON implements json.Marshaler interface.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}
