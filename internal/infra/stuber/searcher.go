package stuber

import (
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	"github.com/google/uuid"
)

// PriorityMultiplier is used to boost stub priority in ranking calculations.
// Higher values give more weight to explicit priority settings.
const PriorityMultiplier = 10.0

// Specificity calculation constants.
const (
	// EmptySpecificity is returned when no fields match.
	EmptySpecificity = 0
	// MinStreamLength is the minimum length for stream calculations.
	MinStreamLength = 0
	// parallelProcessingThreshold is the threshold for using parallel processing.
	parallelProcessingThreshold = 100
)

// ErrServiceNotFound is returned when the service is not found.
var ErrServiceNotFound = errors.New("service not found")

// ErrMethodNotFound is returned when the method is not found.
var ErrMethodNotFound = errors.New("method not found")

// ErrStubNotFound is returned when the stub is not found.
var ErrStubNotFound = errors.New("stub not found")

// searcher is a struct that manages the storage of search results.
//
// It contains a mutex for concurrent access, a map to store and retrieve
// used stubs by their UUID, and a pointer to the storage struct.
type searcher struct {
	mu       sync.RWMutex
	stubUsed map[uuid.UUID]struct{}
	storage  *storage
}

// newSearcher creates a new searcher instance.
func newSearcher() *searcher {
	return &searcher{
		storage:  newStorage(),
		stubUsed: make(map[uuid.UUID]struct{}),
	}
}

// Result holds the search result: exact match (Found) or best similar (Similar).
type Result struct {
	found   *Stub // The exact match found in the search
	similar *Stub // The most similar match found
}

// Found returns the exact match found in the search.
func (r *Result) Found() *Stub {
	return r.found
}

// Similar returns the most similar match found in the search.
func (r *Result) Similar() *Stub {
	return r.similar
}

// BidiResult holds matching stubs for bidirectional streaming.
type BidiResult struct {
	searcher      *searcher
	query         QueryBidi
	matchingStubs []*Stub      // Stubs that match the current message pattern
	messageCount  atomic.Int32 // Number of messages processed so far
	mu            sync.RWMutex // Thread safety for concurrent access
}

// Next processes the next message in the bidirectional stream and returns the matching stub.
func (br *BidiResult) Next(messageData map[string]any) (*Stub, error) {
	br.mu.Lock()
	defer br.mu.Unlock()

	if messageData == nil {
		return nil, ErrStubNotFound
	}

	if !br.ensureMatchingStubs(messageData) {
		return nil, ErrStubNotFound
	}

	bestStub := br.selectBestStub(messageData)
	if bestStub == nil {
		return nil, ErrStubNotFound
	}

	br.finalizeStubSelection(bestStub, messageData)

	return bestStub, nil
}

// GetMessageIndex returns the current message index in the bidirectional stream.
func (br *BidiResult) GetMessageIndex() int {
	return int(br.messageCount.Load())
}

func (br *BidiResult) ensureMatchingStubs(messageData map[string]any) bool {
	if len(br.matchingStubs) == 0 {
		allStubs, err := br.searcher.findBy(br.query.Service, br.query.Method)
		if err != nil {
			return false
		}

		for _, stub := range allStubs {
			if br.stubMatchesMessage(stub, messageData) {
				br.matchingStubs = append(br.matchingStubs, stub)
			}
		}
	} else {
		br.messageCount.Add(1)
		br.matchingStubs = br.filterMatchingStubs(messageData)
	}

	return len(br.matchingStubs) > 0
}

func (br *BidiResult) filterMatchingStubs(messageData map[string]any) []*Stub {
	var filtered []*Stub

	for _, stub := range br.matchingStubs {
		if br.stubMatchesMessage(stub, messageData) {
			filtered = append(filtered, stub)
		}
	}

	return filtered
}

func (br *BidiResult) selectBestStub(messageData map[string]any) *Stub {
	messageIndex := int(br.messageCount.Load())

	var (
		bestStub               *Stub
		bestRank               float64
		candidatesWithSameRank []*Stub
	)

	for _, stub := range br.matchingStubs {
		rank := br.rankStubForMessage(stub, messageData, messageIndex)
		priorityBonus := float64(stub.Priority) * PriorityMultiplier
		totalRank := rank + priorityBonus

		if totalRank > bestRank {
			bestStub = stub
			bestRank = totalRank
			candidatesWithSameRank = []*Stub{stub}
		} else if totalRank == bestRank {
			candidatesWithSameRank = append(candidatesWithSameRank, stub)
		}
	}

	if len(candidatesWithSameRank) > 1 {
		sortStubsByID(candidatesWithSameRank)
		bestStub = candidatesWithSameRank[0]
	}

	return bestStub
}

