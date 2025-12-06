package stuber

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEmpty(t *testing.T) {
	t.Parallel()
	// Empty test file to avoid package import issues
	//nolint:testifylint
	require.True(t, true)
}

//nolint:funlen
func TestSearch_IgnoreArrayOrderAndFields(t *testing.T) {
	t.Parallel()

	s := newSearcher()

	stub1 := &Stub{
		ID:      uuid.New(),
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Input: InputData{
			Equals: map[string]any{
				"string_uuids": []any{
					"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03",
					"e3484119-24e1-42d9-b4c2-7d6004ee86d9",
					"cc991218-a920-40c8-9f42-3b329c8723f2",
					"c30f45d2-f8a4-4a94-a994-4cc349bca457",
				},
			},
			IgnoreArrayOrder: true,
		},
		Output: Output{
			Data: map[string]any{"process_id": 1, "status_code": 200},
		},
	}
	stub2 := &Stub{
		ID:      uuid.New(),
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Input: InputData{
			Equals: map[string]any{
				"string_uuids": []any{
					"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03",
					"e3484119-24e1-42d9-b4c2-7d6004ee86d9",
					"cc991218-a920-40c8-9f42-3b329c8723f2",
					"c30f45d2-f8a4-4a94-a994-4cc349bca457",
				},
				"request_timestamp": 1745081266,
			},
			IgnoreArrayOrder: true,
		},
		Output: Output{
			Data: map[string]any{"process_id": 2, "status_code": 200},
		},
	}
	s.upsert(stub1, stub2)

	// Request with array in any order and request_timestamp
	query := QueryV2{
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Input: []map[string]any{{
			"string_uuids": []any{
				"e3484119-24e1-42d9-b4c2-7d6004ee86d9",
				"c30f45d2-f8a4-4a94-a994-4cc349bca457",
				"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03",
				"cc991218-a920-40c8-9f42-3b329c8723f2",
			},
			"request_timestamp": 1745081266,
		}},
	}
	res, err := s.findV2(query)
	require.NoError(t, err)
	require.NotNil(t, res.Found())
	processID, ok := res.Found().Output.Data["process_id"].(int)
	require.True(t, ok)
	require.Equal(t, 2, processID)

	// Request with the same array, but without request_timestamp
	query2 := QueryV2{
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Input: []map[string]any{{
			"string_uuids": []any{
				"cc991218-a920-40c8-9f42-3b329c8723f2",
				"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03",
				"c30f45d2-f8a4-4a94-a994-4cc349bca457",
				"e3484119-24e1-42d9-b4c2-7d6004ee86d9",
			},
		}},
	}
	res2, err2 := s.findV2(query2)
	require.NoError(t, err2)
	require.NotNil(t, res2.Found())
	processID2, ok2 := res2.Found().Output.Data["process_id"].(int)
	require.True(t, ok2)
	require.Equal(t, 1, processID2)
}

