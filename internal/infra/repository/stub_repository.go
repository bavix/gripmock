package repository

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/internal/app/port"
	domain "github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// StubRepository provides a unified interface for stub storage and management.
type StubRepository struct {
	// Core storage
	budgerigar *stuber.Budgerigar

	// File tracking
	mapIDsByFile map[string][]uuid.UUID
	uniqueIDs    map[uuid.UUID]struct{}
	muUniqueIDs  sync.RWMutex

	// Loading state
	loaded       atomic.Bool
	loadComplete chan struct{}

	// Cache for file content
	fileCache map[string]FileCacheEntry
	muCache   sync.RWMutex
}

// FileCacheEntry represents cached file content.
type FileCacheEntry struct {
	Content    map[string]any
	StubIDs    []uuid.UUID
	LastUpdate int64
}

// LoadFromDirectory loads stubs from a directory and watches for changes.
func (r *StubRepository) LoadFromDirectory(ctx context.Context, dirPath string) error {
	if dirPath == "" {
		r.loaded.Store(true)
		close(r.loadComplete)

		return nil
	}

	zerolog.Ctx(ctx).Info().Msg("Loading stubs from directory")

	// Load initial stubs
	// Implementation placeholder

	r.loaded.Store(true)
	close(r.loadComplete)

	// Start watching for changes
	// Implementation placeholder
	return nil
}

// WaitForLoad waits for the initial loading to complete.
func (r *StubRepository) WaitForLoad(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-r.loadComplete:
		return
	}
}

// IsLoaded returns whether the repository has finished initial loading.
func (r *StubRepository) IsLoaded() bool {
	return r.loaded.Load()
}

// Create adds a new stub to the repository.
func (r *StubRepository) Create(ctx context.Context, stub domain.Stub) (domain.Stub, error) {
	if stub.ID == "" {
		stub.ID = uuid.New().String()
	}

	// Convert to stuber.Stub and add to budgerigar
	s := r.convertToStuberStub(stub)
	ids := r.budgerigar.PutMany(s)

	// Track the stub
	r.trackStub(ids[0], stub.ID)

	return stub, nil
}

// Update updates an existing stub.
func (r *StubRepository) Update(ctx context.Context, id string, stub domain.Stub) (domain.Stub, error) {
	// Check if stub exists
	if uid, err := uuid.Parse(id); err == nil {
		if r.budgerigar.FindByID(uid) == nil {
			return domain.Stub{}, nil
		}
	}

	stub.ID = id
	s := r.convertToStuberStub(stub)
	ids := r.budgerigar.PutMany(s)

	// Update tracking
	if len(ids) > 0 {
		r.trackStub(ids[0], stub.ID)
	}

	return stub, nil
}

// Delete removes a stub by ID.
func (r *StubRepository) Delete(ctx context.Context, id string) error {
	if uid, err := uuid.Parse(id); err == nil {
		r.budgerigar.DeleteByID(uid)
		r.untrackStub(uid)
	}

	return nil
}

// DeleteMany removes multiple stubs by IDs.
func (r *StubRepository) DeleteMany(ctx context.Context, ids []string) error {
	for _, id := range ids {
		_ = r.Delete(ctx, id)
	}

	return nil
}

// GetByID retrieves a stub by ID.
func (r *StubRepository) GetByID(ctx context.Context, id string) (domain.Stub, bool) {
	if uid, err := uuid.Parse(id); err == nil {
		if stub := r.budgerigar.FindByID(uid); stub != nil {
			return r.convertFromStuberStub(stub), true
		}
	}

	return domain.Stub{}, false
}

