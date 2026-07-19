package stuber

import "github.com/bavix/gripmock/v3/internal/infra/deeply"

// matchHeaders checks if query headers match stub headers.
//
//nolint:cyclop
func matchHeaders(queryHeaders map[string]any, stubHeaders InputHeader) bool {
	if !equals(stubHeaders.Equals, queryHeaders, false) ||
		!contains(stubHeaders.Contains, queryHeaders) ||
		!matches(stubHeaders.Matches, queryHeaders) ||
		!globMatch(stubHeaders.Glob, queryHeaders) {
		return false
	}

	if len(stubHeaders.AnyOf) == 0 {
		return true
	}

	for i := range stubHeaders.AnyOf {
		alt := &stubHeaders.AnyOf[i]
		if equals(alt.Equals, queryHeaders, false) &&
			contains(alt.Contains, queryHeaders) &&
			matches(alt.Matches, queryHeaders) &&
			globMatch(alt.Glob, queryHeaders) {
			return true
		}
	}

	return false
}

// matchInput checks if query data matches stub input.
//
//nolint:cyclop
func matchInput(queryData map[string]any, stubInput InputData) bool {
	if !equals(stubInput.Equals, queryData, stubInput.IgnoreArrayOrder) ||
		!contains(stubInput.Contains, queryData) ||
		!matches(stubInput.Matches, queryData) ||
		!globMatch(stubInput.Glob, queryData) {
		return false
	}

	if len(stubInput.AnyOf) == 0 {
		return true
	}

	for i := range stubInput.AnyOf {
		alt := &stubInput.AnyOf[i]
		if equals(alt.Equals, queryData, alt.IgnoreArrayOrder) &&
			contains(alt.Contains, queryData) &&
			matches(alt.Matches, queryData) &&
			globMatch(alt.Glob, queryData) {
			return true
		}
	}

	return false
}

// rankHeaders ranks query headers against stub headers.
func rankHeaders(queryHeaders map[string]any, stubHeaders InputHeader) float64 {
	if stubHeaders.Len() == 0 {
		return 0
	}

	base := deeply.RankMatch(stubHeaders.Equals, queryHeaders) +
		deeply.RankMatch(stubHeaders.Contains, queryHeaders) +
		deeply.RankMatch(stubHeaders.Matches, queryHeaders) +
		rankGlob(stubHeaders.Glob, queryHeaders)

	if len(stubHeaders.AnyOf) == 0 {
		return base
	}

	best := 0.0

	for i := range stubHeaders.AnyOf {
		alt := &stubHeaders.AnyOf[i]

		r := deeply.RankMatch(alt.Equals, queryHeaders) +
			deeply.RankMatch(alt.Contains, queryHeaders) +
			deeply.RankMatch(alt.Matches, queryHeaders) +
			rankGlob(alt.Glob, queryHeaders)
		if r > best {
			best = r
		}
	}

	return base + best
}

// rankInput ranks query data against stub input.
func rankInput(queryData map[string]any, stubInput InputData) float64 {
	base := deeply.RankMatch(stubInput.Equals, queryData) +
		deeply.RankMatch(stubInput.Contains, queryData) +
		deeply.RankMatch(stubInput.Matches, queryData) +
		rankGlob(stubInput.Glob, queryData)

	if len(stubInput.AnyOf) == 0 {
		return base
	}

	best := 0.0

	for i := range stubInput.AnyOf {
		alt := &stubInput.AnyOf[i]

		r := deeply.RankMatch(alt.Equals, queryData) +
			deeply.RankMatch(alt.Contains, queryData) +
			deeply.RankMatch(alt.Matches, queryData) +
			rankGlob(alt.Glob, queryData)
		if r > best {
			best = r
		}
	}

	return base + best
}
