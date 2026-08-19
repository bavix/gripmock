package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMatcherKinds(t *testing.T) {
	t.Parallel()

	require.Nil(t, parseMatcherKinds(""), "empty input → no filter")
	require.Equal(t, []string{"glob"}, parseMatcherKinds("glob"))
	require.Equal(t, []string{"glob", "anyOf"}, parseMatcherKinds("glob,anyOf"))
	require.Equal(t, []string{"equals", "contains"}, parseMatcherKinds(" equals , contains "))
	require.Equal(t, []string{"matches"}, parseMatcherKinds("bogus,matches,xxx"))
	require.Empty(t, parseMatcherKinds("nope,zzz"))
}
