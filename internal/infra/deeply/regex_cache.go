package deeply

import (
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"
)

const regexCacheSize = 8192

const regexMeta = `\^$.+*?()|[]{}`

//nolint:gochecknoglobals
var regexMetaTable = buildRegexMetaTable()

func buildRegexMetaTable() [256]bool {
	var table [256]bool

	for i := range len(regexMeta) {
		table[regexMeta[i]] = true
	}

	return table
}

func hasRegexMeta(pattern string) bool {
	for i := range len(pattern) {
		if regexMetaTable[pattern[i]] {
			return true
		}
	}

	return false
}

func canMatch(pattern, subject string) bool {
	prefix := anchoredPrefix(pattern)

	return prefix == "" || strings.HasPrefix(subject, prefix)
}

func anchoredPrefix(pattern string) string {
	if pattern == "" || pattern[0] != '^' {
		return ""
	}

	end := 1
	for end < len(pattern) && strings.IndexByte(regexMeta, pattern[end]) < 0 {
		end++
	}

	if end < len(pattern) && strings.IndexByte("*?{", pattern[end]) >= 0 {
		_, size := utf8.DecodeLastRuneInString(pattern[1:end])
		end -= size
	}

	if end <= 1 {
		return ""
	}

	return pattern[1:end]
}

//nolint:gochecknoglobals
var (
	regexCache  sync.Map
	regexCached atomic.Int64
)

func compileRegex(pattern string) (*regexp.Regexp, error) {
	if cached, ok := regexCache.Load(pattern); ok {
		compiled, _ := cached.(*regexp.Regexp)

		return compiled, nil
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	if regexCached.Load() < regexCacheSize {
		if _, loaded := regexCache.LoadOrStore(pattern, compiled); !loaded {
			regexCached.Add(1)
		}
	}

	return compiled, nil
}
