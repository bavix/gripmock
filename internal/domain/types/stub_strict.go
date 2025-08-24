package types

// StubStrict represents a strictly typed stub with no raw maps.
type StubStrict struct {
	Service          string            `json:"service"`
	Method           string            `json:"method"`
	Priority         int               `json:"priority,omitempty"`
	Times            int               `json:"times,omitempty"`
	Inputs           []Matcher         `json:"inputs,omitempty"`
	Headers          *Matcher          `json:"headers,omitempty"`
	Outputs          []OutputStrict    `json:"outputs"`
	ResponseHeaders  map[string]string `json:"responseHeaders,omitempty"`
	ResponseTrailers map[string]string `json:"responseTrailers,omitempty"`
	ID               string            `json:"id,omitempty"`
}
