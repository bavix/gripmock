package stuber_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// BenchmarkV1vsV2_Found compares V1 vs V2 when stub is found.
func BenchmarkV1vsV2_Found(b *testing.B) {
	// V1 Test - Found case
	b.Run("V1_Found", func(b *testing.B) {
		budgerigar := stuber.NewBudgerigar(features.New())

		// Add stub that will be found
		stub := &stuber.Stub{
			ID:      uuid.New(),
			Service: "test-service",
			Method:  "test-method",
			Input: stuber.InputData{
				Equals: map[string]any{"key1": "value1"},
			},
		}
		budgerigar.PutMany(stub)

		query := stuber.Query{
			Service: "test-service",
			Method:  "test-method",
			Data:    map[string]any{"key1": "value1"},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			_, _ = budgerigar.FindByQuery(context.Background(), query)
		}
	})

	// V2 Test - Found case
	b.Run("V2_Found", func(b *testing.B) {
		budgerigar := stuber.NewBudgerigar(features.New())

		// Add stub that will be found
		stub := &stuber.Stub{
			ID:      uuid.New(),
			Service: "test-service",
			Method:  "test-method",
			Input: stuber.InputData{
				Equals: map[string]any{"key1": "value1"},
			},
		}
		budgerigar.PutMany(stub)

		query := stuber.QueryV2{
			Service: "test-service",
			Method:  "test-method",
			Input:   []map[string]any{{"key1": "value1"}},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			_, _ = budgerigar.FindByQueryV2(context.Background(), query)
		}
	})
}

// BenchmarkV1vsV2_NotFound compares V1 vs V2 when stub is not found.
func BenchmarkV1vsV2_NotFound(b *testing.B) {
	// V1 Test - Not Found case
	b.Run("V1_NotFound", func(b *testing.B) {
		budgerigar := stuber.NewBudgerigar(features.New())

		// Add some stubs but search for different one
		for i := range 100 {
			stub := &stuber.Stub{
				ID:      uuid.New(),
				Service: "service-" + string(rune(i)),
				Method:  "method-" + string(rune(i)),
				Input: stuber.InputData{
					Equals: map[string]any{"key": "value"},
				},
			}
			budgerigar.PutMany(stub)
		}

		query := stuber.Query{
			Service: "non-existent-service",
			Method:  "non-existent-method",
			Data:    map[string]any{"key": "value"},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			_, _ = budgerigar.FindByQuery(context.Background(), query)
		}
	})

	// V2 Test - Not Found case
	b.Run("V2_NotFound", func(b *testing.B) {
		budgerigar := stuber.NewBudgerigar(features.New())

		// Add some stubs but search for different one
		for i := range 100 {
			stub := &stuber.Stub{
				ID:      uuid.New(),
				Service: "service-" + string(rune(i)),
				Method:  "method-" + string(rune(i)),
				Input: stuber.InputData{
					Equals: map[string]any{"key": "value"},
				},
			}
			budgerigar.PutMany(stub)
		}

		query := stuber.QueryV2{
			Service: "non-existent-service",
			Method:  "non-existent-method",
			Input:   []map[string]any{{"key": "value"}},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			_, _ = budgerigar.FindByQueryV2(context.Background(), query)
		}
	})
}

// BenchmarkV1vsV2_MultipleStubs compares V1 vs V2 with multiple matching stubs.
func BenchmarkV1vsV2_MultipleStubs(b *testing.B) {
	// V1 Test - Multiple stubs
	b.Run("V1_Multiple", func(b *testing.B) {
		budgerigar := stuber.NewBudgerigar(features.New())

		// Add multiple stubs with same service/method but different priorities
		for i := range 10 {
			stub := &stuber.Stub{
				ID:       uuid.New(),
				Service:  "test-service",
				Method:   "test-method",
				Priority: i,
				Input: stuber.InputData{
					Equals: map[string]any{"key1": "value1"},
				},
			}
			budgerigar.PutMany(stub)
		}

		query := stuber.Query{
			Service: "test-service",
			Method:  "test-method",
			Data:    map[string]any{"key1": "value1"},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			_, _ = budgerigar.FindByQuery(context.Background(), query)
		}
	})

	// V2 Test - Multiple stubs
	b.Run("V2_Multiple", func(b *testing.B) {
		budgerigar := stuber.NewBudgerigar(features.New())

		// Add multiple stubs with same service/method but different priorities
		for i := range 10 {
			stub := &stuber.Stub{
				ID:       uuid.New(),
				Service:  "test-service",
				Method:   "test-method",
				Priority: i,
				Input: stuber.InputData{
					Equals: map[string]any{"key1": "value1"},
				},
			}
			budgerigar.PutMany(stub)
		}

		query := stuber.QueryV2{
			Service: "test-service",
			Method:  "test-method",
			Input:   []map[string]any{{"key1": "value1"}},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			_, _ = budgerigar.FindByQueryV2(context.Background(), query)
		}
	})
}

// BenchmarkV1vsV2_Stream compares V1 vs V2 for streaming scenarios.
func BenchmarkV1vsV2_Stream(b *testing.B) {
	// V1 Test - Stream
	b.Run("V1_Stream", func(b *testing.B) {
		budgerigar := stuber.NewBudgerigar(features.New())

		stub := &stuber.Stub{
			ID:      uuid.New(),
			Service: "test-service",
			Method:  "test-method",
			Inputs: []stuber.InputData{
				{Equals: map[string]any{"stream1": "value1"}},
				{Equals: map[string]any{"stream2": "value2"}},
			},
		}
		budgerigar.PutMany(stub)

		query := stuber.Query{
			Service: "test-service",
			Method:  "test-method",
			Data:    map[string]any{"stream1": "value1"},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			_, _ = budgerigar.FindByQuery(context.Background(), query)
		}
	})

	// V2 Test - Stream
	b.Run("V2_Stream", func(b *testing.B) {
		budgerigar := stuber.NewBudgerigar(features.New())

		stub := &stuber.Stub{
			ID:      uuid.New(),
			Service: "test-service",
			Method:  "test-method",
			Inputs: []stuber.InputData{
				{Equals: map[string]any{"stream1": "value1"}},
				{Equals: map[string]any{"stream2": "value2"}},
			},
		}
		budgerigar.PutMany(stub)

		query := stuber.QueryV2{
			Service: "test-service",
			Method:  "test-method",
			Input:   []map[string]any{{"stream1": "value1"}, {"stream2": "value2"}},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			_, _ = budgerigar.FindByQueryV2(context.Background(), query)
		}
	})
}
