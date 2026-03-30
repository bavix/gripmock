package sdk

import (
	"context"
	stderrors "errors"
	"fmt"
	"maps"
	"strings"

	"github.com/bavix/gripmock/v3/pkg/sdk/internal/remoteapi"
)

// remoteHistory fetches call history from GET /api/history.
type remoteHistory struct {
	mock *remoteMock
}

func convertHistory(history []remoteapi.HistoryCall) []CallRecord {
	out := make([]CallRecord, len(history))
	for i, c := range history {
		out[i] = CallRecord{
			Service:   c.Service,
			Method:    c.Method,
			Request:   c.Request,
			Response:  c.Response,
			Error:     c.Error,
			StubID:    c.StubID,
			Timestamp: c.Timestamp,
		}
	}

	return out
}

func (r *remoteHistory) fetchWithClient(client remoteapi.Client) ([]CallRecord, error) {
	history, err := client.FetchHistory()
	if err != nil {
		r.mock.setOpErr(err)
		return nil, err
	}

	return convertHistory(history), nil
}

func (r *remoteHistory) fetch() ([]CallRecord, error) {
	return r.fetchWithClient(r.mock.api())
}

func (r *remoteHistory) All() []CallRecord {
	calls, err := r.fetch()
	if err != nil {
		return nil
	}

	return calls
}

func (r *remoteHistory) AllContext(ctx context.Context) ([]CallRecord, error) {
	return r.fetchWithClient(r.mock.apiWithContext(ctx))
}

func (r *remoteHistory) Count() int {
	calls, err := r.fetch()
	if err != nil {
		return 0
	}

	return len(calls)
}

func (r *remoteHistory) CountContext(ctx context.Context) (int, error) {
	calls, err := r.fetchWithClient(r.mock.apiWithContext(ctx))
	if err != nil {
		return 0, err
	}

	return len(calls), nil
}

func (r *remoteHistory) FilterByMethod(svc, m string) []CallRecord {
	calls, err := r.fetch()
	if err != nil {
		return nil
	}

	var out []CallRecord
	for _, c := range calls {
		if c.Service == svc && c.Method == m {
			out = append(out, c)
		}
	}

	return out
}

func (r *remoteHistory) FilterByMethodContext(ctx context.Context, svc, m string) ([]CallRecord, error) {
	calls, err := r.fetchWithClient(r.mock.apiWithContext(ctx))
	if err != nil {
		return nil, err
	}

	var out []CallRecord
	for _, c := range calls {
		if c.Service == svc && c.Method == m {
			out = append(out, c)
		}
	}

	return out, nil
}

// remoteVerifier verifies via POST /api/verify.
type remoteVerifier struct {
	mock *remoteMock
}

func (v *remoteVerifier) Method(service, method string) MethodVerifier {
	if strings.TrimSpace(service) == "" || strings.TrimSpace(method) == "" {
		panic("sdk.Verifier.Method: service and method must be non-empty")
	}

	return &remoteMethodVerifier{mock: v.mock, service: service, method: method}
}

func (v *remoteVerifier) Total(t TestingT, want int) {
	history := &remoteHistory{mock: v.mock}
	calls, err := history.fetchWithClient(v.mock.apiWithContext(t.Context()))
	if err != nil {
		t.Error("gripmock: failed to fetch history: ", err)
		t.Fail()
		return
	}

	got := len(calls)
	if got != want {
		t.Error("expected ", want, " total calls, got ", got)
		t.Fail()
	}
}

func (v *remoteVerifier) VerifyStubTimes(t TestingT) {
	if err := v.verifyStubTimesErr(v.mock.apiWithContext(t.Context())); err != nil {
		t.Error(err)
		t.Fail()
	}
}

func (v *remoteVerifier) VerifyStubTimesErr() error {
	return v.verifyStubTimesErr(v.mock.api())
}

func (v *remoteVerifier) VerifyStubTimesErrContext(ctx context.Context) error {
	return v.verifyStubTimesErr(v.mock.apiWithContext(ctx))
}

func (v *remoteVerifier) verifyStubTimesErr(client remoteapi.Client) error {
	if opErr := v.mock.getOpErr(); opErr != nil {
		return opErr
	}

	want := int(v.mock.expectedTotal.Load())
	if want == 0 {
		return nil
	}

	v.mock.expectedMu.Lock()
	perMethod := maps.Clone(v.mock.expectedByMth)
	v.mock.expectedMu.Unlock()

	for key, expected := range perMethod {
		service, method, ok := splitMethodKey(key)
		if !ok {
			return fmt.Errorf("gripmock: invalid expected method key %q", key)
		}

		err := client.VerifyMethodCalled(service, method, expected)
		if err != nil {
			return fmt.Errorf("gripmock: expected %d calls for %s/%s: %w", expected, service, method, err)
		}
	}

	return nil
}

type remoteMethodVerifier struct {
	mock    *remoteMock
	service string
	method  string
}

func (mv *remoteMethodVerifier) Called(t TestingT, n int) {
	if opErr := mv.mock.getOpErr(); opErr != nil {
		t.Error("gripmock: operation failed: ", opErr)
		t.Fail()
		return
	}

	err := mv.mock.apiWithContext(t.Context()).VerifyMethodCalled(mv.service, mv.method, n)
	if err == nil {
		return
	}

	var badReq remoteapi.VerifyBadRequestError
	if stderrors.As(err, &badReq) {
		t.Error(badReq.Error())
		t.Fail()
		return
	}

	t.Error("gripmock: verify request failed: ", err)
	t.Fail()
}

func (mv *remoteMethodVerifier) Never(t TestingT) {
	mv.Called(t, 0)
}

func methodKey(service, method string) string {
	return service + "/" + method
}

func splitMethodKey(key string) (service string, method string, ok bool) {
	service, method, ok = strings.Cut(key, "/")
	if !ok || service == "" || method == "" {
		return "", "", false
	}

	return service, method, true
}
