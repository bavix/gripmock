package stuber

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/runtime"
	"github.com/bavix/gripmock/v3/internal/infra/store/memory"
)

// mockWriter implements runtime.Writer for testing.
type mockWriter struct {
	headers  map[string]string
	trailers map[string]string
	messages []map[string]any
	status   *domain.GrpcStatus
}

func (w *mockWriter) SetHeaders(headers map[string]string) error {
	w.headers = headers

	return nil
}

func (w *mockWriter) Send(message map[string]any) error {
	w.messages = append(w.messages, message)

	return nil
}

func (w *mockWriter) SetTrailers(trailers map[string]string) error {
	w.trailers = trailers

	return nil
}

func (w *mockWriter) End(status *domain.GrpcStatus) error {
	w.status = status

	return nil
}

func TestTimesFunctionalityWithExecutor(t *testing.T) {
	t.Parallel()

	// Create analytics store
	analytics := memory.NewInMemoryAnalytics()

	// Create executor
	executor := &runtime.Executor{
		Analytics: analytics,
	}

	// Create stub with times limit
	stub := domain.Stub{
		ID:         uuid.New().String(),
		Service:    "test.Service",
		Method:     "TestMethod",
		Priority:   1,
		Times:      3, // Allow exactly 3 uses
		Inputs:     []domain.Matcher{{Equals: map[string]any{"field": "value"}}},
		OutputsRaw: []map[string]any{{"data": map[string]any{"result": "success"}}},
	}

	// Create mock writer
	writer := &mockWriter{}

	// Execute stub 3 times (should work)
	for i := range 3 {
		used, err := executor.Execute(context.Background(), stub, "unary", nil, []map[string]any{{"field": "value"}}, writer)
		require.NoError(t, err)
		assert.True(t, used, "Stub should be used for execution %d", i+1)
	}

	// Try to execute stub 4th time (should fail due to times limit)
	used, err := executor.Execute(context.Background(), stub, "unary", nil, []map[string]any{{"field": "value"}}, writer)
	require.NoError(t, err)
	assert.False(t, used, "Stub should not be used after 3 executions")

	// Verify analytics data
	analyticsData, exists := analytics.GetByStubID(context.Background(), stub.ID)
	assert.True(t, exists, "Analytics data should exist")
	assert.Equal(t, int64(3), analyticsData.UsedCount, "UsedCount should be exactly 3")
}

func TestTimesUnlimitedWithExecutor(t *testing.T) {
	t.Parallel()

	// Create analytics store
	analytics := memory.NewInMemoryAnalytics()

	// Create executor
	executor := &runtime.Executor{
		Analytics: analytics,
	}

	// Create stub with unlimited times
	stub := domain.Stub{
		ID:         uuid.New().String(),
		Service:    "test.Service",
		Method:     "TestMethod",
		Priority:   1,
		Times:      0, // Unlimited
		Inputs:     []domain.Matcher{{Equals: map[string]any{"field": "value"}}},
		OutputsRaw: []map[string]any{{"data": map[string]any{"result": "success"}}},
	}

	// Create mock writer
	writer := &mockWriter{}

	// Execute stub 10 times (should work since unlimited)
	for i := range 10 {
		used, err := executor.Execute(context.Background(), stub, "unary", nil, []map[string]any{{"field": "value"}}, writer)
		require.NoError(t, err)
		assert.True(t, used, "Stub should be used for execution %d", i+1)
	}

	// Verify analytics data
	analyticsData, exists := analytics.GetByStubID(context.Background(), stub.ID)
	assert.True(t, exists, "Analytics data should exist")
	assert.Equal(t, int64(10), analyticsData.UsedCount, "UsedCount should be 10")
}

func TestTimesNegativeValueWithExecutor(t *testing.T) {
	t.Parallel()

	// Create analytics store
	analytics := memory.NewInMemoryAnalytics()

	// Create executor
	executor := &runtime.Executor{
		Analytics: analytics,
	}

	// Create stub with negative times
	stub := domain.Stub{
		ID:         uuid.New().String(),
		Service:    "test.Service",
		Method:     "TestMethod",
		Priority:   1,
		Times:      -1, // Negative value (should be treated as unlimited)
		Inputs:     []domain.Matcher{{Equals: map[string]any{"field": "value"}}},
		OutputsRaw: []map[string]any{{"data": map[string]any{"result": "success"}}},
	}

	// Create mock writer
	writer := &mockWriter{}

	// Execute stub 5 times (should work since negative values are treated as unlimited)
	for i := range 5 {
		used, err := executor.Execute(context.Background(), stub, "unary", nil, []map[string]any{{"field": "value"}}, writer)
		require.NoError(t, err)
		assert.True(t, used, "Stub should be used for execution %d", i+1)
	}

	// Verify analytics data
	analyticsData, exists := analytics.GetByStubID(context.Background(), stub.ID)
	assert.True(t, exists, "Analytics data should exist")
	assert.Equal(t, int64(5), analyticsData.UsedCount, "UsedCount should be 5")
}

