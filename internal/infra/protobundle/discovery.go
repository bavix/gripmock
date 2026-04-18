package protobundle

import (
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

const DefaultMaxEdition = 2024

// DiscoverParams configures proto file discovery across roots.
type DiscoverParams struct {
	Roots      []string // import root dirs, priority order
	Include    []string // glob patterns (default: **/*.proto)
	Exclude    []string // glob patterns to skip
	MaxEdition int      // highest supported edition year; 0 → DefaultMaxEdition
}

// DiscoverResult holds discovered proto files ready for compilation.
type DiscoverResult struct {
	// Files maps relative path -> absolute path.
	// On conflict: file with newer syntax wins (edition "2024" > edition "2023" > ... > proto3 > proto2);
	// among equal syntax, first root wins.
	Files map[string]string
	// Skipped lists relative paths that were shadowed by deduplication.
	Skipped []string
	// UnsupportedEdition lists relative paths of files skipped because their edition exceeds MaxEdition.
	UnsupportedEdition []string
}

// Sorted returns the file relative paths in sorted order.
func (r *DiscoverResult) Sorted() []string {
	keys := make([]string, 0, len(r.Files))
	for k := range r.Files {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

// Discover walks roots and returns deduplicated proto files.
// Only .proto files are collected; all other content is ignored.
// Files with edition above MaxEdition (default DefaultMaxEdition) are skipped.
func Discover(params DiscoverParams) (*DiscoverResult, error) {
	if len(params.Roots) == 0 {
		return nil, errors.New("at least one root is required")
	}

	include := params.Include
	if len(include) == 0 {
		include = []string{"**/*.proto"}
	}

	maxEdition := params.MaxEdition
	if maxEdition == 0 {
		maxEdition = DefaultMaxEdition
	}

	ctx := &discoveryContext{
		include:    include,
		exclude:    params.Exclude,
		maxEdition: maxEdition,
		result: &DiscoverResult{
			Files: make(map[string]string),
		},
		ranks: make(map[string]int),
	}

	for _, root := range params.Roots {
		if err := ctx.walkRoot(root); err != nil {
			return nil, err
		}
	}

	return ctx.result, nil
}

// discoveryContext holds mutable state during file discovery.
type discoveryContext struct {
	include    []string
	exclude    []string
	maxEdition int
	result     *DiscoverResult
	ranks      map[string]int
}

func (c *discoveryContext) walkRoot(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve root: %s", root)
	}

	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return errors.Wrapf(err, "walk error at %s", path)
		}

		if entry.IsDir() || filepath.Ext(path) != ".proto" {
			return nil
		}

		return c.processFile(absRoot, path)
	})
	if err != nil {
		return errors.Wrapf(err, "failed to walk root: %s", absRoot)
	}

	return nil
}

func (c *discoveryContext) processFile(absRoot, path string) error {
	relPath, err := filepath.Rel(absRoot, path)
	if err != nil {
		return errors.Wrapf(err, "failed to compute relative path for %s", path)
	}

	// Normalize to forward slashes for consistent matching.
	relPath = filepath.ToSlash(relPath)

	if !matchesAny(relPath, c.include) || matchesAny(relPath, c.exclude) {
		return nil
	}

	rank := c.readRank(path)

	// Skip files whose edition exceeds the compiler's supported maximum.
	// Edition ranks are year numbers (2023, 2024, ...); proto2=0, proto3=1.
	if rank > c.maxEdition {
		log.Warn().
			Str("file", relPath).
			Int("edition", rank).
			Int("max_supported", c.maxEdition).
			Msg("skipping file with unsupported edition")

		c.result.UnsupportedEdition = append(c.result.UnsupportedEdition, relPath)

		return nil
	}

	existing, exists := c.result.Files[relPath]
	if !exists {
		c.result.Files[relPath] = path
		c.ranks[relPath] = rank

		return nil
	}

	// Conflict: prefer newer syntax.
	if rank > c.ranks[relPath] {
		c.result.Skipped = append(c.result.Skipped, relPath+" (from "+existing+")")
		c.result.Files[relPath] = path
		c.ranks[relPath] = rank
	} else {
		c.result.Skipped = append(c.result.Skipped, relPath+" (from "+path+")")
	}

	return nil
}

func (c *discoveryContext) readRank(path string) int {
	rank, err := syntaxRank(path)
	if err != nil {
		log.Warn().Err(err).Str("file", path).Msg("failed to determine syntax rank, treating as proto2")
	}

	return rank
}

// matchesAny checks if path matches any of the glob patterns.
func matchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := doublestar.Match(pattern, path)
		if err != nil {
			continue
		}

		if matched {
			return true
		}
	}

	return false
}
