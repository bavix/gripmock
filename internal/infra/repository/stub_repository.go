package repository

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

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

	// Loading state
	loaded       atomic.Bool
	loadComplete chan struct{}
}

// NewStubRepository creates a new stub repository that uses Budgerigar for storage.
func NewStubRepository(budgerigar *stuber.Budgerigar) *StubRepository {
	return &StubRepository{
		budgerigar:   budgerigar,
		mapIDsByFile: make(map[string][]uuid.UUID),
		uniqueIDs:    make(map[uuid.UUID]struct{}),
		loadComplete: make(chan struct{}),
	}
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
	// Convert domain.Stub to stuber.Stub
	stubV4 := &stuber.Stub{
		ID:               uuid.New(), // Generate new UUID if not provided
		Service:          stub.Service,
		Method:           stub.Method,
		ResponseHeaders:  stub.ResponseHeaders,
		ResponseTrailers: stub.ResponseTrailers,
		Times:            stub.Times,
		// Convert v4 fields
		InputsV4:     stub.Inputs,
		OutputsRawV4: stub.OutputsRaw,
	}

	// If ID is provided in domain.Stub, use it
	if stub.ID != "" {
		if id, err := uuid.Parse(stub.ID); err == nil {
			stubV4.ID = id
		}
	}

	// Add to Budgerigar using PutMany
	ids := r.budgerigar.PutMany(stubV4)
	if len(ids) > 0 {
		stub.ID = ids[0].String()
	}

	return stub, nil
}

// Update updates an existing stub in the repository.
func (r *StubRepository) Update(ctx context.Context, id string, stub domain.Stub) (domain.Stub, error) {
	// Parse the ID
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return domain.Stub{}, err
	}

	// Convert domain.Stub to stuber.Stub
	stubV4 := &stuber.Stub{
		ID:               uuidID,
		Service:          stub.Service,
		Method:           stub.Method,
		ResponseHeaders:  stub.ResponseHeaders,
		ResponseTrailers: stub.ResponseTrailers,
		Times:            stub.Times,
		// Convert v4 fields
		InputsV4:     stub.Inputs,
		OutputsRawV4: stub.OutputsRaw,
	}

	// Update in Budgerigar using UpdateMany
	ids := r.budgerigar.UpdateMany(stubV4)
	if len(ids) > 0 {
		stub.ID = ids[0].String()
	}

	return stub, nil
}

// Delete removes a stub from the repository.
func (r *StubRepository) Delete(ctx context.Context, id string) error {
	// Parse the ID
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	// Delete from Budgerigar
	r.budgerigar.DeleteByID(uuidID)

	return nil
}

// DeleteMany removes multiple stubs from the repository.
func (r *StubRepository) DeleteMany(ctx context.Context, ids []string) error {
	// Convert string IDs to UUIDs
	uuidIDs := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if uuidID, err := uuid.Parse(id); err == nil {
			uuidIDs = append(uuidIDs, uuidID)
		}
	}

	// Delete from Budgerigar
	r.budgerigar.DeleteByID(uuidIDs...)

	return nil
}

// GetByID retrieves a stub by its ID.
func (r *StubRepository) GetByID(ctx context.Context, id string) (domain.Stub, bool) {
	// Parse the ID
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return domain.Stub{}, false
	}

	// Get from Budgerigar
	stub := r.budgerigar.FindByID(uuidID)
	if stub == nil {
		return domain.Stub{}, false
	}

	// Convert to domain.Stub - simplified conversion
	return domain.Stub{
		ID:               stub.ID.String(),
		Service:          stub.Service,
		Method:           stub.Method,
		Priority:         stub.Priority,
		Headers:          r.convertInputHeader(stub.Headers),
		Inputs:           r.convertInputs(stub),
		OutputsRaw:       r.convertOutputs(stub),
		ResponseHeaders:  stub.ResponseHeaders,
		ResponseTrailers: stub.ResponseTrailers,
		Times:            stub.Times,
	}, true
}