func TestTimesRealtimeUpdateWithExecutor(t *testing.T) {
	t.Parallel()

	// Create analytics store
	analytics := memory.NewInMemoryAnalytics()

	// Create executor
	executor := &runtime.Executor{
		Analytics: analytics,
	}

	// Create initial stub with times limit
	stub := domain.Stub{
		ID:         uuid.New().String(),
		Service:    "test.Service",
		Method:     "TestMethod",
		Priority:   1,
		Times:      2, // Initial limit: 2 uses
		Inputs:     []domain.Matcher{{Equals: map[string]any{"field": "value"}}},
		OutputsRaw: []map[string]any{{"data": map[string]any{"result": "success"}}},
	}

	// Create mock writer
	writer := &mockWriter{}

	// Execute stub once
	used, err := executor.Execute(context.Background(), stub, "unary", nil, []map[string]any{{"field": "value"}}, writer)
	require.NoError(t, err)
	assert.True(t, used, "Stub should be used for first execution")

	// Verify analytics shows 1 use
	analyticsData1, exists := analytics.GetByStubID(context.Background(), stub.ID)
	assert.True(t, exists)
	assert.Equal(t, int64(1), analyticsData1.UsedCount)

	// Update stub with new times limit (increase to 5)
	stub.Times = 5

	// Execute stub 4 more times (should work since limit is now 5)
	for i := range 4 {
		used, err := executor.Execute(context.Background(), stub, "unary", nil, []map[string]any{{"field": "value"}}, writer)
		require.NoError(t, err)
		assert.True(t, used, "Stub should be used for execution %d", i+2)
	}

	// Try to execute stub 6th time (should fail since limit is 5)
	used, err = executor.Execute(context.Background(), stub, "unary", nil, []map[string]any{{"field": "value"}}, writer)
	require.NoError(t, err)
	assert.False(t, used, "Stub should not be used after 5 executions")

	// Verify analytics shows 5 total uses
	analyticsData2, exists := analytics.GetByStubID(context.Background(), stub.ID)
	assert.True(t, exists)
	assert.Equal(t, int64(5), analyticsData2.UsedCount)
}

func TestTimesRealtimeDecreaseWithExecutor(t *testing.T) {
	t.Parallel()

	// Create analytics store
	analytics := memory.NewInMemoryAnalytics()

	// Create executor
	executor := &runtime.Executor{
		Analytics: analytics,
	}

	// Create initial stub with times limit
	stub := domain.Stub{
		ID:         uuid.New().String(),
		Service:    "test.Service",
		Method:     "TestMethod",
		Priority:   1,
		Times:      5, // Initial limit: 5 uses
		Inputs:     []domain.Matcher{{Equals: map[string]any{"field": "value"}}},
		OutputsRaw: []map[string]any{{"data": map[string]any{"result": "success"}}},
	}

	// Create mock writer
	writer := &mockWriter{}

	// Execute stub 3 times
	for i := range 3 {
		used, err := executor.Execute(context.Background(), stub, "unary", nil, []map[string]any{{"field": "value"}}, writer)
		require.NoError(t, err)
		assert.True(t, used, "Stub should be used for execution %d", i+1)
	}

	// Verify analytics shows 3 uses
	analyticsData1, exists := analytics.GetByStubID(context.Background(), stub.ID)
	assert.True(t, exists)
	assert.Equal(t, int64(3), analyticsData1.UsedCount)

	// Update stub with decreased times limit (decrease to 2)
	stub.Times = 2

	// Try to execute stub again (should fail since already used 3 times but limit is now 2)
	used, err := executor.Execute(context.Background(), stub, "unary", nil, []map[string]any{{"field": "value"}}, writer)
	require.NoError(t, err)
	assert.False(t, used, "Stub should not be used since used 3 times but limit is 2")

	// Verify analytics still shows 3 uses (counter doesn't reset)
	analyticsData2, exists := analytics.GetByStubID(context.Background(), stub.ID)
	assert.True(t, exists)
	assert.Equal(t, int64(3), analyticsData2.UsedCount)
}