// List retrieves stubs with filtering, sorting, and pagination.
func (r *StubRepository) List(
	ctx context.Context,
	filter port.StubFilter,
	sortOpt port.SortOption,
	rng port.RangeOption,
) ([]domain.Stub, int) {
	// Get all stubs and convert
	allStubs := r.getAllStubs()
	stubs := r.convertStubs(allStubs)

	// Filter
	filteredStubs := make([]domain.Stub, 0, len(stubs))
	for _, stub := range stubs {
		if r.matchesFilter(stub, filter) {
			filteredStubs = append(filteredStubs, stub)
		}
	}

	// Apply sorting
	r.sortStubs(filteredStubs, sortOpt)

	// Apply pagination
	total := len(filteredStubs)
	if rng.Start >= total {
		return []domain.Stub{}, total
	}

	end := rng.End
	if end > total {
		end = total
	}

	return filteredStubs[rng.Start:end], total
}

// Search searches for stubs using the underlying searcher.
func (r *StubRepository) Search(ctx context.Context, query string) ([]domain.Stub, error) {
	if query == "" {
		// Return all stubs if query is empty
		allStubs := r.getAllStubs()

		return r.convertStubs(allStubs), nil
	}

	// Get all stubs and filter by search query
	allStubs := r.getAllStubs()
	convertedStubs := r.convertStubs(allStubs)
	matchedStubs := make([]domain.Stub, 0, len(convertedStubs))

	queryLower := strings.ToLower(query)

	for _, stub := range convertedStubs {
		// Search in various fields
		if r.matchesSearchQuery(stub, queryLower) {
			matchedStubs = append(matchedStubs, stub)
		}
	}

	return matchedStubs, nil
}

// GetStats returns repository statistics.
func (r *StubRepository) GetStats() map[string]any {
	r.muUniqueIDs.RLock()
	defer r.muUniqueIDs.RUnlock()

	return map[string]any{
		"totalStubs":    len(r.uniqueIDs),
		"totalFiles":    len(r.mapIDsByFile),
		"isLoaded":      r.loaded.Load(),
		"fileCacheSize": len(r.fileCache),
	}
}

// ClearCache clears the file cache.
func (r *StubRepository) ClearCache() {
	r.muCache.Lock()
	defer r.muCache.Unlock()

	r.fileCache = make(map[string]FileCacheEntry)
}

// getAllStubs retrieves all stubs from budgerigar with fallback to Used/Unused.
func (r *StubRepository) getAllStubs() []*stuber.Stub {
	allStubs := r.budgerigar.All()

	// Fallback to Used/Unused if no stubs found
	if len(allStubs) == 0 {
		usedStubs := r.budgerigar.Used()
		unusedStubs := r.budgerigar.Unused()

		allStubs = append(allStubs, usedStubs...)
		allStubs = append(allStubs, unusedStubs...)
	}

	return allStubs
}

// convertStubs converts a slice of stuber.Stub to domain.Stub.
func (r *StubRepository) convertStubs(stubs []*stuber.Stub) []domain.Stub {
	return ConvertStubs(stubs)
}

// convertFromStuberStub converts stuber.Stub to domain.Stub.
func (r *StubRepository) convertFromStuberStub(stub *stuber.Stub) domain.Stub {
	return ConvertFromStuberStub(stub)
}

// matchesSearchQuery checks if a stub matches the search query.
func (r *StubRepository) matchesSearchQuery(stub domain.Stub, queryLower string) bool {
	// Search in basic fields
	if r.searchInBasicFields(stub, queryLower) {
		return true
	}

	// Search in inputs
	if r.searchInInputs(stub.Inputs, queryLower) {
		return true
	}

	// Search in outputs
	if r.searchInOutputs(stub.OutputsRaw, queryLower) {
		return true
	}

	// Search in response metadata
	if r.searchInResponseMetadata(stub, queryLower) {
		return true
	}

	return false
}

// trackStub tracks a stub by its UUID and string ID.
//
//nolint:unparam
func (r *StubRepository) trackStub(uid uuid.UUID, stringID string) {
	r.muUniqueIDs.Lock()
	defer r.muUniqueIDs.Unlock()

	r.uniqueIDs[uid] = struct{}{}
}

