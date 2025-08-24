package matcher

import "testing"

func TestMatch_Equals(t *testing.T) {
	t.Parallel()

	m := Matcher{Equals: map[string]any{"a": 1, "b": map[string]any{"c": "x"}}}

	c := map[string]any{"a": 1, "b": map[string]any{"c": "x", "d": 2}, "e": true}
	if !Match(m, c) {
		t.Fatalf("expected match")
	}
}

func TestMatch_ContainsString(t *testing.T) {
	t.Parallel()

	m := Matcher{Contains: map[string]any{"x": "hello"}}

	c := map[string]any{"x": "well hello there"}
	if !Match(m, c) {
		t.Fatalf("expected match")
	}
}

func TestMatch_MatchesRegex(t *testing.T) {
	t.Parallel()

	m := Matcher{Matches: map[string]string{"name": "^jo.*"}}

	c := map[string]any{"name": "john"}
	if !Match(m, c) {
		t.Fatalf("expected match")
	}
}

func TestMatch_AnyOr(t *testing.T) {
	t.Parallel()

	m := Matcher{Any: []Matcher{
		{Equals: map[string]any{"k": 1}},
		{Equals: map[string]any{"k": 2}},
	}}

	c := map[string]any{"k": 2}
	if !Match(m, c) {
		t.Fatalf("expected match")
	}
}

func TestMatch_IgnoreArrayOrder(t *testing.T) {
	t.Parallel()

	m := Matcher{Equals: map[string]any{"arr": []any{1, 2, 3}}, IgnoreArrayOrder: true}

	c := map[string]any{"arr": []any{3, 2, 1}}
	if !Match(m, c) {
		t.Fatalf("expected match")
	}
}
