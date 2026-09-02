package deeply

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanMatchNeverRejectsARealMatch(t *testing.T) {
	t.Parallel()

	patterns := []string{
		"^foo|bar",
		"^foo$|^bar$",
		"^ACTIVE$|^PENDING$",
		"^abc|xyz",
		"^abc(d|e)",
		"^abc(d|e)|xyz",
		"^a[b|c]d",
		`^a\|b`,
		"^prefix",
		"^prefix.*",
		"^[0-9]+$",
		"suffix$",
		"^(a|b)c",
	}

	subjects := []string{
		"foo", "bar", "ACTIVE", "PENDING", "INACTIVE", "abc", "xyz",
		"abcd", "abce", "abd", "acd", "a|b", "prefix", "prefixed",
		"12345", "with suffix", "ac", "bc", "", "zzz",
	}

	for _, pattern := range patterns {
		compiled, err := regexp.Compile(pattern)
		require.NoError(t, err, "pattern %q", pattern)

		for _, subject := range subjects {
			if compiled.MatchString(subject) {
				require.True(
					t,
					canMatch(pattern, subject),
					"pre-filter rejected a real match: pattern %q subject %q",
					pattern, subject,
				)
			}
		}
	}
}

func TestHasTopLevelAlternation(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"^foo|bar":       true,
		"^foo$|^bar$":    true,
		"^abc(d|e)":      false,
		"^abc(d|e)|xyz":  true,
		"^a[b|c]d":       false,
		`^a\|b`:          false,
		"^plain":         false,
		"^((a|b)|(c|d))": false,
		"":               false,
	}

	for pattern, expected := range cases {
		require.Equal(t, expected, hasTopLevelAlternation(pattern), "pattern %q", pattern)
	}
}

func TestAnchoredPrefixStillNarrowsSimplePatterns(t *testing.T) {
	t.Parallel()

	require.Equal(t, "prefix", anchoredPrefix("^prefix"))
	require.Equal(t, "abc", anchoredPrefix("^abc(d|e)"))
	require.Empty(t, anchoredPrefix("^abc(d|e)|xyz"))
	require.Empty(t, anchoredPrefix("no anchor"))
}