func (br *BidiResult) rankStubForMessage(stub *Stub, messageData map[string]any, messageIndex int) float64 {
	if stub.IsBidirectional() && len(stub.Inputs) > 0 {
		if messageIndex < len(stub.Inputs) {
			return br.rankInputData(stub.Inputs[messageIndex], messageData)
		}

		return 0.1 //nolint:mnd
	}

	query := Query{
		Service: br.query.Service,
		Method:  br.query.Method,
		Headers: br.query.Headers,
		Input:   []map[string]any{messageData},
		toggles: br.query.toggles,
	}

	return br.rankStub(stub, query)
}

func (br *BidiResult) finalizeStubSelection(bestStub *Stub, messageData map[string]any) {
	query := Query{
		Service: br.query.Service,
		Method:  br.query.Method,
		Headers: br.query.Headers,
		Input:   []map[string]any{messageData},
		toggles: br.query.toggles,
	}

	if !bestStub.IsClientStream() && br.messageCount.Load() == 0 {
		br.matchingStubs = br.matchingStubs[:0]
		br.messageCount.Store(0)
	}

	br.searcher.mark(query, bestStub.ID)
}

// stubMatchesMessage checks if a stub matches the given message.
// For bidirectional streaming, we check if the message matches any of the stream elements.
func (br *BidiResult) stubMatchesMessage(stub *Stub, messageData map[string]any) bool {
	// For bidirectional streaming stubs, check if message matches any stream element
	if stub.IsBidirectional() {
		// New format: use Inputs for input matching
		if len(stub.Inputs) > 0 {
			for _, streamElement := range stub.Inputs {
				if br.matchInputData(streamElement, messageData) {
					return true
				}
			}

			return false
		}
		// Old format: use Input for matching (backward compatibility)
		return br.matchInputData(stub.Input, messageData)
	}

	// For client streaming stubs, check if message matches any stream element
	if stub.IsClientStream() {
		for _, streamElement := range stub.Inputs {
			if br.matchInputData(streamElement, messageData) {
				return true
			}
		}

		return false
	}

	// For unary stubs, use Input matching
	if stub.IsUnary() {
		return br.matchInputData(stub.Input, messageData)
	}

	// For server streaming stubs, use Input matching
	if stub.IsServerStream() {
		return br.matchInputData(stub.Input, messageData)
	}

	return false
}

// rankInputData ranks how well messageData matches the given InputData.
//
//nolint:cyclop
func (br *BidiResult) rankInputData(inputData InputData, messageData map[string]any) float64 {
	// Early exit if InputData is empty
	if len(inputData.Equals) == 0 && len(inputData.Contains) == 0 && len(inputData.Matches) == 0 {
		return 1.0 // Perfect match for empty matchers
	}

	var totalRank float64

	// Rank Equals - each match gives high weight
	if len(inputData.Equals) > 0 {
		equalsRank := 0.0

		for key, expectedValue := range inputData.Equals {
			if actualValue, exists := br.findValueWithVariations(messageData, key); exists && deepEqual(actualValue, expectedValue) {
				equalsRank += 100.0 // High weight for exact matches
			}
		}

		totalRank += equalsRank
	}

	// Rank Contains - each match gives medium weight
	if len(inputData.Contains) > 0 {
		containsRank := 0.0

		for key, expectedValue := range inputData.Contains {
			actualValue, exists := messageData[key]
			if exists {
				// Create minimal map for contains check
				tempMap := map[string]any{key: expectedValue}
				if contains(tempMap, actualValue, false) {
					containsRank += 10.0 // Medium weight for contains matches
				}
			}
		}

		totalRank += containsRank
	}

	// Rank Matches - each match gives medium weight
	if len(inputData.Matches) > 0 {
		matchesRank := 0.0

		for key, expectedValue := range inputData.Matches {
			actualValue, exists := messageData[key]
			if exists {
				// Create minimal map for matches check
				tempMap := map[string]any{key: expectedValue}
				if matches(tempMap, actualValue, false) {
					matchesRank += 10.0 // Medium weight for matches
				}
			}
		}

		totalRank += matchesRank
	}

	return totalRank
}

