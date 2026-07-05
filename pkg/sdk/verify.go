package sdk

import (
	"context"
	"maps"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cockroachdb/errors"
)

// TestingT is the minimal interface for test assertions.
// Compatible with *testing.T, Ginkgo's *types.GinkgoTInterface, etc.
type TestingT interface {
	Error(args ...any)
	Fail()
	Context() context.Context
	Cleanup(f func())
}

// HistoryReader provides read access to recorded gRPC calls.
type HistoryReader interface {
	All() []CallRecord
	AllContext(ctx context.Context) ([]CallRecord, error)
	Count() int
	CountContext(ctx context.Context) (int, error)
	FilterByMethod(service, method string) []CallRecord
	FilterByMethodContext(ctx context.Context, service, method string) ([]CallRecord, error)
}

// Verifier provides assertion methods for call verification.
//
// Deprecated: use Server.Called, Server.TotalCalls, Server.ExpectationsWereMet instead.
type Verifier interface {
	// Method narrows verification to a specific service and method.
	Method(service, method string) MethodVerifier
	// Total asserts the total number of recorded calls.
	Total(t TestingT, want int)
	// VerifyStubTimes verifies total calls equal the sum of Times from stubs added via Stub().
	// Use when all stubs have finite Times (no Times(0)); otherwise use Total(t, n).
	// No-op when no stubs with Times > 0 were added.
	VerifyStubTimes(t TestingT)
	// VerifyStubTimesErr returns an error if total calls don't match the sum of Times from stubs.
	VerifyStubTimesErr() error
}

// MethodVerifier verifies calls for a specific method.
//
// Deprecated: use Server.Called instead.
type MethodVerifier interface {
	// Called asserts the method was called exactly n times.
	Called(t TestingT, n int)
	// Never asserts the method was never called.
	Never(t TestingT)
}

type verifier struct {
	recorder      *InMemoryRecorder
	expectedTotal *atomic.Int32
	expectedByMth map[string]int
	expectedMu    *sync.Mutex
}

func (v *verifier) Method(service, method string) MethodVerifier { //nolint:ireturn
	if strings.TrimSpace(service) == "" || strings.TrimSpace(method) == "" {
		panic("sdk.Verifier.Method: service and method must be non-empty")
	}

	return &methodVerifier{recorder: v.recorder, service: service, method: method}
}

func (v *verifier) Total(t TestingT, want int) {
	got := v.recorder.Count()
	if got != want {
		t.Error("expected ", want, " total calls, got ", got)
		t.Fail()
	}
}

func (v *verifier) VerifyStubTimes(t TestingT) {
	if err := v.VerifyStubTimesErr(); err != nil {
		t.Error(err)
		t.Fail()
	}
}

func (v *verifier) VerifyStubTimesErr() error {
	if v.expectedTotal == nil {
		return nil
	}

	want := int(v.expectedTotal.Load())
	if want == 0 {
		return nil
	}

	if v.expectedMu != nil {
		v.expectedMu.Lock()
		perMethod := maps.Clone(v.expectedByMth)
		v.expectedMu.Unlock()

		for key, expected := range perMethod {
			service, method, ok := splitMethodKey(key)
			if !ok {
				return errors.Wrapf(ErrVerificationFailed, "invalid expected method key %q", key)
			}

			got := len(v.recorder.FilterByMethod(service, method))
			if got != expected {
				return errors.Wrapf(ErrVerificationFailed, "expected %d calls for %s/%s (from stub Times), got %d", expected, service, method, got)
			}
		}

		return nil
	}

	got := v.recorder.Count()
	if got != want {
		return errors.Wrapf(ErrVerificationFailed, "expected %d total calls (from stub Times), got %d", want, got)
	}

	return nil
}

type methodVerifier struct {
	recorder *InMemoryRecorder
	service  string
	method   string
}

func (mv *methodVerifier) Called(t TestingT, n int) {
	got := len(mv.recorder.FilterByMethod(mv.service, mv.method))
	if got != n {
		t.Error("expected ", mv.service, "/", mv.method, " called ", n, " times, got ", got)
		t.Fail()
	}
}

func (mv *methodVerifier) Never(t TestingT) {
	mv.Called(t, 0)
}
