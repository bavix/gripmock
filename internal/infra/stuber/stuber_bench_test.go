package stuber_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// BenchmarkPutMany measures the performance of inserting multiple Stub values.
func BenchmarkPutMany(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	// Prepare a slice of Stub values to insert.
	values := make([]*stuber.Stub, 500)

	for i := range 500 {
		values[i] = &stuber.Stub{
			ID:      uuid.New(),
			Service: "service-" + uuid.NewString(),
			Method:  "method-" + uuid.NewString(),
		}
	}

	b.ReportAllocs()

	// Insert the values into the Budgerigar.
	for b.Loop() {
		for range 1000 {
			budgerigar.PutMany(values...)
		}
	}
}

// BenchmarkUpdateMany measures the performance of updating multiple Stub values.
func BenchmarkUpdateMany(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	// Insert initial values.
	values := make([]*stuber.Stub, 500)
	for i := range 500 {
		values[i] = &stuber.Stub{
			ID:      uuid.New(),
			Service: "service-" + uuid.NewString(),
			Method:  "method-" + uuid.NewString(),
		}
	}

	budgerigar.PutMany(values...)

	// Update the values.
	b.ReportAllocs()

	for b.Loop() {
		for range 1000 {
			budgerigar.UpdateMany(values...)
		}
	}
}

// BenchmarkDeleteByID measures the performance of deleting Stub values by ID.
func BenchmarkDeleteByID(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	// Insert initial values and collect their IDs.
	ids := make([]uuid.UUID, 500)

	for i := range 500 {
		id := uuid.New()
		ids[i] = id
		budgerigar.PutMany(&stuber.Stub{
			ID:      id,
			Service: "service-" + uuid.NewString(),
			Method:  "method-" + uuid.NewString(),
		})
	}

	b.ReportAllocs()
	b.ResetTimer()

	// Delete the values by their IDs.
	for _, id := range ids {
		for range 1000 {
			budgerigar.DeleteByID(id)
		}
	}
}

// BenchmarkFindByID measures the performance of finding a Stub value by ID.
func BenchmarkFindByID(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	for range 500 {
		budgerigar.PutMany(&stuber.Stub{
			ID:      uuid.New(),
			Service: "service-" + uuid.NewString(),
			Method:  "method-" + uuid.NewString(),
		})
	}

	b.ReportAllocs()

	// Find the target value by its ID.
	for b.Loop() {
		for range 1000 {
			_ = budgerigar.FindByID(uuid.Nil)
		}
	}
}

// BenchmarkFindByQuery measures the performance of finding a Stub value by Query.
func BenchmarkFindByQuery(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	// Insert initial values.
	for range 500 {
		budgerigar.PutMany(&stuber.Stub{
			ID:      uuid.New(),
			Service: "service-" + uuid.NewString(),
			Method:  "method-" + uuid.NewString(),
		})
	}

	query := stuber.Query{
		Service: "service-some-name",
		Method:  "method-some-name",
	}

	b.ReportAllocs()

	// Find values by the query.
	for b.Loop() {
		for range 1000 {
			_, _ = budgerigar.FindByQuery(query)
		}
	}
}

// BenchmarkFindBy measures the performance of finding Stub values by service and method.
func BenchmarkFindBy(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	// Insert initial values.
	for range 500 {
		budgerigar.PutMany(&stuber.Stub{
			ID:      uuid.New(),
			Service: "service-" + uuid.NewString(),
			Method:  "method-" + uuid.NewString(),
		})
	}

	service := "service-some-name"
	method := "method-some-name"

	b.ReportAllocs()

	// Find values by service and method.
	for b.Loop() {
		for range 1000 {
			_, _ = budgerigar.FindBy(service, method)
		}
	}
}

// BenchmarkAll measures the performance of retrieving all Stub values.
func BenchmarkAll(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	// Insert initial values.
	for range 500 {
		budgerigar.PutMany(&stuber.Stub{
			ID:      uuid.New(),
			Service: "service-" + uuid.NewString(),
			Method:  "method-" + uuid.NewString(),
		})
	}

	b.ReportAllocs()

	// Retrieve all values.
	for b.Loop() {
		for range 1000 {
			_ = budgerigar.All()
		}
	}
}

// BenchmarkUsed measures the performance of retrieving used Stub values.
func BenchmarkUsed(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	// Insert initial values.
	for range 500 {
		budgerigar.PutMany(&stuber.Stub{
			ID:      uuid.New(),
			Service: "service-" + uuid.NewString(),
			Method:  "method-" + uuid.NewString(),
		})
	}

	b.ReportAllocs()

	// Retrieve used values.
	for b.Loop() {
		for range 1000 {
			_ = budgerigar.Used()
		}
	}
}

// BenchmarkUnused measures the performance of retrieving unused Stub values.
func BenchmarkUnused(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	// Insert initial values.
	for range 500 {
		budgerigar.PutMany(&stuber.Stub{
			ID:      uuid.New(),
			Service: "service-" + uuid.NewString(),
			Method:  "method-" + uuid.NewString(),
		})
	}

	b.ReportAllocs()

	// Retrieve unused values.
	for b.Loop() {
		for range 1000 {
			_ = budgerigar.Unused()
		}
	}
}