// matchInputData checks if messageData matches the given InputData.
//
//nolint:cyclop
func (br *BidiResult) matchInputData(inputData InputData, messageData map[string]any) bool {
	// Early exit if InputData is empty
	if len(inputData.Equals) == 0 && len(inputData.Contains) == 0 && len(inputData.Matches) == 0 {
		return true
	}

	// Check Equals
	if len(inputData.Equals) > 0 {
		for key, expectedValue := range inputData.Equals {
			if actualValue, exists := br.findValueWithVariations(messageData, key); !exists || !deepEqual(actualValue, expectedValue) {
				return false
			}
		}
	}

	// Check Contains - avoid creating temporary maps
	if len(inputData.Contains) > 0 {
		for key, expectedValue := range inputData.Contains {
			actualValue, exists := messageData[key]
			if !exists {
				return false
			}
			// Create minimal map for contains check
			tempMap := map[string]any{key: expectedValue}
			if !contains(tempMap, actualValue, false) {
				return false
			}
		}
	}

	// Check Matches - avoid creating temporary maps
	if len(inputData.Matches) > 0 {
		for key, expectedValue := range inputData.Matches {
			actualValue, exists := messageData[key]
			if !exists {
				return false
			}
			// Create minimal map for matches check
			tempMap := map[string]any{key: expectedValue}
			if !matches(tempMap, actualValue, false) {
				return false
			}
		}
	}

	return true
}

// findValueWithVariations tries to find a value using different field name conventions.
func (br *BidiResult) findValueWithVariations(messageData map[string]any, key string) (any, bool) {
	// Try exact match first
	if value, exists := messageData[key]; exists {
		return value, true
	}

	// Try camelCase variations
	camelKey := toCamelCase(key)
	if value, exists := messageData[camelKey]; exists {
		return value, true
	}

	// Try snake_case variations
	snakeKey := toSnakeCase(key)
	if value, exists := messageData[snakeKey]; exists {
		return value, true
	}

	return nil, false
}

// toCamelCase converts snake_case to camelCase.
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		return s
	}

	result := parts[0]

	var resultSb452 strings.Builder

	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			resultSb452.WriteString(strings.ToUpper(parts[i][:1]) + parts[i][1:])
		}
	}

	result += resultSb452.String()

	return result
}

// toSnakeCase converts camelCase to snake_case.
func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder

	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteByte('_')
		}

		result.WriteRune(unicode.ToLower(r))
	}

	return result.String()
}

// deepEqual performs deep equality check with better implementation.
func deepEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	switch a.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return a == b
	}

	if eq := deepEqualMap(a, b); eq != nil {
		return *eq
	}

	if eq := deepEqualSlice(a, b); eq != nil {
		return *eq
	}

	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func deepEqualMap(a, b any) *bool {
	aMap, aOk := a.(map[string]any)

	bMap, bOk := b.(map[string]any)
	if !aOk || !bOk {
		return nil
	}

	if len(aMap) != len(bMap) {
		f := false

		return &f
	}

	for k, v := range aMap {
		if bv, exists := bMap[k]; !exists || !deepEqual(v, bv) {
			f := false

			return &f
		}
	}

	t := true

	return &t
}

func deepEqualSlice(a, b any) *bool {
	aSlice, aOk := a.([]any)

	bSlice, bOk := b.([]any)
	if !aOk || !bOk {
		return nil
	}

	if len(aSlice) != len(bSlice) {
		f := false

		return &f
	}

	for i, v := range aSlice {
		if !deepEqual(v, bSlice[i]) {
			f := false

			return &f
		}
	}

	t := true

	return &t
}

// sortStubsByID sorts stubs by ID for stable ordering when ranks are equal.
// This ensures consistent results across multiple runs.
func sortStubsByID(stubs []*Stub) {
	slices.SortFunc(stubs, func(a, b *Stub) int {
		return strings.Compare(a.ID.String(), b.ID.String())
	})
}