// untrackStub removes tracking for a stub.
func (r *StubRepository) untrackStub(uid uuid.UUID) {
	r.muUniqueIDs.Lock()
	defer r.muUniqueIDs.Unlock()

	delete(r.uniqueIDs, uid)
}

// matchesFilter checks if a stub matches the given filter.
func (r *StubRepository) matchesFilter(stub domain.Stub, filter port.StubFilter) bool {
	if filter.Service != "" && stub.Service != filter.Service {
		return false
	}

	if filter.Method != "" && stub.Method != filter.Method {
		return false
	}

	return true
}

// sortStubs sorts stubs according to the sort option.
func (r *StubRepository) sortStubs(stubs []domain.Stub, sortOpt port.SortOption) {
	// Implementation of sorting logic
	// This would sort by priority, ID, service, method, etc.
}

// convertToStuberStub converts domain.StubV4 to stuber.Stub.
func (r *StubRepository) convertToStuberStub(stub domain.Stub) *stuber.Stub {
	// Implementation of conversion logic
	return &stuber.Stub{
		ID:      uuid.MustParse(stub.ID),
		Service: stub.Service,
		Method:  stub.Method,
		// Add other fields as needed
	}
}

// searchInBasicFields searches in basic stub fields.
func (r *StubRepository) searchInBasicFields(stub domain.Stub, queryLower string) bool {
	return ContainsIgnoreCase(stub.Service, queryLower) ||
		ContainsIgnoreCase(stub.Method, queryLower) ||
		ContainsIgnoreCase(stub.ID, queryLower)
}

// searchInInputs searches in stub inputs.
func (r *StubRepository) searchInInputs(inputs []domain.Matcher, queryLower string) bool {
	for _, input := range inputs {
		if r.searchInMap(input.Equals, queryLower) ||
			r.searchInMap(input.Contains, queryLower) ||
			r.searchInStringMap(input.Matches, queryLower) {
			return true
		}
	}

	return false
}

// searchInOutputs searches in stub outputs.
func (r *StubRepository) searchInOutputs(outputs []map[string]any, queryLower string) bool {
	for _, output := range outputs {
		if r.searchInMap(output, queryLower) {
			return true
		}
	}

	return false
}

// searchInResponseMetadata searches in response headers and trailers.
func (r *StubRepository) searchInResponseMetadata(stub domain.Stub, queryLower string) bool {
	return r.searchInStringMap(stub.ResponseHeaders, queryLower) ||
		r.searchInStringMap(stub.ResponseTrailers, queryLower)
}

// searchInStringMap searches for the query string in a map[string]string.
func (r *StubRepository) searchInStringMap(data map[string]string, queryLower string) bool {
	if data == nil {
		return false
	}

	for key, value := range data {
		if ContainsIgnoreCase(key, queryLower) ||
			ContainsIgnoreCase(value, queryLower) {
			return true
		}
	}

	return false
}

// searchInMap recursively searches for the query string in a map.
func (r *StubRepository) searchInMap(data map[string]any, queryLower string) bool {
	if data == nil {
		return false
	}

	for key, value := range data {
		// Search in key
		if ContainsIgnoreCase(key, queryLower) {
			return true
		}

		// Search in value
		switch v := value.(type) {
		case string:
			if ContainsIgnoreCase(v, queryLower) {
				return true
			}
		case map[string]any:
			if r.searchInMap(v, queryLower) {
				return true
			}
		case []any:
			if r.searchInSlice(v, queryLower) {
				return true
			}
		}
	}

	return false
}

// searchInSlice recursively searches for the query string in a slice.
func (r *StubRepository) searchInSlice(data []any, queryLower string) bool {
	if data == nil {
		return false
	}

	for _, value := range data {
		switch v := value.(type) {
		case string:
			if ContainsIgnoreCase(v, queryLower) {
				return true
			}
		case map[string]any:
			if r.searchInMap(v, queryLower) {
				return true
			}
		case []any:
			if r.searchInSlice(v, queryLower) {
				return true
			}
		}
	}

	return false
}
