package protobundle

import (
	"bufio"
	"math"
	"os"
	"strconv"
	"strings"
)

// syntaxRank returns a numeric priority for a proto file.
// Higher value = more modern syntax.
//
// Ranking:
//
//	proto2          → 0
//	proto3          → 1
//	edition "2023"  → 2023
//	edition "2024"  → 2024
//	...
//	edition "XYZ"   → math.MaxInt (non-numeric → always exceeds MaxEdition)
//
// Files with no explicit syntax declaration are treated as proto2 (rank 0).
func syntaxRank(path string) (int, error) {
	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return 0, err
	}
	defer file.Close() //nolint:errcheck

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}

		if strings.HasPrefix(line, "edition") {
			return parseEdition(line), nil
		}

		if strings.HasPrefix(line, "syntax") {
			return parseSyntax(line), nil
		}

		// First non-comment, non-empty line is not syntax/edition — proto2 by default.
		return 0, nil
	}

	return 0, scanner.Err()
}

// parseSyntax extracts the syntax version from a line like syntax = "proto3".
func parseSyntax(line string) int {
	if strings.Contains(line, `"proto3"`) {
		return 1
	}

	// proto2 or unrecognized.
	return 0
}

// parseEdition extracts the edition year from a line like edition = "2023".
// Non-numeric editions (e.g. "UNSTABLE") return math.MaxInt, ensuring they always exceed MaxEdition.
func parseEdition(line string) int {
	start := strings.IndexByte(line, '"')
	if start == -1 {
		return math.MaxInt
	}

	end := strings.IndexByte(line[start+1:], '"')
	if end == -1 {
		return math.MaxInt
	}

	year, err := strconv.Atoi(line[start+1 : start+1+end])
	if err != nil {
		return math.MaxInt
	}

	return year
}