// rankStub calculates the ranking score for a stub.
func (br *BidiResult) rankStub(stub *Stub, query Query) float64 {
	// Use the existing V2 ranking logic
	// Rank headers first
	headersRank := rankHeaders(query.Headers, stub.Headers)

	// Priority to Inputs (newer functionality) over Input (legacy)
	if len(stub.Inputs) > 0 {
		// Streaming case
		return headersRank + rankStreamElements(query.Input, stub.Inputs)
	}

	// Handle Input (legacy) - check if query has input data
	if len(query.Input) == 0 {
		// Empty query - return header rank only
		return headersRank
	}

	if len(query.Input) == 1 {
		// Unary case
		return headersRank + rankInput(query.Input[0], stub.Input)
	}

	return headersRank
}

// upsert inserts the given stub values into the searcher. If a stub value
// already exists with the same key, it is updated.
//
// The function returns a slice of UUIDs representing the keys of the
// inserted or updated values.
func (s *searcher) upsert(values ...*Stub) []uuid.UUID {
	return s.storage.upsert(values...)
}

// del deletes the stub values with the given UUIDs from the searcher.
//
// Returns the number of stub values that were successfully deleted.
func (s *searcher) del(ids ...uuid.UUID) int {
	return s.storage.del(ids...)
}

// findByID retrieves the stub value associated with the given ID from the
// searcher.
//
// Returns a pointer to the Stub struct associated with the given ID, or nil
// if not found.
func (s *searcher) findByID(id uuid.UUID) *Stub {
	return s.storage.findByID(id)
}

// findBy retrieves all Stub values that match the given service and method
// from the searcher, sorted by score in descending order.
func (s *searcher) findBy(service, method string) ([]*Stub, error) {
	seq, err := s.storage.findAll(service, method)
	if err != nil {
		return nil, s.wrap(err)
	}

	return collectStubs(seq), nil
}

// clear resets the searcher.
//
// It clears the stubUsed map and calls the storage clear method.
func (s *searcher) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear the stubUsed map.
	s.stubUsed = make(map[uuid.UUID]struct{})

	// Clear the storage.
	s.storage.clear()
}

// all returns all Stub values stored in the searcher.
//
// Returns:
// - []*Stub: The Stub values stored in the searcher.
func (s *searcher) all() []*Stub {
	return collectStubs(s.storage.values())
}

// used returns all Stub values that have been used by the searcher.
//
// Returns:
// - []*Stub: The Stub values that have been used by the searcher.
func (s *searcher) used() []*Stub {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return collectStubs(s.storage.findByIDs(maps.Keys(s.stubUsed)))
}

// unused returns all Stub values that have not been used by the searcher.
//
// Returns:
// - []*Stub: The Stub values that have not been used by the searcher.
func (s *searcher) unused() []*Stub {
	s.mu.RLock()
	defer s.mu.RUnlock()

	unused := make([]*Stub, 0)

	for stub := range s.iterAll() {
		if _, exists := s.stubUsed[stub.ID]; !exists {
			unused = append(unused, stub)
		}
	}

	return unused
}

// find retrieves the Stub value associated with the given Query from the searcher.
//
// Parameters:
// - query: The Query used to search for a Stub value.
//
// Returns:
// - *Result: The Result containing the found Stub value (if any), or nil.
// - error: An error if the search fails.
func (s *searcher) find(query Query) (*Result, error) {
	if query.ID != nil {
		return s.searchByID(query)
	}

	return s.search(query)
}

// searchByID retrieves the Stub value associated with the given ID from the searcher.
func (s *searcher) searchByID(query Query) (*Result, error) {
	_, err := s.storage.posByPN(query.Service, query.Method)
	if err != nil {
		return nil, s.wrap(err)
	}

	if found := s.findByID(*query.ID); found != nil {
		s.mark(query, *query.ID)

		return &Result{found: found}, nil
	}

	return nil, ErrServiceNotFound
}

// search retrieves the Stub value using the optimized matching and ranking.
func (s *searcher) search(query Query) (*Result, error) {
	return s.searchOptimized(query)
}

