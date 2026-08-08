package stuber_test

import (
	"strconv"
	"testing"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func regexStubs(count int) []*stuber.Stub {
	stubs := make([]*stuber.Stub, count)

	for i := range stubs {
		name := "user-" + strconv.Itoa(i)
		last := len(name) - 1
		stubs[i] = &stuber.Stub{
			Service: "Greeter",
			Method:  "SayHello",
			Input: stuber.InputData{Matches: map[string]any{
				"name": "^" + name[:last] + "[" + name[last:] + "]$",
			}},
			Output: stuber.Output{Data: map[string]any{"message": "hi"}},
		}
	}

	return stubs
}

func BenchmarkMatchesScanHit(b *testing.B) {
	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(regexStubs(10000)...)

	query := stuber.Query{
		Service: "Greeter",
		Method:  "SayHello",
		Input:   []map[string]any{{"name": "user-9999"}},
	}

	b.ReportAllocs()

	for b.Loop() {
		result, err := budgerigar.FindByQuery(query)
		if err != nil {
			b.Fatal(err)
		}

		if result.Found() == nil {
			b.Fatal("expected a match")
		}
	}
}

func BenchmarkMatchesScanMiss(b *testing.B) {
	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(regexStubs(10000)...)

	query := stuber.Query{
		Service: "Greeter",
		Method:  "SayHello",
		Input:   []map[string]any{{"name": "nobody"}},
	}

	b.ReportAllocs()

	for b.Loop() {
		result, err := budgerigar.FindByQuery(query)
		if err != nil {
			b.Fatal(err)
		}

		if result.Found() != nil {
			b.Fatal("expected no match")
		}
	}
}
