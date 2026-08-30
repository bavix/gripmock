package sdk

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// TestingT is the minimal interface for test assertions.
type TestingT interface {
	Error(args ...any)
	Fail()
	Context() context.Context
	Cleanup(f func())
}

// Server is a running gRPC mock server (v2 API).
type Server struct {
	t TestingT

	bg context.Context //nolint:containedctx

	embedded *embeddedMock
	remote   *remoteMock

	budgerigar *stuber.Budgerigar
	recorder   *history.MemoryStore

	session string

	pending   []*stuber.Stub
	pendingMu sync.Mutex
	batchMode bool

	mu           sync.Mutex
	expectations []expectedCall
	verified     bool
}

type expectedCall struct {
	service string
	method  string
	stubID  uuid.UUID
	times   int
}

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

	srv := &Server{
		t:         t,
		bg:        context.WithoutCancel(t.Context()),
		batchMode: o.batchMode,
		session:   o.session,
	}

	err := startServer(t.Context(), o, srv)
	if err != nil {
		return nil, err
	}

	t.Cleanup(func() {
		verr := srv.ExpectationsWereMetContext(srv.bg)
		if verr != nil {
			t.Error(verr)
		}

		cerr := srv.Close()
		if cerr != nil {
			t.Error(cerr)
		}
	})

	return srv, nil
}

// NewServer creates and starts a mock server.
func NewServer(t TestingT, opts ...Option) *Server {
	srv, err := initServer(t, opts...)
	if err != nil {
		panic("gripmock: " + err.Error())
	}

	return srv
}

// Address returns the server address (e.g. "127.0.0.1:PORT").
func (s *Server) Address() string {
	if s.remote != nil {
		return s.remote.addr
	}

	return s.embedded.addr
}

func (s *Server) Conn() *grpc.ClientConn {
	if s.remote != nil {
		return s.remote.conn
	}

	return s.embedded.conn
}

// ExpectUnary terminal: Return, ReturnProto, ReturnError, Run.
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
func (s *Server) ExpectationsWereMet() error {
	return s.ExpectationsWereMetContext(s.readCtx())
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

	_ = s.Flush() //nolint:contextcheck

	if s.remote != nil {
		return s.remoteVerify(ctx, ec)
	}

	return s.embeddedVerify(ec)
}

//nolint:funcorder
func (s *Server) embeddedVerify(ec []expectedCall) error {
	var errs []error

	counts := make(map[uuid.UUID]int)
	for rec := range s.recorder.FilterSeq(history.FilterOpts{}) {
		counts[rec.StubID]++
	}

	for _, e := range ec {
		if e.times == 0 {
			continue
		}

		got := counts[e.stubID]
		if got != e.times {
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
	calls, err := s.remote.history().AllContext(ctx)
	if err != nil {
		return err
	}

	counts := make(map[uuid.UUID]int)
	for _, rec := range calls {
		counts[rec.StubID]++
	}

	var errs []error

	for _, e := range ec {
		if e.times == 0 {
			continue
		}

		if got := counts[e.stubID]; got != e.times {
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

// Called returns the number of times a method was called.
func (s *Server) Called(fullMethod string) int {
	service, method := splitMethodName(fullMethod)
	if s.remote != nil {
		calls, err := s.remote.history().FilterByMethodContext(s.readCtx(), service, method)
		s.reportRemoteErr("Called", err)

		return len(calls)
	}

	return s.recorder.CountFilter(history.FilterOpts{
		Service: service,
		Method:  method,
		Session: s.session,
	})
}

func (s *Server) TotalCalls() int {
	if s.remote != nil {
		count, err := s.remote.history().CountContext(s.readCtx())
		s.reportRemoteErr("TotalCalls", err)

		return count
	}

	if s.session == "" {
		return s.recorder.Count()
	}

	return s.recorder.CountFilter(history.FilterOpts{Session: s.session})
}

func (s *Server) History() []CallRecord {
	if s.remote != nil {
		calls, err := s.remote.history().AllContext(s.readCtx())
		s.reportRemoteErr("History", err)

		return calls
	}

	records := s.recorder.All()
	if s.session != "" {
		records = s.recorder.Filter(history.FilterOpts{Session: s.session})
	}

	result := make([]CallRecord, len(records))
	copy(result, records)

	return result
}

//nolint:funcorder
func (s *Server) reportRemoteErr(op string, err error) {
	if err != nil {
		s.t.Error("gripmock: ", op, " failed to read remote history: ", err)
	}
}

func (s *Server) Reset() {
	s.mu.Lock()
	s.expectations = nil
	s.verified = false
	s.mu.Unlock()

	s.pendingMu.Lock()
	s.pending = nil
	s.pendingMu.Unlock()

	if s.budgerigar != nil {
		s.budgerigar.Clear()
	}

	if s.remote != nil {
		err := s.remote.deleteOwnedStubs()
		if err != nil {
			s.t.Error("gripmock: Reset failed to delete remote stubs: ", err)
		}

		err = s.remote.purgeHistory()
		if err != nil {
			s.t.Error("gripmock: Reset failed to purge remote history: ", err)
		}
	}

	if s.recorder != nil {
		s.recorder.Clear()
	}
}

// Close flushes pending stubs in batch mode and shuts down the server.
func (s *Server) Close() error {
	if s.remote != nil {
		if s.batchMode {
			_ = s.Flush()
		}

		return s.remote.Close()
	}

	return s.embedded.Close()
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

func (s *Server) readCtx() context.Context {
	if s.t.Context().Err() != nil {
		return s.bg
	}

	return s.t.Context()
}

func (s *Server) trackExpectation(stub *stuber.Stub) {
	s.mu.Lock()
	if stub.Options.Times > 0 {
		s.expectations = append(s.expectations, expectedCall{
			service: stub.Service,
			method:  stub.Method,
			stubID:  stub.ID,
			times:   stub.Options.Times,
		})
	}
	s.mu.Unlock()

	s.registerStub(stub)
}

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

func (s *Server) upsertStub(stub *stuber.Stub) {
	if s.remote != nil && s.batchMode {
		s.pendingMu.Lock()
		defer s.pendingMu.Unlock()

		for i, queued := range s.pending {
			if queued.ID == stub.ID {
				s.pending[i] = stub

				return
			}
		}

		s.pending = append(s.pending, stub)

		return
	}

	s.registerStub(stub)
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

func startServer(ctx context.Context, o *options, srv *Server) error {
	if o.remoteAddr != "" {
		rm, err := runRemote(ctx, o)
		if err != nil {
			return err
		}

		srv.remote = rm

		return nil
	}

	err := resolveEmbeddedDescriptors(ctx, o)
	if err != nil {
		return err
	}

	em, err := runEmbedded(ctx, o)
	if err != nil {
		return err
	}

	srv.embedded = em
	srv.budgerigar = em.budgerigar
	srv.recorder = em.recorder

	return nil
}

func resolveEmbeddedDescriptors(ctx context.Context, o *options) error {
	if len(o.protoPaths) > 0 {
		fds, err := compileProtoFiles(ctx, o.protoPaths)
		if err != nil {
			return err
		}

		o.appendDescriptorFiles(fds.GetFile())
	}

	if len(o.descriptorFiles) == 0 && o.mockFromAddr == "" {
		return ErrDescriptorsRequired
	}

	if o.mockFromAddr != "" {
		fds, err := resolveDescriptorsFromReflection(ctx, o.mockFromAddr)
		if err != nil {
			return err
		}

		o.descriptorFiles = fds.GetFile()
	}

	return nil
}