// mark marks the given Stub value as used in the searcher.
//
// If the query's RequestInternal flag is set, the mark is skipped.
//
// Parameters:
// - query: The query used to mark the Stub value.
// - id: The UUID of the Stub value to mark.
func (s *searcher) mark(query Query, id uuid.UUID) {
	// If the query's RequestInternal flag is set, skip the mark.
	if query.RequestInternal() {
		return
	}

	// Lock the mutex to ensure concurrent access.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Mark the Stub value as used by adding it to the stubUsed map.
	s.stubUsed[id] = struct{}{}
}

// findBidi retrieves a BidiResult for bidirectional streaming with the given QueryBidi.
// For bidirectional streaming, each message is treated as a separate unary request.
func (s *searcher) findBidi(query QueryBidi) (*BidiResult, error) {
	// Check if the QueryBidi has an ID field
	if query.ID != nil {
		// For ID-based queries, we can't use bidirectional streaming - fallback to regular search
		return s.searchByIDBidi(query)
	}

	// Check if the given service and method are valid
	_, err := s.storage.posByPN(query.Service, query.Method)
	if err != nil {
		return nil, s.wrap(err)
	}

	// Fetch all stubs for this service/method
	seq, err := s.storage.findAll(query.Service, query.Method)
	if err != nil {
		return nil, s.wrap(err)
	}

	var allStubs []*Stub

	for stub := range seq {
		allStubs = append(allStubs, stub)
	}

	return &BidiResult{
		searcher:      s,
		query:         query,
		matchingStubs: make([]*Stub, 0),
	}, nil
}

// searchByIDBidi handles ID-based queries for bidirectional streaming.
// Since we can't use bidirectional streaming for ID-based queries, we fallback to regular search.
func (s *searcher) searchByIDBidi(query QueryBidi) (*BidiResult, error) {
	// Check if the given service and method are valid
	_, err := s.storage.posByPN(query.Service, query.Method)
	if err != nil {
		return nil, s.wrap(err)
	}

	// Search for the Stub value with the given ID
	if found := s.findByID(*query.ID); found != nil {
		return &BidiResult{
			searcher:      s,
			query:         query,
			matchingStubs: []*Stub{found},
		}, nil
	}

	// Return an error if the Stub value is not found
	return nil, ErrServiceNotFound
}

// searchOptimized performs ultra-fast search with minimal allocations.
func (s *searcher) searchOptimized(query Query) (*Result, error) {
	// Get stubs from storage (single call - optimized)
	seq, err := s.storage.findAll(query.Service, query.Method)
	if err != nil {
		return nil, s.wrap(err)
	}

	// Collect all stubs in a single pass
	var stubs []*Stub

	for stub := range seq {
		stubs = append(stubs, stub)
	}

	// Process collected stubs
	return s.processStubs(query, stubs)
}

// processStubs processes the collected stubs with ultra-fast paths.
func (s *searcher) processStubs(query Query, stubs []*Stub) (*Result, error) {
	if len(stubs) == 0 {
		return nil, ErrStubNotFound
	}

	if len(stubs) == 1 {
		stub := stubs[0]
		if s.fastMatchV2(query, stub) {
			s.mark(query, stub.ID)

			return &Result{found: stub}, nil
		}

		return &Result{similar: stub}, nil
	}

	// Parallel processing for multiple stubs
	if len(stubs) >= parallelProcessingThreshold {
		return s.processStubsParallel(query, stubs)
	}

	// Single-threaded processing for small sets
	return s.processStubsSequential(query, stubs)
}

// processStubsSequential processes stubs sequentially (original logic).
func (s *searcher) processStubsSequential(query Query, stubs []*Stub) (*Result, error) {
	var (
		bestMatch       *Stub
		bestScore       float64
		bestSpecificity int
		mostSimilar     *Stub
		highestRank     float64
	)

	for _, stub := range stubs {
		rank := s.fastRankV2(query, stub)
		priorityBonus := float64(stub.Priority) * PriorityMultiplier
		specificity := s.calcSpecificity(stub, query)
		totalScore := rank + priorityBonus

		if s.fastMatchV2(query, stub) {
			if specificity > bestSpecificity || (specificity == bestSpecificity && totalScore > bestScore) {
				bestMatch, bestScore, bestSpecificity = stub, totalScore, specificity
			}
		}

		if totalScore > highestRank { // Track most similar even if not exact match
			mostSimilar, highestRank = stub, totalScore
		}
	}

	if bestMatch != nil {
		s.mark(query, bestMatch.ID)

		return &Result{found: bestMatch}, nil
	}

	if mostSimilar != nil {
		return &Result{similar: mostSimilar}, nil
	}

	return nil, ErrStubNotFound
}

