package sdk

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// Server is a running gRPC mock server (v2 API).
// Create via NewServer, which calls v1 Run() internally.
//
// Thread safety: All exported methods are safe for concurrent use.
// Each NewServer call creates an independent instance — safe for t.Parallel().
//
// Usage:
//
//	srv := sdk.NewServer(t, sdk.WithProtoFiles("service.proto"))
//	defer srv.Close()
//
//	srv.ExpectUnary("/svc/Method").
//	    Match("field", "value").
//	    Return("responseField", "responseValue")
//
//	client := pb.NewServiceClient(srv.Conn())
//	resp, _ := client.Method(t.Context(), &pb.Request{Field: "value"})
type Server struct {
	t    TestingT
	mock Mock

	// Direct access for fast path (embedded mode)
	budgerigar *stuber.Budgerigar
	recorder   *history.MemoryStore

	// Remote handle (non-nil in remote mode)
	remote *remoteMock

	// Batch queue: stubs are accumulated in remote mode only when WithBatch() is used.
	// Otherwise each terminal method sends stubs immediately.
	pending   []*stuber.Stub
	pendingMu sync.Mutex
	batchMode bool

	mu           sync.Mutex
	expectations []expectedCall // protected by mu
	verified     bool           // protected by mu
}

type expectedCall struct {
	service string
	method  string
	times   int
}

// initServer is the shared initialization for both NewServer and Run (v1 compatibility).
func initServer(t TestingT, opts ...Option) (*Server, error) {
	if t == nil {
		panic("gripmock: TestingT must not be nil")
	}

	o := &options{healthyTimeout: defaultHealthyTimeout, sessionTTL: defaultSessionTTL}

	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	if o.httpClient == nil {
		o.httpClient = &http.Client{Timeout: 10 * time.Second} //nolint:mnd
	}

	mock, err := startServer(t.Context(), o)
	if err != nil {
		return nil, err
	}

	srv := &Server{t: t, mock: mock, batchMode: o.batchMode}

	// Extract internals for direct, lock-free stub registration
	switch m := mock.(type) {
	case *embeddedMock:
		srv.budgerigar = m.budgerigar
		srv.recorder = m.recorder
	case *remoteMock:
		srv.remote = m
	}

	t.Cleanup(func() {
		if verr := srv.ExpectationsWereMet(); verr != nil {
			t.Error(verr)
		}

		if cerr := srv.Close(); cerr != nil {
			t.Error(cerr)
		}
	})

	return srv, nil
}

// NewServer creates and starts a mock server.
// Panics on initialization errors. Registers t.Cleanup for auto-verify + close.
// Each call creates an independent server — safe for t.Parallel().
func NewServer(t TestingT, opts ...Option) *Server {
	srv, err := initServer(t, opts...)
	if err != nil {
		panic("gripmock: " + err.Error())
	}

	return srv
}

// Address returns the server address (e.g. "127.0.0.1:PORT").
func (s *Server) Address() string { return s.mock.Addr() }

func (s *Server) Conn() *grpc.ClientConn {
	return s.mock.Conn()
}

// ExpectUnary terminal: Return, ReturnProto, ReturnError, Run.
// Use Delay() inside Return for delayed responses: Return(Delay(100*ms, "msg", "hello")).
func (s *Server) ExpectUnary(fullMethod string) *UnaryExpectation {
	return newUnaryExpectation(s, fullMethod)
}

// ExpectServerStream terminal: SendStream.
func (s *Server) ExpectServerStream(fullMethod string) *ServerStreamExpectation {
	return newServerStreamExpectation(s, fullMethod)
}

// ExpectClientStream terminal: Return, ReturnError.
func (s *Server) ExpectClientStream(fullMethod string) *ClientStreamExpectation {
	return newClientStreamExpectation(s, fullMethod)
}

// ExpectBidirectionalStream terminal: Run.
func (s *Server) ExpectBidirectionalStream(fullMethod string) *BidirectionalExpectation {
	return newBidiExpectation(s, fullMethod)
}

// ExpectationsWereMet checks all expectations with non-zero Times were fulfilled.
// Idempotent: second call returns nil. Thread-safe.
// Flushes pending stubs before verification (important for remote mode).
func (s *Server) ExpectationsWereMet() error {
	return s.ExpectationsWereMetContext(s.t.Context())
}

// ExpectationsWereMetContext is the context-aware version of ExpectationsWereMet.
func (s *Server) ExpectationsWereMetContext(ctx context.Context) error {
	s.mu.Lock()
	if s.verified {
		s.mu.Unlock()

		return nil
	}

	s.verified = true
	ec := make([]expectedCall, len(s.expectations))
	copy(ec, s.expectations)
	s.mu.Unlock()

	// Ensure all pending stubs are sent before verifying
	_ = s.Flush() //nolint:contextcheck

	if s.remote != nil {
		return s.remoteVerify(ctx, ec)
	}

	return s.embeddedVerify(ec)
}