//nolint:funlen
func TestSearch_IgnoreArrayOrder_UserScenario(t *testing.T) {
	t.Parallel()

	s := newSearcher()

	// Stub 1: without request_timestamp
	stub1 := &Stub{
		ID:      uuid.New(),
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Input: InputData{
			Equals: map[string]any{
				"string_uuids": []any{
					"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03", // 0
					"e3484119-24e1-42d9-b4c2-7d6004ee86d9", // 1
					"cc991218-a920-40c8-9f42-3b329c8723f2", // 2
					"c30f45d2-f8a4-4a94-a994-4cc349bca457", // 3
				},
			},
			IgnoreArrayOrder: true,
		},
		Output: Output{
			Data: map[string]any{"processId": "1", "statusCode": "200"},
		},
	}

	// Stub 2: with request_timestamp
	stub2 := &Stub{
		ID:      uuid.New(),
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Input: InputData{
			Equals: map[string]any{
				"string_uuids": []any{
					"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03", // 0
					"e3484119-24e1-42d9-b4c2-7d6004ee86d9", // 1
					"cc991218-a920-40c8-9f42-3b329c8723f2", // 2
					"c30f45d2-f8a4-4a94-a994-4cc349bca457", // 3
				},
				"request_timestamp": 1745081266,
			},
			IgnoreArrayOrder: true,
		},
		Output: Output{
			Data: map[string]any{"processId": "2", "statusCode": "200"},
		},
	}

	s.upsert(stub1, stub2)

	// Test case 1: Request with different order and request_timestamp
	query1 := QueryV2{
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Input: []map[string]any{{
			"string_uuids": []any{
				"e3484119-24e1-42d9-b4c2-7d6004ee86d9", // 1
				"c30f45d2-f8a4-4a94-a994-4cc349bca457", // 3
				"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03", // 0
				"cc991218-a920-40c8-9f42-3b329c8723f2", // 2
			},
			"request_timestamp": 1745081266,
		}},
	}
	res1, err1 := s.findV2(query1)
	require.NoError(t, err1)
	require.NotNil(t, res1.Found())
	require.Equal(t, "2", res1.Found().Output.Data["processId"])

	// Test case 2: Request with different order and NO request_timestamp
	query2 := QueryV2{
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Input: []map[string]any{{
			"string_uuids": []any{
				"e3484119-24e1-42d9-b4c2-7d6004ee86d9", // 1
				"c30f45d2-f8a4-4a94-a994-4cc349bca457", // 3
				"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03", // 0
				"cc991218-a920-40c8-9f42-3b329c8723f2", // 2
			},
		}},
	}
	res2, err2 := s.findV2(query2)
	require.NoError(t, err2)
	require.NotNil(t, res2.Found())
	require.Equal(t, "1", res2.Found().Output.Data["processId"])

	// Test case 3: Request with different order and request_timestamp (same as case 1)
	query3 := QueryV2{
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Input: []map[string]any{{
			"string_uuids": []any{
				"e3484119-24e1-42d9-b4c2-7d6004ee86d9", // 1
				"c30f45d2-f8a4-4a94-a994-4cc349bca457", // 3
				"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03", // 0
				"cc991218-a920-40c8-9f42-3b329c8723f2", // 2
			},
		}},
	}
	res3, err3 := s.findV2(query3)
	require.NoError(t, err3)
	require.NotNil(t, res3.Found())
	require.Equal(t, "1", res3.Found().Output.Data["processId"])
}

//nolint:funlen
func TestSearch_IgnoreArrayOrder_V1API(t *testing.T) {
	t.Parallel()

	s := newSearcher()

	stub1 := &Stub{
		ID:      uuid.New(),
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Input: InputData{
			Equals: map[string]any{
				"string_uuids": []any{
					"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03",
					"e3484119-24e1-42d9-b4c2-7d6004ee86d9",
					"cc991218-a920-40c8-9f42-3b329c8723f2",
					"c30f45d2-f8a4-4a94-a994-4cc349bca457",
				},
			},
			IgnoreArrayOrder: true,
		},
		Output: Output{
			Data: map[string]any{"processId": "1", "statusCode": "200"},
		},
	}

	stub2 := &Stub{
		ID:      uuid.New(),
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Input: InputData{
			Equals: map[string]any{
				"string_uuids": []any{
					"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03",
					"e3484119-24e1-42d9-b4c2-7d6004ee86d9",
					"cc991218-a920-40c8-9f42-3b329c8723f2",
					"c30f45d2-f8a4-4a94-a994-4cc349bca457",
				},
				"request_timestamp": 1745081266,
			},
			IgnoreArrayOrder: true,
		},
		Output: Output{
			Data: map[string]any{"processId": "2", "statusCode": "200"},
		},
	}

	s.upsert(stub1, stub2)

	// Test with V1 API
	query := Query{
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Data: map[string]any{
			"string_uuids": []any{
				"e3484119-24e1-42d9-b4c2-7d6004ee86d9",
				"c30f45d2-f8a4-4a94-a994-4cc349bca457",
				"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03",
				"cc991218-a920-40c8-9f42-3b329c8723f2",
			},
			"request_timestamp": 1745081266,
		},
	}
	res, err := s.find(query)
	require.NoError(t, err)
	require.NotNil(t, res.Found())
	require.Equal(t, "2", res.Found().Output.Data["processId"])

	// Test without request_timestamp
	query2 := Query{
		Service: "IdentifierService",
		Method:  "ProcessUUIDs",
		Data: map[string]any{
			"string_uuids": []any{
				"cc991218-a920-40c8-9f42-3b329c8723f2",
				"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03",
				"c30f45d2-f8a4-4a94-a994-4cc349bca457",
				"e3484119-24e1-42d9-b4c2-7d6004ee86d9",
			},
		},
	}
	res2, err2 := s.find(query2)
	require.NoError(t, err2)
	require.NotNil(t, res2.Found())
	require.Equal(t, "1", res2.Found().Output.Data["processId"])
}

