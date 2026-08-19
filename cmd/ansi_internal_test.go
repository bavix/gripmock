package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnsiStyleRenderEmitsCodes(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	require.Equal(t, "\x1b[1;38;5;212mX\x1b[0m", newStyle("212").Bold().wrap("X"))
	require.Equal(t, "\x1b[38;5;244mX\x1b[0m", newStyle("244").wrap("X"))
	require.Equal(t, "X", ansiStyle{}.wrap("X"))
}

func TestAnsiStyleRenderIsPlainWithoutTerminal(t *testing.T) {
	t.Parallel()

	require.Equal(t, "X", newStyle("212").Bold().Render("X"))
}

func TestColorDisabledByNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	require.False(t, colorEnabled())
}

func TestColorDisabledByDumbTerm(t *testing.T) {
	t.Setenv("TERM", "dumb")
	require.False(t, colorEnabled())
}
