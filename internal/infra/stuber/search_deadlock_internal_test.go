package stuber

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestConcurrentSearchAndDeleteDoesNotDeadlock(t *testing.T) {
	t.Parallel()

	budgerigar := NewBudgerigar()
	ids := seedStubs(t, budgerigar)

	var wg sync.WaitGroup

	stop := make(chan struct{})

	for range 8 {
		wg.Go(func() { searchUntilStopped(budgerigar, stop) })
	}

	wg.Go(func() {
		for _, id := range ids {
			budgerigar.DeleteByID(id)
		}
	})

	done := make(chan struct{})

	go func() {
		defer close(done)

		wg.Wait()
	}()

	time.Sleep(300 * time.Millisecond)
	close(stop)

	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("deadlock: searches and deletes did not complete")
	}
}

func seedStubs(t *testing.T, budgerigar *Budgerigar) []uuid.UUID {
	t.Helper()

	const stubCount = 64

	ids := make([]uuid.UUID, 0, stubCount)

	for range stubCount {
		stub := &Stub{
			ID:      uuid.New(),
			Service: "svc",
			Method:  "M",
			Input:   InputData{Equals: map[string]any{"name": "known"}},
			Output:  Output{Data: map[string]any{"ok": true}},
		}
		budgerigar.PutMany(stub)
		ids = append(ids, stub.ID)
	}

	return ids
}

func searchUntilStopped(budgerigar *Budgerigar, stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		default:
		}

		_, _ = budgerigar.FindByQuery(Query{
			Service: "svc",
			Method:  "M",
			Input:   []map[string]any{{"name": "unmatched"}},
		})
	}
}
