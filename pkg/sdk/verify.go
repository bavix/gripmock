package sdk

// TestingT is the minimal interface for test assertions.
// Compatible with *testing.T, Ginkgo's *types.GinkgoTInterface, etc.
type TestingT interface {
	Error(args ...any)
	Fail()
}

// HistoryReader provides read access to recorded gRPC calls.
type HistoryReader interface {
	All() []CallRecord
	Count() int
	FilterByMethod(service, method string) []CallRecord
}

// Verifier provides assertion methods for call verification.
type Verifier interface {
	// Method narrows verification to a specific service and method.
	Method(service, method string) MethodVerifier
	// Total asserts the total number of recorded calls.
	Total(t TestingT, want int)
}

// MethodVerifier verifies calls for a specific method.
type MethodVerifier interface {
	// Called asserts the method was called exactly n times.
	Called(t TestingT, n int)
	// Never asserts the method was never called.
	Never(t TestingT)
}

type verifier struct {
	recorder *InMemoryRecorder
}

func (v *verifier) Method(service, method string) MethodVerifier {
	return &methodVerifier{recorder: v.recorder, service: service, method: method}
}

func (v *verifier) Total(t TestingT, want int) {
	got := v.recorder.Count()
	if got != want {
		t.Error("expected ", want, " total calls, got ", got)
		t.Fail()
	}
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