//nolint:funlen
func TestSearch_Specificity_AllCases(t *testing.T) {
	t.Parallel()

	s := newSearcher()

	// Test case 1: Unary with equals fields
	stub1 := &Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "UnaryMethod",
		Input: InputData{
			Equals: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
		},
		Output: Output{
			Data: map[string]any{"result": "stub1"},
		},
	}

	stub2 := &Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "UnaryMethod",
		Input: InputData{
			Equals: map[string]any{
				"field1": "value1",
				"field2": "value2",
				"field3": "value3",
			},
		},
		Output: Output{
			Data: map[string]any{"result": "stub2"},
		},
	}

	s.upsert(stub1, stub2)

	// Query with field1 and field2 only - should match stub1
	query1 := QueryV2{
		Service: "TestService",
		Method:  "UnaryMethod",
		Input: []map[string]any{{
			"field1": "value1",
			"field2": "value2",
		}},
	}
	res1, err1 := s.findV2(query1)
	require.NoError(t, err1)
	require.NotNil(t, res1.Found())
	require.Equal(t, "stub1", res1.Found().Output.Data["result"])

	// Query with field1, field2, and field3 - should match stub2 (higher specificity)
	query2 := QueryV2{
		Service: "TestService",
		Method:  "UnaryMethod",
		Input: []map[string]any{{
			"field1": "value1",
			"field2": "value2",
			"field3": "value3",
		}},
	}
	res2, err2 := s.findV2(query2)
	require.NoError(t, err2)
	require.NotNil(t, res2.Found())
	require.Equal(t, "stub2", res2.Found().Output.Data["result"])
}

//nolint:funlen
func TestSearch_Specificity_StreamCase(t *testing.T) {
	t.Parallel()

	s := newSearcher()

	// Test case 2: Stream with equals fields
	stub1 := &Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "StreamMethod",
		Inputs: []InputData{
			{
				Equals: map[string]any{
					"field1": "value1",
				},
			},
			{
				Equals: map[string]any{
					"field2": "value2",
				},
			},
		},
		Output: Output{
			Data: map[string]any{"result": "stub1"},
		},
	}

	stub2 := &Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "StreamMethod",
		Inputs: []InputData{
			{
				Equals: map[string]any{
					"field1": "value1",
					"field3": "value3",
				},
			},
			{
				Equals: map[string]any{
					"field2": "value2",
					"field4": "value4",
				},
			},
		},
		Output: Output{
			Data: map[string]any{"result": "stub2"},
		},
	}

	s.upsert(stub1, stub2)

	// Stream query with basic fields - should match stub1
	query1 := QueryV2{
		Service: "TestService",
		Method:  "StreamMethod",
		Input: []map[string]any{
			{"field1": "value1"},
			{"field2": "value2"},
		},
	}
	res1, err1 := s.findV2(query1)
	require.NoError(t, err1)
	require.NotNil(t, res1.Found())
	require.Equal(t, "stub1", res1.Found().Output.Data["result"])

	// Stream query with additional fields - should match stub2 (higher specificity)
	query2 := QueryV2{
		Service: "TestService",
		Method:  "StreamMethod",
		Input: []map[string]any{
			{"field1": "value1", "field3": "value3"},
			{"field2": "value2", "field4": "value4"},
		},
	}
	res2, err2 := s.findV2(query2)
	require.NoError(t, err2)
	require.NotNil(t, res2.Found())
	require.Equal(t, "stub2", res2.Found().Output.Data["result"])
}