// BenchmarkFindByQueryStream measures the performance of finding stubs with stream data.
func BenchmarkFindByQueryStream(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	stubs := make([]*stuber.Stub, 100)
	for i := range 100 {
		stubs[i] = &stuber.Stub{
			ID:      uuid.New(),
			Service: "service-" + uuid.NewString(),
			Method:  "method-" + uuid.NewString(),
			Inputs: []stuber.InputData{
				{Equals: map[string]any{"stream1": "value1"}},
				{Equals: map[string]any{"stream2": "value2"}},
			},
		}
	}

	budgerigar.PutMany(stubs...)

	query := stuber.Query{
		Service: "service-" + uuid.NewString(),
		Method:  "method-" + uuid.NewString(),
		Input:   []map[string]any{{"stream1": "value1"}},
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = budgerigar.FindByQuery(query)
	}
}

// BenchmarkFindByQueryStreamBackwardCompatibility measures the performance of backward compatibility.
func BenchmarkFindByQueryStreamBackwardCompatibility(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	stubs := make([]*stuber.Stub, 100)
	for i := range 100 {
		stubs[i] = &stuber.Stub{
			ID:      uuid.New(),
			Service: "service-" + uuid.NewString(),
			Method:  "method-" + uuid.NewString(),
			Input: stuber.InputData{
				Equals: map[string]any{"key1": "value1"},
			},
		}
	}

	budgerigar.PutMany(stubs...)

	query := stuber.Query{
		Service: "service-" + uuid.NewString(),
		Method:  "method-" + uuid.NewString(),
		Input:   []map[string]any{{"key1": "value1"}},
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = budgerigar.FindByQuery(query)
	}
}

// BenchmarkMatchStream measures the performance of stream matching through public API.
func BenchmarkMatchStream(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"key1": "value1"}},
			{Equals: map[string]any{"key2": "value2"}},
		},
	}

	budgerigar.PutMany(stub)

	query := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"key1": "value1"}},
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = budgerigar.FindByQuery(query)
	}
}

// BenchmarkRankMatchStream measures the performance of stream ranking through public API.
func BenchmarkRankMatchStream(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	stubs := make([]*stuber.Stub, 10)
	for i := range 10 {
		stubs[i] = &stuber.Stub{
			ID:      uuid.New(),
			Service: "test",
			Method:  "test",
			Inputs: []stuber.InputData{
				{Equals: map[string]any{"key1": "value1"}},
				{Equals: map[string]any{"key2": "value2"}},
			},
		}
	}

	budgerigar.PutMany(stubs...)

	query := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"key1": "value1"}},
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = budgerigar.FindByQuery(query)
	}
}

// BenchmarkQueryV2Unary measures the performance of V2 unary requests.
func BenchmarkQueryV2Unary(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Input: stuber.InputData{
			Equals: map[string]any{"key1": "value1"},
		},
	}

	budgerigar.PutMany(stub)

	query := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"key1": "value1"}},
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = budgerigar.FindByQuery(query)
	}
}

// BenchmarkQueryV2Stream measures the performance of V2 stream requests.
func BenchmarkQueryV2Stream(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"stream1": "value1"}},
			{Equals: map[string]any{"stream2": "value2"}},
		},
	}

	budgerigar.PutMany(stub)

	query := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"stream1": "value1"}, {"stream2": "value2"}},
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = budgerigar.FindByQuery(query)
	}
}

// BenchmarkQueryV2Comparison compares V1 vs V2 performance.
func BenchmarkQueryV2Comparison(b *testing.B) {
	budgerigar := stuber.NewBudgerigar(features.New())

	stub := &stuber.Stub{
		Service: "test",
		Method:  "test",
		Input: stuber.InputData{
			Equals: map[string]any{"key1": "value1"},
		},
	}

	budgerigar.PutMany(stub)

	queryUnary := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"key1": "value1"}},
	}

	queryStream := stuber.Query{
		Service: "test",
		Method:  "test",
		Input:   []map[string]any{{"stream1": "value1"}, {"stream2": "value2"}},
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = budgerigar.FindByQuery(queryUnary)
		_, _ = budgerigar.FindByQuery(queryStream)
	}
}

// BenchmarkBidiStreaming benchmarks bidirectional streaming performance.
//
//nolint:funlen
func BenchmarkBidiStreaming(b *testing.B) {
	s := stuber.NewBudgerigar(features.New())

	// Create multiple stubs with different patterns
	stub1 := &stuber.Stub{
		ID:       uuid.New(),
		Service:  "ChatService",
		Method:   "Chat",
		Priority: 1,
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"message": "hello"}},
			{Equals: map[string]any{"message": "world"}},
			{Equals: map[string]any{"message": "goodbye"}},
		},
		Output: stuber.Output{
			Data: map[string]any{"response": "Pattern 1 completed"},
		},
	}

	stub2 := &stuber.Stub{
		ID:       uuid.New(),
		Service:  "ChatService",
		Method:   "Chat",
		Priority: 2,
		Inputs: []stuber.InputData{
			{Equals: map[string]any{"message": "hello"}},
			{Equals: map[string]any{"message": "universe"}},
			{Equals: map[string]any{"message": "farewell"}},
		},
		Output: stuber.Output{
			Data: map[string]any{"response": "Pattern 2 completed"},
		},
	}

	s.PutMany(stub1, stub2)

	query := stuber.QueryBidi{
		Service: "ChatService",
		Method:  "Chat",
		Headers: map[string]any{"content-type": "application/json"},
	}

	b.ReportAllocs()

	for b.Loop() {
		result, err := s.FindByQueryBidi(query)
		if err != nil {
			b.Fatal(err)
		}

		// Simulate a conversation
		messages := []map[string]any{
			{"message": "hello"},
			{"message": "world"},
			{"message": "goodbye"},
		}

		for _, msg := range messages {
			_, err := result.Next(msg)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
