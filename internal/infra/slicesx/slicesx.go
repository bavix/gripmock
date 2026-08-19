// Package slicesx holds the two generic slice helpers the codebase needs on top
// of the standard slices package.
package slicesx

import "slices"

// Uniq returns the elements of s in order, with later duplicates dropped.
func Uniq[T comparable](s []T) []T {
	if len(s) == 0 {
		return s
	}

	seen := make(map[T]struct{}, len(s))
	out := make([]T, 0, len(s))

	for _, v := range s {
		if _, dup := seen[v]; dup {
			continue
		}

		seen[v] = struct{}{}

		out = append(out, v)
	}

	return out
}

// Without returns the elements of s that are not in exclude, preserving order.
func Without[T comparable](s []T, exclude ...T) []T {
	if len(s) == 0 {
		return s
	}

	if len(exclude) == 0 {
		return slices.Clone(s)
	}

	drop := make(map[T]struct{}, len(exclude))
	for _, v := range exclude {
		drop[v] = struct{}{}
	}

	out := make([]T, 0, len(s))

	for _, v := range s {
		if _, skip := drop[v]; skip {
			continue
		}

		out = append(out, v)
	}

	return out
}
