package sdk

import (
	"maps"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// Matcher describes how to match a gRPC request field or header.
// Same type works for both payload fields and gRPC metadata headers.
// Created via Equals, Contains, Matches, Glob; composed via AnyOf, And.
//
// Use Match(sdk.Contains(...)) for payload matching and
// WithHeader(sdk.Contains(...)) for gRPC metadata matching.
type Matcher struct {
	equals   map[string]any
	contains map[string]any
	matches  map[string]any
	glob     map[string]any
	anyOf    []Matcher
	noOrder  bool
}

func Equals(key string, value any) Matcher {
	return Matcher{equals: map[string]any{key: value}}
}

func Contains(key string, value any) Matcher {
	return Matcher{contains: map[string]any{key: value}}
}

func Matches(key, pattern string) Matcher {
	return Matcher{matches: map[string]any{key: pattern}}
}

func Glob(key, pattern string) Matcher {
	return Matcher{glob: map[string]any{key: pattern}}
}

// AnyOf returns a Matcher that passes when at least one alternative matches (OR logic).
func AnyOf(matchers ...Matcher) Matcher {
	return Matcher{anyOf: matchers}
}

// And returns a Matcher that passes when all given matchers match (AND logic).
func And(matchers ...Matcher) Matcher {
	out := Matcher{}
	for _, m := range matchers {
		out.equals = mergeStrAny(out.equals, m.equals)
		out.contains = mergeStrAny(out.contains, m.contains)
		out.matches = mergeStrAny(out.matches, m.matches)

		out.glob = mergeStrAny(out.glob, m.glob)
		if m.noOrder {
			out.noOrder = true
		}

		out.anyOf = append(out.anyOf, m.anyOf...)
	}

	return out
}

func IgnoreArrayOrder(m Matcher) Matcher {
	m.noOrder = true

	return m
}

// compilePayload converts the matcher to stuber.InputData for payload matching.
func (m Matcher) compilePayload() stuber.InputData {
	return stuber.InputData{
		Equals:           m.equals,
		Contains:         m.contains,
		Matches:          m.matches,
		Glob:             m.glob,
		IgnoreArrayOrder: m.noOrder,
		AnyOf:            compileAnyOf(m.anyOf),
	}
}

// compileHeader converts the matcher to stuber.InputHeader for header matching.
func (m Matcher) compileHeader() stuber.InputHeader {
	return stuber.InputHeader{
		Equals:   m.equals,
		Contains: m.contains,
		Matches:  m.matches,
		Glob:     m.glob,
		AnyOf:    compileHeaderAnyOf(m.anyOf),
	}
}

func compileAnyOf(matchers []Matcher) []stuber.AnyOfElement {
	out := make([]stuber.AnyOfElement, len(matchers))
	for i, m := range matchers {
		out[i] = stuber.AnyOfElement{
			IgnoreArrayOrder: m.noOrder,
			Equals:           m.equals,
			Contains:         m.contains,
			Matches:          m.matches,
			Glob:             m.glob,
		}
	}

	return out
}

func compileHeaderAnyOf(matchers []Matcher) []stuber.AnyOfHeaderElement {
	out := make([]stuber.AnyOfHeaderElement, len(matchers))
	for i, m := range matchers {
		out[i] = stuber.AnyOfHeaderElement{
			Equals:   m.equals,
			Contains: m.contains,
			Matches:  m.matches,
			Glob:     m.glob,
		}
	}

	return out
}

func mergeStrAny(a, b map[string]any) map[string]any {
	if len(b) == 0 {
		return a
	}

	if a == nil {
		return b
	}

	maps.Copy(a, b)

	return a
}
