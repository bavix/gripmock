package cmd

import (
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

type ansiStyle struct {
	color string
	bold  bool
}

func newStyle(color string) ansiStyle { return ansiStyle{color: color} }

func (s ansiStyle) Bold() ansiStyle {
	s.bold = true

	return s
}

func (s ansiStyle) Render(text string) string {
	if !colorEnabled() {
		return text
	}

	return s.wrap(text)
}

func (s ansiStyle) wrap(text string) string {
	var codes []string

	if s.bold {
		codes = append(codes, "1")
	}

	if s.color != "" {
		codes = append(codes, "38;5;"+s.color)
	}

	if len(codes) == 0 {
		return text
	}

	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

func colorEnabled() bool {
	if _, off := os.LookupEnv("NO_COLOR"); off {
		return false
	}

	if term := os.Getenv("TERM"); term == "dumb" {
		return false
	}

	fd := os.Stdout.Fd()

	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