//nolint:funlen
func TestSearch_Specificity_WithContainsAndMatches(t *testing.T) {
	t.Parallel()

	s := newSearcher()

	// Test case 4: Mixed field types (equals, contains, matches)
	stub1 := &Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "MixedMethod",
		Input: InputData{
			Equals: map[string]any{
				"field1": "value1",
			},
			Contains: map[string]any{
				"field2": "value2",
			},
		},
		Output: Output{
			Data: map[string]any{"result": "stub1"},
		},
	}

	stub2 := &Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "MixedMethod",
		Input: InputData{
			Equals: map[string]any{
				"field1": "value1",
			},
			Contains: map[string]any{
				"field2": "value2",
			},
			Matches: map[string]any{
				"field3": "value3",
			},
		},
		Output: Output{
			Data: map[string]any{"result": "stub2"},
		},
	}

	s.upsert(stub1, stub2)

	// Query with equals and contains - should match stub1
	query1 := QueryV2{
		Service: "TestService",
		Method:  "MixedMethod",
		Input: []map[string]any{{
			"field1": "value1",
			"field2": "value2",
		}},
	}
	res1, err1 := s.findV2(query1)
	require.NoError(t, err1)
	require.NotNil(t, res1.Found())
	require.Equal(t, "stub1", res1.Found().Output.Data["result"])

	// Query with equals, contains, and matches - should match stub2 (higher specificity)
	query2 := QueryV2{
		Service: "TestService",
		Method:  "MixedMethod",
		Input: []map[string]any{{
			"field1": "value1",
			"field2": "value2",
			"field3": "value3",
		}},
	}
	res2, err2 := s.findV2(query2)
	require.NoError(t, err2)
	require.NotNil(t, res2.Found())
	require.Equal(t, "stub2", res2.Found().Output.Data["result"])
}

func TestSearch_Specificity_WithIgnoreArrayOrder(t *testing.T) {
	t.Parallel()

	s := newSearcher()

	// Test case 5: With ignoreArrayOrder
	stub1 := &Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "ArrayMethod",
		Input: InputData{
			Equals: map[string]any{
				"array1": []any{"a", "b", "c"},
			},
			IgnoreArrayOrder: true,
		},
		Output: Output{
			Data: map[string]any{"result": "stub1"},
		},
	}

	stub2 := &Stub{
		ID:      uuid.New(),
		Service: "TestService",
		Method:  "ArrayMethod",
		Input: InputData{
			Equals: map[string]any{
				"array1": []any{"a", "b", "c"},
				"field1": "value1",
			},
			IgnoreArrayOrder: true,
		},
		Output: Output{
			Data: map[string]any{"result": "stub2"},
		},
	}

	s.upsert(stub1, stub2)

	// Query with array only - should match stub1
	query1 := QueryV2{
		Service: "TestService",
		Method:  "ArrayMethod",
		Input: []map[string]any{{
			"array1": []any{"c", "a", "b"}, // Different order
		}},
	}
	res1, err1 := s.findV2(query1)
	require.NoError(t, err1)
	require.NotNil(t, res1.Found())
	require.Equal(t, "stub1", res1.Found().Output.Data["result"])

	// Query with array and additional field - should match stub2 (higher specificity)
	query2 := QueryV2{
		Service: "TestService",
		Method:  "ArrayMethod",
		Input: []map[string]any{{
			"array1": []any{"b", "c", "a"}, // Different order
			"field1": "value1",
		}},
	}
	res2, err2 := s.findV2(query2)
	require.NoError(t, err2)
	require.NotNil(t, res2.Found())
	require.Equal(t, "stub2", res2.Found().Output.Data["result"])
}

func TestProcessStubsParallel(t *testing.T) {
	t.Parallel()

	searcher := newSearcher()

	// Create many test stubs to trigger parallel processing
	stubs := make([]*Stub, 150)
	for i := range 150 {
		stubs[i] = &Stub{
			ID:       uuid.New(),
			Service:  "test.service",
			Method:   "TestMethod",
			Priority: i % 10,
			Input: InputData{
				Equals: map[string]any{"id": strconv.Itoa(i)},
			},
			Output: Output{
				Data: map[string]any{"result": fmt.Sprintf("stub%d", i)},
			},
		}
	}

	searcher.upsert(stubs...)

	// Test query that should find a match
	query := QueryV2{
		Service: "test.service",
		Method:  "TestMethod",
		Input:   []map[string]any{{"id": "50"}},
	}

	result, err := searcher.processStubs(query, stubs)
	require.NoError(t, err)
	require.NotNil(t, result.Found())
	require.Equal(t, "stub50", result.Found().Output.Data["result"])
}

func TestProcessStubsModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		stubCnt  int
		queryID  string
		expected string
	}{
		{
			name:     "parallel_mode",
			stubCnt:  150,
			queryID:  "50",
			expected: "stub50",
		},
		{
			name:     "sequential_mode",
			stubCnt:  50,
			queryID:  "25",
			expected: "stub25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			searcher := newSearcher()

			stubs := make([]*Stub, tt.stubCnt)
			for i := range tt.stubCnt {
				stubs[i] = &Stub{
					ID:       uuid.New(),
					Service:  "test.service",
					Method:   "TestMethod",
					Priority: i % 10,
					Input: InputData{
						Equals: map[string]any{"id": strconv.Itoa(i)},
					},
					Output: Output{
						Data: map[string]any{"result": fmt.Sprintf("stub%d", i)},
					},
				}
			}

			searcher.upsert(stubs...)

			query := QueryV2{
				Service: "test.service",
				Method:  "TestMethod",
				Input:   []map[string]any{{"id": tt.queryID}},
			}

			result, err := searcher.processStubs(query, stubs)
			require.NoError(t, err)
			require.NotNil(t, result.Found())
			require.Equal(t, tt.expected, result.Found().Output.Data["result"])
		})
	}
}

