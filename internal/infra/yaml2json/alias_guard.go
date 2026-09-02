package yaml2json

import (
	"errors"
	"regexp"
	"strings"
)

// ErrAliasExpansion is returned when a document's aliases expand beyond the supported size.
var ErrAliasExpansion = errors.New("yaml alias expansion exceeded max size")

const maxAliasExpansion = 10_000

var anchorDeclPattern = regexp.MustCompile(`&([\w-]+)`)

type anchorScope struct {
	name   string
	indent int
}

type aliasBudget struct {
	cost  map[string]int
	open  []anchorScope
	total int
}

func checkAliasExpansion(data []byte) error {
	budget := newAliasBudget()

	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := leadingWhitespace(line)

		budget.closeScopes(indent)
		budget.declare(line, indent)

		if err := budget.reference(line); err != nil {
			return err
		}
	}

	return nil
}

func newAliasBudget() *aliasBudget {
	return &aliasBudget{
		cost:  make(map[string]int),
		open:  nil,
		total: 1,
	}
}

func (b *aliasBudget) closeScopes(indent int) {
	for len(b.open) > 0 && b.open[len(b.open)-1].indent >= indent {
		b.open = b.open[:len(b.open)-1]
	}
}

func (b *aliasBudget) declare(line string, indent int) {
	for _, match := range anchorDeclPattern.FindAllStringSubmatch(line, -1) {
		b.cost[match[1]] = 1
		b.open = append(b.open, anchorScope{name: match[1], indent: indent})
	}
}

func (b *aliasBudget) reference(line string) error {
	for _, match := range anchorRefPattern.FindAllStringSubmatch(line, -1) {
		cost, known := b.cost[match[1]]
		if !known {
			continue
		}

		for i := range b.open {
			b.cost[b.open[i].name] = saturatingAdd(b.cost[b.open[i].name], cost)
		}

		b.total = saturatingAdd(b.total, cost)
		if b.total > maxAliasExpansion {
			return ErrAliasExpansion
		}
	}

	return nil
}

func saturatingAdd(left, right int) int {
	if right > maxAliasExpansion || left > maxAliasExpansion-right {
		return maxAliasExpansion + 1
	}

	return left + right
}