// List retrieves stubs with filtering, sorting, and pagination.
func (r *StubRepository) List(
	ctx context.Context,
	filter port.StubFilter,
	sort port.SortOption,
	rng port.RangeOption,
) ([]domain.Stub, int) {
	// Get all stubs from Budgerigar
	allStubs := r.budgerigar.All()

	// Apply filters
	filteredStubs := r.applyFilters(allStubs, filter)

	// Apply sorting
	r.applySorting(filteredStubs, sort)

	// Apply pagination
	total := len(filteredStubs)
	paginatedStubs := r.applyPagination(filteredStubs, rng)

	// Convert to domain.Stub
	result := make([]domain.Stub, 0, len(paginatedStubs))
	for _, stub := range paginatedStubs {
		result = append(result, domain.Stub{
			ID:               stub.ID.String(),
			Service:          stub.Service,
			Method:           stub.Method,
			Inputs:           stub.InputsV4,
			OutputsRaw:       stub.OutputsRawV4,
			ResponseHeaders:  stub.ResponseHeaders,
			ResponseTrailers: stub.ResponseTrailers,
			Times:            stub.Times,
		})
	}

	return result, total
}

// applyFilters applies filtering to the stub list.
func (r *StubRepository) applyFilters(stubs []*stuber.Stub, filter port.StubFilter) []*stuber.Stub {
	if r.isNoFilter(filter) {
		return stubs
	}

	filtered := make([]*stuber.Stub, 0)

	for _, stub := range stubs {
		if r.matchesFilter(stub, filter) {
			filtered = append(filtered, stub)
		}
	}

	return filtered
}

// isNoFilter checks if no filters are applied.
func (r *StubRepository) isNoFilter(filter port.StubFilter) bool {
	return filter.Service == "" && filter.Method == "" && filter.Used == nil && filter.Query == "" && len(filter.IDs) == 0
}

// matchesFilter checks if a stub matches the given filter.
func (r *StubRepository) matchesFilter(stub *stuber.Stub, filter port.StubFilter) bool {
	if !r.matchesServiceFilter(stub, filter.Service) {
		return false
	}

	if !r.matchesMethodFilter(stub, filter.Method) {
		return false
	}

	if !r.matchesUsageFilter(stub, filter.Used) {
		return false
	}

	if !r.matchesQueryFilter(stub, filter.Query) {
		return false
	}

	if !r.matchesIDFilter(stub, filter.IDs) {
		return false
	}

	return true
}

// matchesServiceFilter checks if a stub matches the service filter.
func (r *StubRepository) matchesServiceFilter(stub *stuber.Stub, service string) bool {
	if service == "" {
		return true
	}

	return strings.EqualFold(stub.Service, service)
}

// matchesMethodFilter checks if a stub matches the method filter.
func (r *StubRepository) matchesMethodFilter(stub *stuber.Stub, method string) bool {
	if method == "" {
		return true
	}

	return strings.EqualFold(stub.Method, method)
}

// matchesUsageFilter checks if a stub matches the usage filter.
func (r *StubRepository) matchesUsageFilter(stub *stuber.Stub, used *bool) bool {
	if used == nil {
		return true
	}

	isUsed := r.isStubUsed(stub)

	return isUsed == *used
}

// matchesQueryFilter checks if a stub matches the query filter.
func (r *StubRepository) matchesQueryFilter(stub *stuber.Stub, query string) bool {
	if query == "" {
		return true
	}

	queryLower := strings.ToLower(query)
	service := strings.ToLower(stub.Service)
	method := strings.ToLower(stub.Method)

	return strings.Contains(service, queryLower) || strings.Contains(method, queryLower)
}

// matchesIDFilter checks if a stub matches the ID filter.
func (r *StubRepository) matchesIDFilter(stub *stuber.Stub, ids []string) bool {
	for _, id := range ids {
		if id == stub.ID.String() {
			return true
		}
	}

	return false
}

// convertInputHeader converts stuber.InputHeader to domain.Matcher.
func (r *StubRepository) convertInputHeader(header stuber.InputHeader) *domain.Matcher {
	if header.Len() == 0 {
		return nil
	}

	return &domain.Matcher{
		Equals:   header.Equals,
		Contains: header.Contains,
		Matches:  header.Matches,
	}
}