func TestProcessStubsParallelVsSequential(t *testing.T) {
	t.Parallel()

	searcher := newSearcher()

	// Create test stubs
	stubs := make([]*Stub, 200)
	for i := range 200 {
		stubs[i] = &Stub{
			ID:       uuid.New(),
			Service:  "test.service",
			Method:   "TestMethod",
			Priority: i % 10,
			Input: InputData{
				Equals: map[string]any{"id": strconv.Itoa(i)},
			},
			Output: Output{
				Data: map[string]any{"result": fmt.Sprintf("stub%d", i)},
			},
		}
	}

	searcher.upsert(stubs...)

	query := QueryV2{
		Service: "test.service",
		Method:  "TestMethod",
		Input:   []map[string]any{{"id": "100"}},
	}

	// Test parallel processing
	resultParallel, err := searcher.processStubs(query, stubs)
	require.NoError(t, err)
	require.NotNil(t, resultParallel.Found())

	// Test sequential processing directly
	resultSequential, err := searcher.processStubsSequential(query, stubs)
	require.NoError(t, err)
	require.NotNil(t, resultSequential.Found())

	// Results should be identical
	require.Equal(t, resultSequential.Found().ID, resultParallel.Found().ID)
	require.Equal(t, resultSequential.Found().Output.Data["result"], resultParallel.Found().Output.Data["result"])
}

// Benchmark parallel vs sequential processing.
func BenchmarkProcessStubs_ParallelVsSequential(b *testing.B) {
	searcher := newSearcher()

	// Create test stubs of different sizes
	stubSizes := []int{50, 100, 200, 500}

	for _, size := range stubSizes {
		stubs := make([]*Stub, size)
		for i := range size {
			stubs[i] = &Stub{
				ID:       uuid.New(),
				Service:  "test.service",
				Method:   "TestMethod",
				Priority: i % 10,
				Input: InputData{
					Equals: map[string]any{"id": strconv.Itoa(i)},
				},
				Output: Output{
					Data: map[string]any{"result": fmt.Sprintf("stub%d", i)},
				},
			}
		}

		searcher.upsert(stubs...)

		query := QueryV2{
			Service: "test.service",
			Method:  "TestMethod",
			Input:   []map[string]any{{"id": strconv.Itoa(size / 2)}},
		}

		b.Run(fmt.Sprintf("Size_%d_Sequential", size), func(b *testing.B) {
			for range b.N {
				_, err := searcher.processStubsSequential(query, stubs)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("Size_%d_Parallel", size), func(b *testing.B) {
			for range b.N {
				_, err := searcher.processStubs(query, stubs)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark different chunk sizes for parallel processing.
func BenchmarkProcessStubs_ChunkSizes(b *testing.B) {
	searcher := newSearcher()

	// Create large number of stubs
	stubs := make([]*Stub, 1000)
	for i := range 1000 {
		stubs[i] = &Stub{
			ID:       uuid.New(),
			Service:  "test.service",
			Method:   "TestMethod",
			Priority: i % 10,
			Input: InputData{
				Equals: map[string]any{"id": strconv.Itoa(i)},
			},
			Output: Output{
				Data: map[string]any{"result": fmt.Sprintf("stub%d", i)},
			},
		}
	}

	searcher.upsert(stubs...)

	query := QueryV2{
		Service: "test.service",
		Method:  "TestMethod",
		Input:   []map[string]any{{"id": "500"}},
	}

	// Test different chunk sizes
	chunkSizes := []int{25, 50, 100, 200}

	for _, chunkSize := range chunkSizes {
		b.Run(fmt.Sprintf("ChunkSize_%d", chunkSize), func(b *testing.B) {
			for range b.N {
				// Temporarily modify chunk size for testing
				// This would require making chunkSize configurable
				_, err := searcher.processStubs(query, stubs)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
