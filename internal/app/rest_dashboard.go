package app

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
	"github.com/bavix/gripmock/v3/internal/infra/session"
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
		CoveredMethods:     payload.CoveredMethods,
		TotalMethods:       payload.TotalMethods,
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
	h.writeResponse(r.Context(), w, rest.Sessions{Sessions: h.mergedSessions()})
}

func (h *RestServer) mergedSessions() []string {
	seen := make(map[string]struct{})
	merged := make([]string, 0)

	add := func(ids []string) {
		for _, id := range ids {
			if id == "" {
				continue
			}

			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				merged = append(merged, id)
			}
		}
	}

	add(h.budgerigar.Sessions())
	add(session.IDs())
	sort.Strings(merged)

	return merged
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
func (h *RestServer) ListHistory(w http.ResponseWriter, r *http.Request, params rest.ListHistoryParams) {
	if h.history == nil {
		h.writeResponse(r.Context(), w, rest.HistoryList{})

		return
	}

	calls, total := historyWindow(h.history, history.FilterOpts{
		Session:   muxmiddleware.FromRequest(r),
		Service:   stringFromPtr(params.Service),
		Method:    stringFromPtr(params.Method),
		ErrorOnly: params.Error != nil && *params.Error,
	}, intFromPtr(params.Limit), intFromPtr(params.Offset))

	w.Header().Set("X-Total-Count", strconv.Itoa(total))

	out := make(rest.HistoryList, len(calls))
	for i, c := range calls {
		out[i] = historyCallRecordToRest(c)
	}

	h.writeResponse(r.Context(), w, out)
}

type historyCounter interface {
	CountFilter(opts history.FilterOpts) int
}

type historyWindower interface {
	FilterWindow(opts history.FilterOpts, limit, offset int) ([]history.CallRecord, int)
}

// historyWindow returns one page and the total, without materializing the rest
// when the reader can page for itself.
func historyWindow(reader history.Reader, opts history.FilterOpts, limit, offset int) ([]history.CallRecord, int) {
	if windower, ok := reader.(historyWindower); ok {
		return windower.FilterWindow(opts, limit, offset)
	}

	calls := reader.Filter(opts)
	total := len(calls)

	end := max(total-max(offset, 0), 0)
	start := 0

	if limit > 0 {
		start = max(end-limit, 0)
	}

	return calls[start:end], total
}

func countHistory(reader history.Reader, opts history.FilterOpts) int {
	if counter, ok := reader.(historyCounter); ok {
		return counter.CountFilter(opts)
	}

	return len(reader.Filter(opts))
}

// PurgeHistory deletes recorded calls, scoped to the request's session when set.
func (h *RestServer) PurgeHistory(w http.ResponseWriter, r *http.Request) {
	session := muxmiddleware.FromRequest(r)

	result := rest.HistoryPurged{}
	if session != "" {
		result.Session = &session
	}

	if h.history == nil {
		h.writeResponse(r.Context(), w, result)

		return
	}

	result.DeletedCount = h.purgeHistoryRecords(session)

	h.writeResponse(r.Context(), w, result)
}

func (h *RestServer) purgeHistoryRecords(session string) int {
	if session != "" {
		cleaner, ok := h.history.(history.SessionCleaner)
		if !ok {
			return 0
		}

		return cleaner.DeleteSession(session)
	}

	clearer, ok := h.history.(interface{ Clear() })
	if !ok {
		return 0
	}

	deleted := h.history.Count()

	clearer.Clear()

	return deleted
}

func restCallMessages(c history.CallRecord, r *rest.CallRecord) {
	if len(c.Requests) > 0 {
		r.Requests = &c.Requests
	}

	if len(c.Responses) > 0 {
		r.Responses = &c.Responses
	}

	if len(c.ResponseHeaders) > 0 {
		r.ResponseHeaders = c.ResponseHeaders
	}
}

func historyCallRecordToRest(c history.CallRecord) rest.CallRecord {
	r := rest.CallRecord{
		Service: new(c.Service),
		Method:  new(c.Method),
	}

	if c.StubID != uuid.Nil {
		r.StubId = &c.StubID
	}

	// The field is declared in the schema and was never filled, so every record
	// came back session-less and no client could tell whose call it was.
	if c.Session != "" {
		r.Session = &c.Session
	}

	restCallMessages(c, &r)

	if c.Error != "" {
		r.Error = &c.Error
	}

	if c.Code != 0 {
		code := int(c.Code)
		r.Code = &code
	}

	if !c.Timestamp.IsZero() {
		r.ElapsedMs = &c.ElapsedMS
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

	actual := countHistory(h.history, history.FilterOpts{
		Service: req.Service,
		Method:  req.Method,
		Session: muxmiddleware.FromRequest(r),
	})
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