// convertInputs converts stuber.Stub inputs to domain.Matcher slice.
func (r *StubRepository) convertInputs(stub *stuber.Stub) []domain.Matcher {
	// For V4 stubs, use InputsV4
	if len(stub.InputsV4) > 0 {
		return stub.InputsV4
	}

	// For legacy stubs, convert Input and Inputs
	var inputs []domain.Matcher

	// Convert single Input (unary)
	if r.hasInputData(stub.Input) {
		inputs = append(inputs, r.convertInputData(stub.Input))
	}

	// Convert Inputs (client streaming)
	for _, input := range stub.Inputs {
		if r.hasInputData(input) {
			inputs = append(inputs, r.convertInputData(input))
		}
	}

	return inputs
}

// hasInputData checks if InputData has any content.
func (r *StubRepository) hasInputData(input stuber.InputData) bool {
	return len(input.Equals) > 0 || len(input.Contains) > 0 || len(input.Matches) > 0 || len(input.Any) > 0
}

// convertInputData converts stuber.InputData to domain.Matcher.
func (r *StubRepository) convertInputData(input stuber.InputData) domain.Matcher {
	return domain.Matcher{
		Equals:           input.Equals,
		Contains:         input.Contains,
		Matches:          input.Matches,
		IgnoreArrayOrder: input.IgnoreArrayOrder,
		Any:              r.convertInputDataSlice(input.Any),
	}
}

// convertInputDataSlice converts []stuber.InputData to []domain.Matcher.
func (r *StubRepository) convertInputDataSlice(inputs []stuber.InputData) []domain.Matcher {
	if len(inputs) == 0 {
		return nil
	}

	result := make([]domain.Matcher, 0, len(inputs))
	for _, input := range inputs {
		if r.hasInputData(input) {
			result = append(result, r.convertInputData(input))
		}
	}

	return result
}

// convertOutputs converts stuber.Stub outputs to []map[string]any.
func (r *StubRepository) convertOutputs(stub *stuber.Stub) []map[string]any {
	// For V4 stubs, use OutputsRawV4
	if len(stub.OutputsRawV4) > 0 {
		return stub.OutputsRawV4
	}

	// For legacy stubs, convert Output
	if r.hasOutput(stub.Output) {
		output := make(map[string]any)

		if stub.Output.Data != nil {
			output["data"] = stub.Output.Data
		}

		if len(stub.Output.Stream) > 0 {
			output["stream"] = stub.Output.Stream
		}

		if stub.Output.Error != "" {
			output["error"] = stub.Output.Error
		}

		if stub.Output.Code != nil {
			output["code"] = *stub.Output.Code
		}

		if stub.Output.Delay != 0 {
			output["delay"] = time.Duration(stub.Output.Delay).String()
		}

		if len(stub.Output.Headers) > 0 {
			output["headers"] = stub.Output.Headers
		}

		return []map[string]any{output}
	}

	return nil
}

// hasOutput checks if Output has any content.
func (r *StubRepository) hasOutput(output stuber.Output) bool {
	return output.Data != nil || len(output.Stream) > 0 || output.Error != "" ||
		output.Code != nil || output.Delay != 0 || len(output.Headers) > 0
}

// applySorting applies sorting to the stub list.
func (r *StubRepository) applySorting(stubs []*stuber.Stub, sort port.SortOption) {
	// Placeholder for future sorting implementation
	_ = stubs
	_ = sort
}

// applyPagination applies pagination to the stub list.
func (r *StubRepository) applyPagination(stubs []*stuber.Stub, rng port.RangeOption) []*stuber.Stub {
	if rng.Start >= len(stubs) {
		return []*stuber.Stub{}
	}

	end := rng.End + 1
	if end > len(stubs) {
		end = len(stubs)
	}

	return stubs[rng.Start:end]
}

// isStubUsed checks if a stub has been used.
func (r *StubRepository) isStubUsed(stub *stuber.Stub) bool {
	// Check if stub is in the used list
	usedStubs := r.budgerigar.Used()
	for _, usedStub := range usedStubs {
		if usedStub.ID == stub.ID {
			return true
		}
	}

	return false
}