// processStubsParallel processes stubs in parallel using goroutines.
func (s *searcher) processStubsParallel(query Query, stubs []*Stub) (*Result, error) {
	const chunkSize = 50

	numChunks := (len(stubs) + chunkSize - 1) / chunkSize

	bestMatchChan := make(chan *Stub, numChunks)
	mostSimilarChan := make(chan *Stub, numChunks)
	errorChan := make(chan error, numChunks)

	for i := range numChunks {
		start := i * chunkSize

		end := min(start+chunkSize, len(stubs))
		go func(chunkStubs []*Stub) {
			best, similar := s.processChunk(query, chunkStubs)
			bestMatchChan <- best

			mostSimilarChan <- similar

			errorChan <- nil
		}(stubs[start:end])
	}

	bestMatches, mostSimilars, err := s.collectChunkResults(numChunks, bestMatchChan, mostSimilarChan, errorChan)
	if err != nil {
		return nil, err
	}

	bestMatch := s.pickBestMatch(query, bestMatches)
	if bestMatch != nil {
		s.mark(query, bestMatch.ID)

		return &Result{found: bestMatch}, nil
	}

	mostSimilar := s.pickMostSimilar(query, mostSimilars)
	if mostSimilar != nil {
		return &Result{similar: mostSimilar}, nil
	}

	return nil, ErrStubNotFound
}

func (s *searcher) processChunk(query Query, chunkStubs []*Stub) (*Stub, *Stub) {
	var (
		bestMatch       *Stub
		mostSimilar     *Stub
		bestScore       float64
		bestSpecificity int
		highestRank     float64
	)

	for _, stub := range chunkStubs {
		rank := s.fastRankV2(query, stub)
		priorityBonus := float64(stub.Priority) * PriorityMultiplier
		specificity := s.calcSpecificity(stub, query)
		totalScore := rank + priorityBonus

		if s.fastMatchV2(query, stub) {
			if specificity > bestSpecificity || (specificity == bestSpecificity && totalScore > bestScore) {
				bestMatch, bestScore, bestSpecificity = stub, totalScore, specificity
			}
		}

		if totalScore > highestRank {
			mostSimilar, highestRank = stub, totalScore
		}
	}

	return bestMatch, mostSimilar
}

func (s *searcher) collectChunkResults(
	numChunks int,
	bestMatchChan, mostSimilarChan chan *Stub,
	errorChan chan error,
) ([]*Stub, []*Stub, error) {
	var bestMatches, mostSimilars []*Stub

	for range numChunks {
		if err := <-errorChan; err != nil {
			return nil, nil, err
		}

		if best := <-bestMatchChan; best != nil {
			bestMatches = append(bestMatches, best)
		}

		if similar := <-mostSimilarChan; similar != nil {
			mostSimilars = append(mostSimilars, similar)
		}
	}

	return bestMatches, mostSimilars, nil
}

func (s *searcher) pickBestMatch(query Query, candidates []*Stub) *Stub {
	var (
		best            *Stub
		bestScore       float64
		bestSpecificity int
	)

	for _, stub := range candidates {
		rank := s.fastRankV2(query, stub)
		priorityBonus := float64(stub.Priority) * PriorityMultiplier
		specificity := s.calcSpecificity(stub, query)
		totalScore := rank + priorityBonus

		if specificity > bestSpecificity || (specificity == bestSpecificity && totalScore > bestScore) {
			best, bestScore, bestSpecificity = stub, totalScore, specificity
		}
	}

	return best
}

func (s *searcher) pickMostSimilar(query Query, candidates []*Stub) *Stub {
	var (
		best        *Stub
		highestRank float64
	)

	for _, stub := range candidates {
		rank := s.fastRankV2(query, stub)
		priorityBonus := float64(stub.Priority) * PriorityMultiplier
		totalScore := rank + priorityBonus

		if totalScore > highestRank {
			best, highestRank = stub, totalScore
		}
	}

	return best
}

