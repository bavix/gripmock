package runtime

import (
	"context"
	"testing"

	"github.com/bavix/gripmock/v3/internal/app/port"
	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

type fakeWriter struct {
	sent []map[string]any
}

func (f *fakeWriter) SetHeaders(_ map[string]string) error { return nil }
func (f *fakeWriter) Send(m map[string]any) error {
	f.sent = append(f.sent, m)

	return nil
}
func (f *fakeWriter) SetTrailers(_ map[string]string) error { return nil }
func (f *fakeWriter) End(_ *domain.GrpcStatus) error        { return nil }

type noopStubs struct{}

func (noopStubs) Create(context.Context, domain.Stub) (domain.Stub, error) {
	return domain.Stub{}, nil
}

func (noopStubs) Update(context.Context, string, domain.Stub) (domain.Stub, error) {
	return domain.Stub{}, nil
}
func (noopStubs) Delete(context.Context, string) error       { return nil }
func (noopStubs) DeleteMany(context.Context, []string) error { return nil }
func (noopStubs) GetByID(context.Context, string) (domain.Stub, bool) {
	return domain.Stub{}, false
}

func (noopStubs) List(context.Context, port.StubFilter, port.SortOption, port.RangeOption) ([]domain.Stub, int) {
	return nil, 0
}

type noopAnalytics struct{}

func (noopAnalytics) TouchStub(context.Context, string, int64, bool, int64, int64, int64) {}
func (noopAnalytics) GetByStubID(context.Context, string) (domain.StubAnalytics, bool) {
	return domain.StubAnalytics{}, false
}
func (noopAnalytics) ListAll(context.Context) []domain.StubAnalytics { return nil }

type noopHistory struct{}

func (noopHistory) Add(context.Context, domain.HistoryRecord) domain.HistoryRecord {
	return domain.HistoryRecord{}
}
func (noopHistory) List(context.Context, int, int) ([]domain.HistoryRecord, int) { return nil, 0 }
func (noopHistory) GetByID(context.Context, string) (domain.HistoryRecord, bool) {
	return domain.HistoryRecord{}, false
}
func (noopHistory) Clear(context.Context) {}

func TestExecute_DataRule(t *testing.T) {
	t.Parallel()

	stub := domain.Stub{
		ID:      "s1",
		Service: "svc",
		Method:  "Unary",
		OutputsRaw: []map[string]any{
			{"data": map[string]any{"x": 1}},
		},
	}

	exec := &Executor{Stubs: noopStubs{}, Analytics: noopAnalytics{}, History: noopHistory{}}
	w := &fakeWriter{}

	used, err := exec.Execute(context.Background(), stub, "unary", nil, nil, w)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	if !used {
		t.Fatalf("expected used=true")
	}

	v, ok := w.sent[0]["x"].(int)
	if !ok || v != 1 {
		t.Fatalf("unexpected sent: %#v", w.sent)
	}
}
