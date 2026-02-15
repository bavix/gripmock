package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
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
	expectedTotal atomic.Int32
}

func (m *remoteMock) Conn() *grpc.ClientConn { return m.conn }
func (m *remoteMock) Addr() string           { return m.addr }
func (m *remoteMock) History() HistoryReader  { return &remoteHistory{mock: m} }
func (m *remoteMock) Verify() Verifier       { return &remoteVerifier{mock: m} }
func (m *remoteMock) Stub(service, method string) StubBuilder {
	return m.stubBuilderCore(service, method)
}
func (m *remoteMock) Close() error {
	if m.conn != nil {
		_ = m.conn.Close()
		m.conn = nil
	}
	return nil
}

// remoteHistory fetches call history from GET /api/history.
type remoteHistory struct {
	mock *remoteMock
}

func (r *remoteHistory) All() []CallRecord {
	apiURL := r.mock.restBaseURL + "/api/history"
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
	apiURL := mv.mock.restBaseURL + "/api/verify"
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
	if m.session != "" {
		stub.Session = m.session
	}
	if stub.Options.Times > 0 {
		m.expectedTotal.Add(int32(stub.Options.Times))
	}
	body, err := json.Marshal([]*stuber.Stub{stub})
	if err != nil {
		panic("sdk: failed to marshal stub: " + err.Error())
	}
	apiURL := m.restBaseURL + "/api/stubs"
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

func runRemote(ctx context.Context, o *options) (Mock, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if o.session != "" {
		sess := o.session
		opts = append(opts,
			grpc.WithUnaryInterceptor(sessionUnaryInterceptor(sess)),
			grpc.WithStreamInterceptor(sessionStreamInterceptor(sess)),
		)
	}
	conn, err := grpc.NewClient("passthrough:///"+o.remoteAddr, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to remote gripmock at %s", o.remoteAddr)
	}
	if err := waitForHealthy(ctx, conn, o.healthyTimeout); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &remoteMock{
		conn:        conn,
		addr:        o.remoteAddr,
		restBaseURL: o.remoteRestURL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		session:     o.session,
	}, nil
}

func (m *remoteMock) stubBuilderCore(service, method string) *stubBuilderCore {
	return &stubBuilderCore{
		service:  service,
		method:  method,
		onCommit: func(stub *stuber.Stub) { m.addStub(stub) },
	}
}
