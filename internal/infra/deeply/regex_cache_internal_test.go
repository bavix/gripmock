package deeply

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnchoredPrefix(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		pattern string
		want    string
	}{
		"plain literal":        {`^user-01`, "user-01"},
		"fully anchored":       {`^user-01$`, "user-01"},
		"stops at class":       {`^user-0[12]$`, "user-0"},
		"stops at dot":         {`^user.*`, "user"},
		"stops at group":       {`^user(a|b)`, "user"},
		"star drops last rune": {`^users*`, "user"},
		"plus keeps last rune": {`^users+`, "users"},
		"question drops last":  {`^users?`, "user"},
		"repeat drops last":    {`^users{2}`, "user"},
		"unanchored":           {`user-01`, ""},
		"empty":                {``, ""},
		"anchor only":          {`^`, ""},
		"anchor then meta":     {`^(?i)user`, ""},
		"anchor then star":     {`^a*`, ""},
		"escape":               {`^user\d`, "user"},
		"multibyte":            {`^привет`, "привет"},
		"multibyte quantified": {`^приветы?`, "привет"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, anchoredPrefix(tc.pattern))
		})
	}
}

// TestCanMatchIsSound is the property the prescreen has to hold: it may only
// reject subjects the real engine would reject too.
func TestCanMatchIsSound(t *testing.T) {
	t.Parallel()

	patterns := []string{
		`^user-000001$`, `^user-00000[1]$`, `^user.*`, `^user-\d+$`,
		`user-1`, `^a?bc`, `^(foo|bar)$`, `^привет.*`, `^$`, `.*`,
		`^x{2,3}y`, `^[a-z]+$`, `^\^literal$`, `^tail`,
	}
	subjects := []string{
		"user-000001", "user-000002", "user", "", "abc", "bc", "foo", "bar",
		"привет мир", "xxy", "^literal", "tail end", "a tail", "1", "user-1",
	}

	for _, pattern := range patterns {
		compiled := regexp.MustCompile(pattern)

		for _, subject := range subjects {
			if !canMatch(pattern, subject) {
				require.False(t, compiled.MatchString(subject))
			}
		}
	}
}
