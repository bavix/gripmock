package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
)

func (h *RestServer) Readiness(w http.ResponseWriter, r *http.Request) {
	if !h.ok.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		h.writeResponse(r.Context(), w, rest.MessageOK{Message: "not ready", Time: time.Now()})

		return
	}

	h.liveness(r.Context(), w)
}

// Liveness handles the liveness probe endpoint.
func (h *RestServer) Liveness(w http.ResponseWriter, r *http.Request) {
	h.liveness(r.Context(), w)
}

// DashboardOverview returns aggregated lightweight metrics for admin dashboard.
func (h *RestServer) DashboardOverview(w http.ResponseWriter, r *http.Request) {
	payload := h.dashboardPayload(r)

	response := rest.DashboardOverview{
		TotalServices:      payload.TotalServices,
		TotalStubs:         payload.TotalStubs,
		UsedStubs:          payload.UsedStubs,
		UnusedStubs:        payload.UnusedStubs,
		TotalSessions:      payload.TotalSessions,
		RuntimeDescriptors: payload.RuntimeDescriptors,
		TotalHistory:       payload.TotalHistory,
		HistoryErrors:      payload.HistoryErrors,
	}

	h.writeResponse(r.Context(), w, response)
}

// Dashboard returns combined counters and runtime metadata for dashboard page.
func (h *RestServer) Dashboard(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, h.dashboardPayload(r))
}

// SessionsList returns distinct non-empty session IDs for UI selectors.
func (h *RestServer) SessionsList(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, rest.Sessions{Sessions: h.budgerigar.Sessions()})
}

// DashboardInfo returns build metadata and runtime process information.
func (h *RestServer) DashboardInfo(w http.ResponseWriter, r *http.Request) {
	payload := h.dashboardPayload(r)

	h.writeResponse(r.Context(), w, rest.DashboardInfo{
		AppName:            payload.AppName,
		Version:            payload.Version,
		GoVersion:          payload.GoVersion,
		Compiler:           payload.Compiler,
		Goos:               payload.Goos,
		Goarch:             payload.Goarch,
		NumCPU:             payload.NumCPU,
		StartedAt:          payload.StartedAt,
		UptimeSeconds:      payload.UptimeSeconds,
		Ready:              payload.Ready,
		HistoryEnabled:     payload.HistoryEnabled,
		TotalServices:      payload.TotalServices,
		TotalStubs:         payload.TotalStubs,
		TotalSessions:      payload.TotalSessions,
		RuntimeDescriptors: payload.RuntimeDescriptors,
	})
}

// ListHistory returns recorded gRPC calls.
func (h *RestServer) ListHistory(w http.ResponseWriter, r *http.Request) {
	if h.history == nil {
		h.writeResponse(r.Context(), w, rest.HistoryList{})

		return
	}

	calls := h.history.Filter(history.FilterOpts{Session: muxmiddleware.FromRequest(r)})

	out := make(rest.HistoryList, len(calls))
	for i, c := range calls {
		out[i] = historyCallRecordToRest(c)
	}

	h.writeResponse(r.Context(), w, out)
}

func historyCallRecordToRest(c history.CallRecord) rest.CallRecord {
	r := rest.CallRecord{
		Service: new(c.Service),
		Method:  new(c.Method),
	}

	if c.StubID != uuid.Nil {
		r.StubId = &c.StubID
	}

	if len(c.Requests) > 0 {
		r.Requests = &c.Requests
		r.Request = &c.Requests[0]
	} else if c.Request != nil {
		r.Request = &c.Request
	}

	if len(c.Responses) > 0 {
		r.Responses = &c.Responses
		r.Response = &c.Responses[0]
	} else if c.Response != nil {
		r.Response = &c.Response
	}

	if c.Error != "" {
		r.Error = &c.Error
	}

	if c.Code != 0 {
		code := int(c.Code)
		r.Code = &code
	}

	if !c.Timestamp.IsZero() {
		r.Timestamp = &c.Timestamp
	}

	return r
}

// VerifyCalls verifies that a method was called the expected number of times.
func (h *RestServer) VerifyCalls(w http.ResponseWriter, r *http.Request) {
	if h.history == nil {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponse(r.Context(), w, rest.VerifyError{Message: new("history is disabled")})

		return
	}

	var req rest.VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponseError(r.Context(), w, errors.Wrap(err, "invalid verify request"))

		return
	}

	calls := h.history.Filter(history.FilterOpts{
		Service: req.Service,
		Method:  req.Method,
		Session: muxmiddleware.FromRequest(r),
	})

	actual := len(calls)
	if actual != req.ExpectedCount {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponse(r.Context(), w, rest.VerifyError{
			Message:  new(fmt.Sprintf("expected %s/%s to be called %d times, got %d", req.Service, req.Method, req.ExpectedCount, actual)),
			Expected: &req.ExpectedCount,
			Actual:   &actual,
		})

		return
	}

	h.writeResponse(r.Context(), w, rest.MessageOK{Message: "ok", Time: time.Now()})
}

// AddStub inserts new stubs.
