package sdk

import (
	"strings"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// mockServer wraps *Server to implement the v1 Mock interface.
type mockServer struct{ *Server }

//nolint:ireturn
func (m *mockServer) History() HistoryReader { return m.mock.History() }

func (s *Server) Addr() string { return s.mock.Addr() }

// Stub creates a v1-compatible StubBuilder that delegates to v2 expectations.
//
// Deprecated: use ExpectUnary, ExpectServerStream, ExpectClientStream,
// ExpectBidirectionalStream instead.
func (s *Server) Stub(service, method string) StubBuilder { //nolint:ireturn
	if strings.TrimSpace(service) == "" || strings.TrimSpace(method) == "" {
		panic("gripmock: service and method must be non-empty")
	}

	return &stubBuilderCore{
		service: service,
		method:  method,
		onCommit: func(stub *stuber.Stub) error {
			s.trackExpectation(stub)

			return nil
		},
	}
}

// Verify returns a v1-compatible Verifier.
//
// Deprecated: use ExpectationsWereMet, Called, TotalCalls, History instead.
func (s *Server) Verify() Verifier { //nolint:ireturn
	return &serverVerifier{srv: s}
}

// serverVerifier wraps *Server to implement the v1 Verifier interface.
type serverVerifier struct {
	srv *Server
}

func (v *serverVerifier) Method(service, method string) MethodVerifier { //nolint:ireturn
	return &serverMethodVerifier{srv: v.srv, service: service, method: method}
}

func (v *serverVerifier) Total(t TestingT, want int) {
	got := v.srv.TotalCalls()
	if got != want {
		t.Error("expected ", want, " total calls, got ", got)
		t.Fail()
	}
}

func (v *serverVerifier) VerifyStubTimes(t TestingT) {
	if err := v.srv.ExpectationsWereMet(); err != nil {
		t.Error(err)
		t.Fail()
	}
}

func (v *serverVerifier) VerifyStubTimesErr() error {
	return v.srv.ExpectationsWereMet()
}

// serverMethodVerifier wraps *Server for per-method verification.
type serverMethodVerifier struct {
	srv     *Server
	service string
	method  string
}

func (mv *serverMethodVerifier) Called(t TestingT, n int) {
	got := mv.srv.Called("/" + mv.service + "/" + mv.method)
	if got != n {
		t.Error("expected ", mv.service, "/", mv.method, " called ", n, " times, got ", got)
		t.Fail()
	}
}

func (mv *serverMethodVerifier) Never(t TestingT) {
	mv.Called(t, 0)
}