// fastMatchV2 is an ultra-optimized version of matchV2.
//
//nolint:cyclop
func (s *searcher) fastMatchV2(query Query, stub *Stub) bool {
	// If stub has headers, query must also have headers
	if stub.Headers.Len() > 0 && len(query.Headers) == 0 {
		return false
	}

	if len(query.Headers) > 0 && !matchHeaders(query.Headers, stub.Headers) {
		return false
	}

	// Priority to Inputs (stream) over Input (unary)
	// stub.Inputs != nil means stream stub (even if empty slice)
	if stub.Inputs != nil {
		if len(stub.Inputs) == 0 {
			return false // stream stub with no patterns matches nothing
		}

		return s.fastMatchStream(query.Input, stub.Inputs)
	}

	// Handle Input (unary) - stub uses Input
	// Stub with no input conditions matches any query (including empty)
	if len(query.Input) == 0 {
		// Empty query - check if stub can handle empty input
		return len(stub.Input.Equals) == 0 && len(stub.Input.Contains) == 0 && len(stub.Input.Matches) == 0
	}

	if len(query.Input) == 1 {
		return s.fastMatchInput(query.Input[0], stub.Input)
	}

	return false
}

// fastRankV2 is an ultra-optimized version of rankMatchV2.
func (s *searcher) fastRankV2(query Query, stub *Stub) float64 {
	if len(query.Headers) > 0 && !matchHeaders(query.Headers, stub.Headers) {
		return 0
	}

	// Include header rank so that stubs with matching headers get higher score within same priority
	headersRank := rankHeaders(query.Headers, stub.Headers)

	// Priority to Inputs (stream) over Input (unary)
	if stub.Inputs != nil {
		if len(stub.Inputs) == 0 {
			return headersRank
		}

		inputsBonus := 1000.0

		return headersRank + s.fastRankStream(query.Input, stub.Inputs) + inputsBonus
	}

	// Handle Input (unary)
	if len(query.Input) == 0 {
		// Empty query - return header rank only
		return headersRank
	}

	if len(query.Input) == 1 {
		return headersRank + s.fastRankInput(query.Input[0], stub.Input)
	}

	return headersRank
}

// fastMatchInput is an ultra-optimized version of matchInput.
//
//nolint:cyclop
func (s *searcher) fastMatchInput(queryData map[string]any, stubInput InputData) bool {
	// Fast path: empty query
	if len(queryData) == 0 {
		return len(stubInput.Equals) == 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0
	}

	// Ultra-fast path: equals only (most common case)
	if len(stubInput.Equals) > 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0 {
		return equals(stubInput.Equals, queryData, stubInput.IgnoreArrayOrder)
	}

	// Fast path: contains only
	if len(stubInput.Contains) > 0 && len(stubInput.Equals) == 0 && len(stubInput.Matches) == 0 {
		return contains(stubInput.Contains, queryData, stubInput.IgnoreArrayOrder)
	}

	// Fast path: matches only
	if len(stubInput.Matches) > 0 && len(stubInput.Equals) == 0 && len(stubInput.Contains) == 0 {
		return matches(stubInput.Matches, queryData, stubInput.IgnoreArrayOrder)
	}

	// Full matching (rare case)
	return matchInput(queryData, stubInput)
}

// fastMatchStream is an ultra-optimized version of matchStreamElements.
//
//nolint:cyclop
func (s *searcher) fastMatchStream(queryStream []map[string]any, stubStream []InputData) bool {
	// Check if stub has any input matching conditions
	hasConditions := false

	for _, stubElement := range stubStream {
		if stubElement.Equals != nil || stubElement.Contains != nil || stubElement.Matches != nil {
			hasConditions = true

			break
		}
	}

	if !hasConditions {
		return false // Stub has no input matching conditions
	}

	// Fast path: empty query stream
	if len(queryStream) == 0 {
		// Check if all stub stream elements can handle empty input
		for _, stubElement := range stubStream {
			if len(stubElement.Equals) > 0 || len(stubElement.Contains) > 0 || len(stubElement.Matches) > 0 {
				return false
			}
		}

		return true
	}

	// Fast path: single element
	if len(queryStream) == 1 && len(stubStream) == 1 {
		return s.fastMatchInput(queryStream[0], stubStream[0])
	}

	// Use original implementation for complex cases
	return matchStreamElements(queryStream, stubStream)
}

