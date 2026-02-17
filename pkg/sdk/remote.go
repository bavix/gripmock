package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type remoteMock struct {
	conn          *grpc.ClientConn
	addr          string
	restBaseURL   string
	httpClient    *http.Client
	session       string
	sessionTTL    time.Duration
	ttlTimer      *time.Timer
	expectedTotal atomic.Int32
	stubIDsMu     sync.Mutex
	stubIDs       []uuid.UUID
}

func (m *remoteMock) Conn() *grpc.ClientConn { return m.conn }
func (m *remoteMock) Addr() string           { return m.addr }
func (m *remoteMock) History() HistoryReader { return &remoteHistory{mock: m} }
func (m *remoteMock) Verify() Verifier       { return &remoteVerifier{mock: m} }
func (m *remoteMock) Stub(service, method string) StubBuilder {
	return m.stubBuilderCore(service, method)
}
func (m *remoteMock) Close() error {
	if m.ttlTimer != nil {
		m.ttlTimer.Stop()
	}

	if err := m.cleanupStubs(); err != nil {
		return err
	}

	if m.conn != nil {
		_ = m.conn.Close()
		m.conn = nil
	}
	return nil
}

func (m *remoteMock) armSessionTTL() {
	if m.session == "" || m.sessionTTL <= 0 {
		return
	}

	m.ttlTimer = time.AfterFunc(m.sessionTTL, func() {
		_ = m.deleteSessionStubs()
	})
}

func (m *remoteMock) popStubIDs() []uuid.UUID {
	m.stubIDsMu.Lock()
	defer m.stubIDsMu.Unlock()

	if len(m.stubIDs) == 0 {
		return nil
	}

	ids := slices.Clone(m.stubIDs)
	m.stubIDs = nil

	return ids
}

func (m *remoteMock) cleanupStubs() error {
	if m.session != "" {
		return m.deleteSessionStubs()
	}

	return m.deleteOwnedStubs()
}

func (m *remoteMock) deleteOwnedStubs() error {
	ids := m.popStubIDs()
	if len(ids) == 0 {
		return nil
	}

	return m.batchDelete(ids)
}

func (m *remoteMock) deleteSessionStubs() error {
	if m.session == "" {
		return nil
	}

	apiURL, err := url.JoinPath(m.restBaseURL, "api/stubs")
	if err != nil {
		return fmt.Errorf("sdk: failed to build stubs list URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return fmt.Errorf("sdk: failed to create stubs list request: %w", err)
	}
	req.Header.Set("X-Gripmock-Session", m.session)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sdk: failed to list stubs by session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sdk: list stubs failed with status %d", resp.StatusCode)
	}

	var stubs []struct {
		ID      uuid.UUID `json:"id"`
		Session string    `json:"session,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&stubs); err != nil {
		return fmt.Errorf("sdk: failed to decode stubs list: %w", err)
	}

	ids := make([]uuid.UUID, 0, len(stubs))
	for _, s := range stubs {
		if s.Session == m.session && s.ID != uuid.Nil {
			ids = append(ids, s.ID)
		}
	}

	if len(ids) == 0 {
		return nil
	}

	return m.batchDelete(ids)
}

func (m *remoteMock) batchDelete(ids []uuid.UUID) error {

	body, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("sdk: failed to marshal stub IDs: %w", err)
	}

	apiURL, err := url.JoinPath(m.restBaseURL, "api/stubs/batchDelete")
	if err != nil {
		return fmt.Errorf("sdk: failed to build batch delete URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sdk: failed to create batch delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if m.session != "" {
		req.Header.Set("X-Gripmock-Session", m.session)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sdk: failed to batch delete stubs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("sdk: batch delete stubs failed with status %d", resp.StatusCode)
	}

	return nil
}

// remoteHistory fetches call history from GET /api/history.
type remoteHistory struct {
	mock *remoteMock
}

func (r *remoteHistory) All() []CallRecord {
	apiURL, err := url.JoinPath(r.mock.restBaseURL, "api/history")
	if err != nil {
		return nil
	}

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil
	}
	if r.mock.session != "" {
		req.Header.Set("X-Gripmock-Session", r.mock.session)
	}
	resp, err := r.mock.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var list []struct {
		Service   *string         `json:"service"`
		Method    *string         `json:"method"`
		Request   *map[string]any `json:"request"`
		Response  *map[string]any `json:"response"`
		Error     *string         `json:"error"`
		StubID    *string         `json:"stubId"`
		Timestamp *time.Time      `json:"timestamp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil
	}
	out := make([]CallRecord, len(list))
	for i, c := range list {
		out[i] = CallRecord{
			Service:   ptrVal(c.Service),
			Method:    ptrVal(c.Method),
			Request:   ptrMapVal(c.Request),
			Response:  ptrMapVal(c.Response),
			Error:     ptrVal(c.Error),
			StubID:    ptrVal(c.StubID),
			Timestamp: ptrTimeVal(c.Timestamp),
		}
	}
	return out
}

func (r *remoteHistory) Count() int { return len(r.All()) }
func (r *remoteHistory) FilterByMethod(svc, m string) []CallRecord {
	var out []CallRecord
	for _, c := range r.All() {
		if c.Service == svc && c.Method == m {
			out = append(out, c)
		}
	}
	return out
}

func ptrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func ptrMapVal(m *map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	return *m
}
func ptrTimeVal(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// remoteVerifier verifies via POST /api/verify.
type remoteVerifier struct {
	mock *remoteMock
}

func (v *remoteVerifier) Method(service, method string) MethodVerifier {
	return &remoteMethodVerifier{mock: v.mock, service: service, method: method}
}

func (v *remoteVerifier) Total(t TestingT, want int) {
	calls := (&remoteHistory{mock: v.mock}).All()
	got := len(calls)
	if got != want {
		t.Error("expected ", want, " total calls, got ", got)
		t.Fail()
	}
}

func (v *remoteVerifier) VerifyStubTimes(t TestingT) {
	if err := v.VerifyStubTimesErr(); err != nil {
		t.Error(err)
		t.Fail()
	}
}

func (v *remoteVerifier) VerifyStubTimesErr() error {
	want := int(v.mock.expectedTotal.Load())
	if want == 0 {
		return nil
	}
	calls := (&remoteHistory{mock: v.mock}).All()
	got := len(calls)
	if got != want {
		return fmt.Errorf("gripmock: expected %d total calls (from stub Times), got %d", want, got)
	}
	return nil
}

type remoteMethodVerifier struct {
	mock    *remoteMock
	service string
	method  string
}

func (mv *remoteMethodVerifier) Called(t TestingT, n int) {
	body, _ := json.Marshal(map[string]any{
		"service":       mv.service,
		"method":        mv.method,
		"expectedCount": n,
	})
	apiURL, err := url.JoinPath(mv.mock.restBaseURL, "api/verify")
	if err != nil {
		t.Error("gripmock: verify request failed: ", err)
		t.Fail()
		return
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		t.Error("gripmock: verify request failed: ", err)
		t.Fail()
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if mv.mock.session != "" {
		req.Header.Set("X-Gripmock-Session", mv.mock.session)
	}
	resp, err := mv.mock.httpClient.Do(req)
	if err != nil {
		t.Error("gripmock: verify request failed: ", err)
		t.Fail()
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusBadRequest {
		var errBody struct {
			Message *string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := "verification failed"
		if errBody.Message != nil {
			msg = *errBody.Message
		}
		t.Error(msg)
		t.Fail()
	}
}

func (mv *remoteMethodVerifier) Never(t TestingT) {
	mv.Called(t, 0)
}

func (m *remoteMock) addStub(stub *stuber.Stub) {
	if stub.Options.Times > 0 {
		m.expectedTotal.Add(int32(stub.Options.Times))
	}
	body, err := json.Marshal([]*stuber.Stub{stub})
	if err != nil {
		panic("sdk: failed to marshal stub: " + err.Error())
	}
	apiURL, err := url.JoinPath(m.restBaseURL, "api/stubs")
	if err != nil {
		panic("sdk: failed to build stubs URL: " + err.Error())
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		panic("sdk: failed to create request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	if m.session != "" {
		req.Header.Set("X-Gripmock-Session", m.session)
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		panic("sdk: failed to add stub via REST: " + err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		panic(fmt.Sprintf("sdk: add stub failed with status %d", resp.StatusCode))
	}

	m.stubIDsMu.Lock()
	m.stubIDs = append(m.stubIDs, stub.ID)
	m.stubIDsMu.Unlock()
}

func sessionUnaryInterceptor(session string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if session != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-gripmock-session", session)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func sessionStreamInterceptor(session string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if session != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-gripmock-session", session)
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}

func timeoutUnaryInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if timeout > 0 {
			if _, ok := ctx.Deadline(); !ok {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func timeoutStreamInterceptor(timeout time.Duration) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if timeout > 0 {
			if _, ok := ctx.Deadline(); !ok {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}

func runRemote(ctx context.Context, o *options) (Mock, error) {
	o.remoteAddr = normalizeRemoteAddr(o.remoteAddr)
	o.remoteRestURL = normalizeRemoteRestURL(o.remoteRestURL)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	unaryInterceptors := []grpc.UnaryClientInterceptor{timeoutUnaryInterceptor(o.grpcTimeout)}
	streamInterceptors := []grpc.StreamClientInterceptor{timeoutStreamInterceptor(o.grpcTimeout)}
	if o.session != "" {
		sess := o.session
		unaryInterceptors = append(unaryInterceptors, sessionUnaryInterceptor(sess))
		streamInterceptors = append(streamInterceptors, sessionStreamInterceptor(sess))
	}

	opts = append(opts,
		grpc.WithChainUnaryInterceptor(unaryInterceptors...),
		grpc.WithChainStreamInterceptor(streamInterceptors...),
	)

	conn, err := grpc.NewClient("passthrough:///"+o.remoteAddr, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to remote gripmock at %s", o.remoteAddr)
	}
	if err := waitForHealthy(ctx, conn, o.healthyTimeout); err != nil {
		_ = conn.Close()
		return nil, err
	}
	rm := &remoteMock{
		conn:        conn,
		addr:        o.remoteAddr,
		restBaseURL: o.remoteRestURL,
		httpClient:  o.httpClient,
		session:     o.session,
		sessionTTL:  o.sessionTTL,
	}

	if rm.session != "" {
		if err := rm.deleteSessionStubs(); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}

	rm.armSessionTTL()

	return rm, nil
}

func (m *remoteMock) stubBuilderCore(service, method string) *stubBuilderCore {
	return &stubBuilderCore{
		service:  service,
		method:   method,
		onCommit: func(stub *stuber.Stub) { m.addStub(stub) },
	}
}
