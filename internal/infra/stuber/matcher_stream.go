package stuber

import "github.com/bavix/gripmock/v3/internal/infra/deeply"

// streamItemMatches checks if a single query item matches the stub item matchers.
//
//nolint:cyclop
func streamItemMatches(stubItem InputData, queryItem map[string]any) bool {
	if len(stubItem.Equals) == 0 && len(stubItem.Contains) == 0 && len(stubItem.Matches) == 0 && len(stubItem.Glob) == 0 {
		return false
	}

	return (len(stubItem.Equals) == 0 || equals(stubItem.Equals, queryItem, stubItem.IgnoreArrayOrder)) &&
		(len(stubItem.Contains) == 0 || contains(stubItem.Contains, queryItem)) &&
		(len(stubItem.Matches) == 0 || matches(stubItem.Matches, queryItem)) &&
		(len(stubItem.Glob) == 0 || globMatch(stubItem.Glob, queryItem))
}

// matchStreamElements checks if the query stream matches the stub stream.
func matchStreamElements(queryStream []map[string]any, stubStream []InputData) bool {
	n := len(queryStream)

	// STRICT: If query stream is empty but stub expects data, no match
	if n == 0 && len(stubStream) > 0 {
		return false
	}

	// Broadcast mode: stub has a single pattern — every message must match it.
	// This covers the case where a stub uses inputs:[{pattern}] to match any
	// number of streaming messages (e.g. PROD_789 stub with 2-message stream).
	if len(stubStream) == 1 && n > 1 {
		pattern := stubStream[0]

		for _, msg := range queryStream {
			if !streamItemMatches(pattern, msg) {
				return false
			}
		}

		return true
	}

	if n != len(stubStream) {
		return false
	}

	for i, msg := range queryStream {
		if !streamItemMatches(stubStream[i], msg) {
			return false
		}
	}

	return true
}

// rankStreamElements ranks the match between query stream and stub stream.
func rankStreamElements(queryStream []map[string]any, stubStream []InputData) float64 {
	n := len(queryStream)

	if n == 0 && len(stubStream) > 0 {
		return 0
	}

	// Broadcast mode: single pattern matched against all messages.
	if len(stubStream) == 1 && n > 1 {
		totalRank, perfectMatches := computeStreamElementRanksBroadcast(queryStream, stubStream[0])
		avgRank := totalRank / float64(n)
		perfectMatchBonus := float64(perfectMatches) * 1000.0      //nolint:mnd
		specificityBonus := countStreamMatchers(stubStream) * 50.0 //nolint:mnd

		return avgRank + perfectMatchBonus + specificityBonus
	}

	if n != len(stubStream) {
		return 0
	}

	totalRank, perfectMatches := computeStreamElementRanks(queryStream, stubStream)
	lengthBonus := float64(n) * 10.0                      //nolint:mnd
	perfectMatchBonus := float64(perfectMatches) * 1000.0 //nolint:mnd

	completeMatchBonus := 0.0
	if perfectMatches == n && n > 0 {
		completeMatchBonus = 10000.0
	}

	specificityBonus := countStreamMatchers(stubStream) * 50.0 //nolint:mnd

	return totalRank + lengthBonus + perfectMatchBonus + completeMatchBonus + specificityBonus
}

func computeStreamElementRanksBroadcast(
	queryStream []map[string]any,
	stubPattern InputData,
) (float64, int) {
	var totalRank float64

	var perfectMatches int

	for _, queryItem := range queryStream {
		equalsRank := 0.0
		if len(stubPattern.Equals) > 0 && equals(stubPattern.Equals, queryItem, stubPattern.IgnoreArrayOrder) {
			equalsRank = 1.0
		}

		containsRank := deeply.RankMatch(stubPattern.Contains, queryItem)
		matchesRank := deeply.RankMatch(stubPattern.Matches, queryItem)
		elementRank := equalsRank*100.0 + containsRank*0.1 + matchesRank*0.1 //nolint:mnd
		totalRank += elementRank

		if equalsRank > 0.99 { //nolint:mnd
			perfectMatches++
		}
	}

	return totalRank, perfectMatches
}

func computeStreamElementRanks(
	queryStream []map[string]any,
	stubStream []InputData,
) (float64, int) {
	var totalRank float64

	var perfectMatches int

	for i, queryItem := range queryStream {
		stubItem := stubStream[i]

		equalsRank := 0.0
		if len(stubItem.Equals) > 0 && equals(stubItem.Equals, queryItem, stubItem.IgnoreArrayOrder) {
			equalsRank = 1.0
		}

		containsRank := deeply.RankMatch(stubItem.Contains, queryItem)
		matchesRank := deeply.RankMatch(stubItem.Matches, queryItem)
		elementRank := equalsRank*100.0 + containsRank*0.1 + matchesRank*0.1 //nolint:mnd
		totalRank += elementRank

		if equalsRank > 0.99 { //nolint:mnd
			perfectMatches++
		}
	}

	return totalRank, perfectMatches
}

func countStreamMatchers(stubStream []InputData) float64 {
	var total int

	for _, stubItem := range stubStream {
		total += countNonNil(stubItem.Equals) + countNonNil(stubItem.Contains) + countNonNil(stubItem.Matches)
	}

	return float64(total)
}

func countNonNil(m map[string]any) int {
	var n int

	for _, v := range m {
		if v != nil {
			n++
		}
	}

	return n
}