// fastRankInput is an ultra-optimized version of rankInput.
func (s *searcher) fastRankInput(queryData map[string]any, stubInput InputData) float64 {
	// Fast path: empty query
	if len(queryData) == 0 {
		// Check if stub can handle empty input
		if len(stubInput.Equals) == 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0 {
			return 1.0 // Perfect match for empty input
		}

		return 0
	}

	// Fast path: equals only
	if len(stubInput.Equals) > 0 && len(stubInput.Contains) == 0 && len(stubInput.Matches) == 0 {
		if equals(stubInput.Equals, queryData, stubInput.IgnoreArrayOrder) {
			return 1.0
		}

		return 0
	}

	// Use original implementation for complex cases
	return rankInput(queryData, stubInput)
}

// fastRankStream is an ultra-optimized version of rankStreamElements.
func (s *searcher) fastRankStream(queryStream []map[string]any, stubStream []InputData) float64 {
	// Fast path: empty query stream
	if len(queryStream) == 0 {
		// Check if all stub stream elements can handle empty input
		for _, stubElement := range stubStream {
			if len(stubElement.Equals) > 0 || len(stubElement.Contains) > 0 || len(stubElement.Matches) > 0 {
				return 0
			}
		}

		return 1.0 // Perfect match for empty input
	}

	// Fast path: single element
	if len(queryStream) == 1 && len(stubStream) == 1 {
		return s.fastRankInput(queryStream[0], stubStream[0])
	}

	// Use original implementation for complex cases
	return rankStreamElements(queryStream, stubStream)
}

func collectStubs(seq iter.Seq[*Stub]) []*Stub {
	result := make([]*Stub, 0)

	for stub := range seq {
		result = append(result, stub)
	}

	return result
}

func (s *searcher) iterAll() iter.Seq[*Stub] {
	return s.storage.values()
}

// wrap wraps an error with specific error types.
//
// Parameters:
// - err: The error to wrap.
//
// Returns:
// - The wrapped error.
func (s *searcher) wrap(err error) error {
	if errors.Is(err, ErrLeftNotFound) {
		return ErrServiceNotFound
	}

	if errors.Is(err, ErrRightNotFound) {
		return ErrMethodNotFound
	}

	return err
}

// calcSpecificity calculates the specificity score for a stub against a query.
// Higher specificity means more fields match between stub and query.
// Headers are given higher weight to ensure stubs with headers are preferred.
func (s *searcher) calcSpecificity(stub *Stub, query Query) int {
	// Specificity now reflects only input structure, header impact is accounted in rank via rankHeaders
	specificity := 0

	if len(query.Input) == 0 {
		return specificity
	}

	// Priority to Inputs (newer functionality) over Input (legacy)
	if len(stub.Inputs) > 0 {
		return specificity + s.calcSpecificityStream(stub.Inputs, query.Input)
	}

	if len(query.Input) == 1 {
		return specificity + s.calcSpecificityUnary(stub.Input, query.Input[0])
	}

	return specificity
}

// calcSpecificityUnary calculates specificity for unary case.
// Counts the number of fields that exist in both stub and query.
// Supports all field types: Equals, Contains, and Matches.
//
// Parameters:
// - stubInput: The stub's input data
// - queryData: The query's input data
//
// Returns:
// - int: The number of matching fields.
func (s *searcher) calcSpecificityUnary(stubInput InputData, queryData map[string]any) int {
	specificity := 0

	// Count equals fields
	for key := range stubInput.Equals {
		if _, exists := queryData[key]; exists {
			specificity++
		}
	}

	// Count contains fields
	for key := range stubInput.Contains {
		if _, exists := queryData[key]; exists {
			specificity++
		}
	}

	// Count matches fields
	for key := range stubInput.Matches {
		if _, exists := queryData[key]; exists {
			specificity++
		}
	}

	return specificity
}

// calcSpecificityStream calculates specificity for stream case.
func (s *searcher) calcSpecificityStream(stubStream []InputData, queryStream []map[string]any) int {
	if len(stubStream) == 0 || len(queryStream) == 0 {
		return 0
	}

	totalSpecificity := 0

	minLen := min(len(queryStream), len(stubStream))

	for i := range minLen {
		totalSpecificity += s.calcSpecificityUnary(stubStream[i], queryStream[i])
	}

	return totalSpecificity
}
