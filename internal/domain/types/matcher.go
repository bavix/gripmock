package types

// Matcher represents the unified matcher semantics.
// AND across equals/contains/matches; OR via Any; IgnoreArrayOrder compares arrays as sets.
type Matcher struct {
	Equals           map[string]any `json:"equals,omitempty"`
	Contains         map[string]any `json:"contains,omitempty"`
	Matches          map[string]any `json:"matches,omitempty"`
	Any              []Matcher      `json:"any,omitempty"`
	IgnoreArrayOrder bool           `json:"ignoreArrayOrder,omitempty"`
}