//nolint:funcorder
func (s *Server) embeddedVerify(ec []expectedCall) error {
	var errs []error

	for _, e := range ec {
		if e.times == 0 {
			continue
		}

		got := len(s.recorder.FilterByMethod(e.service, e.method))
		if got < e.times {
			errs = append(errs, &ExpectationNotMetError{
				Service:  e.service,
				Method:   e.method,
				Expected: e.times,
				Actual:   got,
			})
		}
	}

	return joinErrors(errs)
}

//nolint:funcorder
func (s *Server) remoteVerify(ctx context.Context, ec []expectedCall) error {
	var errs []error

	for _, e := range ec {
		if e.times == 0 {
			continue
		}

		client := s.remote.apiWithContext(ctx)
		if err := client.VerifyMethodCalled(e.service, e.method, e.times); err != nil { //nolint:contextcheck
			errs = append(errs, &ExpectationNotMetError{
				Service:  e.service,
				Method:   e.method,
				Expected: e.times,
				Actual:   0,
			})
		}
	}

	return joinErrors(errs)
}

// Called returns the number of times a method was called.
// fullMethod format: "/package.Service/Method".
func (s *Server) Called(fullMethod string) int {
	service, method := splitMethodName(fullMethod)
	if s.remote != nil {
		calls, _ := s.remote.History().FilterByMethodContext(context.TODO(), service, method)

		return len(calls)
	}

	return len(s.recorder.FilterByMethod(service, method))
}

func (s *Server) TotalCalls() int {
	if s.remote != nil {
		count, _ := s.remote.History().CountContext(context.TODO())

		return count
	}

	return s.recorder.Count()
}

func (s *Server) History() []CallRecord {
	if s.remote != nil {
		calls, _ := s.remote.History().AllContext(context.TODO())

		return calls
	}

	records := s.recorder.All()
	result := make([]CallRecord, len(records))
	copy(result, records)

	return result
}

// Reset clears local expectations, pending stubs, and verification state.
// For embedded: also clears budgerigar and history recorder.
// For remote: clears local state only (remote server history is shared).
// Does NOT close the server — register new expectations after Reset.
func (s *Server) Reset() {
	s.mu.Lock()
	s.expectations = nil
	s.verified = false
	s.mu.Unlock()

	s.pending = nil

	if s.budgerigar != nil {
		s.budgerigar.Clear()
		s.recorder = &InMemoryRecorder{}
	}
}

// Close flushes pending stubs in batch mode and shuts down the server.
func (s *Server) Close() error {
	if s.batchMode && s.remote != nil {
		_ = s.Flush()
	}

	return s.mock.Close()
}

// Flush is a no-op for embedded mode and non-batch remote mode.
func (s *Server) Flush() error {
	if s.remote == nil || len(s.pending) == 0 {
		return nil
	}

	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	err := s.remote.commitStubsBatch(s.pending)
	s.pending = nil

	return err
}

// trackExpectation records an expectation and queues/registers the stub.
func (s *Server) trackExpectation(stub *stuber.Stub) {
	s.mu.Lock()
	if stub.Options.Times > 0 {
		s.expectations = append(s.expectations, expectedCall{
			service: stub.Service,
			method:  stub.Method,
			times:   stub.Options.Times,
		})
	}
	s.mu.Unlock()

	s.registerStub(stub)
}

// registerStub registers a stub immediately (embedded) or sends/queues it (remote).
// Thread-safe: in batch mode queues with pendingMu; in immediate mode sends via REST.
func (s *Server) registerStub(stub *stuber.Stub) {
	switch {
	case s.budgerigar != nil:
		s.budgerigar.PutMany(stub)
	case s.remote != nil && s.batchMode:
		s.pendingMu.Lock()
		s.pending = append(s.pending, stub)
		s.pendingMu.Unlock()
	case s.remote != nil:
		_ = s.remote.commitStubsBatch([]*stuber.Stub{stub})
	}
}

func splitMethodName(fullMethod string) (string, string) {
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}

	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[:i], fullMethod[i+1:]
		}
	}

	return "", fullMethod
}

func joinErrors(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		var b strings.Builder

		for i, e := range errs {
			if i > 0 {
				b.WriteString("; ")
			}

			b.WriteString(e.Error())
		}

		return errors.Wrapf(ErrVerificationFailed, "%s", b.String())
	}
}
