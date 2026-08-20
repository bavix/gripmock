// Package types provides custom JSON types for Gripmock.
package types

import (
	"encoding/json"
	"strings"
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

type Delay string //nolint:recvcheck

func NewDelay(d time.Duration) Delay {
	return Delay(d.String())
}

func (d Delay) IsTemplate() bool {
	return strings.Contains(string(d), "{{")
}

func (d Delay) Parse() (Duration, error) {
	if d == "" || d.IsTemplate() {
		return 0, nil
	}

	parsed, err := time.ParseDuration(string(d))

	return Duration(parsed), err
}

func (d Delay) Static() Duration {
	parsed, _ := d.Parse()

	return parsed
}

func (d *Delay) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		var ns time.Duration
		if err := json.Unmarshal(data, &ns); err != nil {
			return err
		}

		*d = NewDelay(ns)

		return nil
	}

	if _, err := Delay(s).Parse(); err != nil {
		return err
	}

	*d = Delay(s)

	return nil
}
