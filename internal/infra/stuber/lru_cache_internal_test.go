package stuber

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringHashCache(t *testing.T) {
	t.Parallel()
	// Clear cache before test
	clearStringHashCache()

	// Test initial state
	size, capacity := getStringHashCacheStats()
	require.Equal(t, 0, size)
	require.Equal(t, 10000, capacity)

	// Test caching
	s := newStorage()

	// First call should calculate hash
	hash1 := s.id("test1")
	require.NotZero(t, hash1)

	// Second call should use cache
	hash2 := s.id("test1")
	require.Equal(t, hash1, hash2)

	// Different string should have different hash
	hash3 := s.id("test2")
	require.NotEqual(t, hash1, hash3)

	// Check cache size
	size, _ = getStringHashCacheStats()
	require.GreaterOrEqual(t, size, 2)
}

func TestRegexCache(t *testing.T) {
	t.Parallel()
	// Clear cache before test
	clearRegexCache()

	// Test initial state
	size, capacity := getRegexCacheStats()
	require.Equal(t, 0, size)
	require.Equal(t, 1000, capacity)

	// Test caching
	pattern := "test.*pattern"

	// First call should compile regex
	re1, err := getRegex(pattern)
	require.NoError(t, err)
	require.NotNil(t, re1)

	// Second call should use cache
	re2, err := getRegex(pattern)
	require.NoError(t, err)
	require.Equal(t, re1, re2)

	// Check cache size
	size, _ = getRegexCacheStats()
	require.Equal(t, 1, size)
}

func TestSearchResultCache(t *testing.T) {
	t.Parallel()
	// This test is disabled as we removed search result caching
	// due to complexity of cache key generation for different query contents
	t.Skip("Search result cache removed due to complexity")
}

func TestLRUCacheEviction(t *testing.T) {
	t.Parallel()
	// Test that LRU cache evicts old entries when full

	// Clear all caches
	clearStringHashCache()
	clearRegexCache()

	s := newStorage()

	// Fill string hash cache beyond capacity
	for i := range 10050 {
		s.id("test" + string(rune(i)))
	}

	// Check that cache size is limited
	size, capacity := getStringHashCacheStats()
	require.LessOrEqual(t, size, capacity)
	require.Equal(t, 10000, capacity)
}

func TestCacheConcurrency(t *testing.T) {
	t.Parallel()
	// Test that caches work correctly under concurrent access

	// Clear all caches
	clearStringHashCache()
	clearRegexCache()

	s := newStorage()

	// Test concurrent string hash caching
	done := make(chan bool, 10)

	for i := range 10 {
		go func(id int) {
			for j := range 100 {
				s.id("concurrent" + string(rune(id)) + string(rune(j)))
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}

	// Check that cache is still functional
	size, capacity := getStringHashCacheStats()
	require.LessOrEqual(t, size, capacity)
	require.Positive(t, size)
}
