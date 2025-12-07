package v4

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/bavix/gripmock/v3/internal/app/port"
	domain "github.com/bavix/gripmock/v3/internal/domain/types"
	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
	"github.com/bavix/gripmock/v3/internal/infra/matcher"
	validator "github.com/bavix/gripmock/v3/internal/infra/schema"
	"github.com/bavix/gripmock/v3/pkg/plugins"
)

const (
	formatYAML  = "yaml"
	maxListEnd  = 999999
	contentYAML = "application/x-yaml"
	contentJSON = "application/json"
)

// Server wires /api/v4 endpoints.
// Handlers are react-admin compatible: filter/sort/range and Content-Range.
type Server struct {
	stubs     port.StubRepository
	analytics port.AnalyticsRepository
	history   port.HistoryRepository
	plugins   []plugins.PluginWithFuncs
}

func NewServer(
	stubs port.StubRepository,
	analytics port.AnalyticsRepository,
	history port.HistoryRepository,
	pluginInfos []plugins.PluginWithFuncs,
) *Server {
	return &Server{stubs: stubs, analytics: analytics, history: history, plugins: pluginInfos}
}

// Mount mounts the v4 routes under the provided router and base path.
func (s *Server) Mount(r *mux.Router, base string) {
	// Health endpoints (mirror legacy responses)
	r.HandleFunc(base+"/health/liveness", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"message": "ok"})
	}).Methods(http.MethodGet)

	r.HandleFunc(base+"/health/readiness", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"message": "ok"})
	}).Methods(http.MethodGet)

	// Utilities
	r.HandleFunc(base+"/stubs/lint", s.handleLintStubs).Methods(http.MethodPost)
	r.HandleFunc(base+"/stubs/match:dryRun", s.handleDryRunMatch).Methods(http.MethodPost)
	r.HandleFunc(base+"/plugins", s.handlePlugins).Methods(http.MethodGet)
	// export/import endpoints are removed; use POST /v4/stubs and GET /v4/stubs instead

	// Services and methods
	r.HandleFunc(base+"/services", handleListServices).Methods(http.MethodGet)
	r.HandleFunc(base+"/methods", handleListMethods).Methods(http.MethodGet)

	// Stubs CRUD and search
	r.HandleFunc(base+"/stubs", s.handleListStubs).Methods(http.MethodGet)
	r.HandleFunc(base+"/stubs", s.handleCreateStub).Methods(http.MethodPost)
	r.HandleFunc(base+"/stubs/search", s.handleSearchStubs).Methods(http.MethodGet)
	r.HandleFunc(base+"/stubs/{id}", s.handleGetStub).Methods(http.MethodGet)
	r.HandleFunc(base+"/stubs/{id}", s.handleUpdateStub).Methods(http.MethodPut)
	r.HandleFunc(base+"/stubs/{id}", s.handleDeleteStub).Methods(http.MethodDelete)
	r.HandleFunc(base+"/stubs/batchDelete", s.handleBatchDelete).Methods(http.MethodPost)
	r.HandleFunc(base+"/stubs/{id}/resetTimes", s.handleResetTimes).Methods(http.MethodPost)

	// Analytics
	r.HandleFunc(base+"/analytics/stubs", s.handleListAnalytics).Methods(http.MethodGet)
	r.HandleFunc(base+"/stubs/{id}/analytics", s.handleStubAnalytics).Methods(http.MethodGet)

	// History
	r.HandleFunc(base+"/history", s.handleListHistory).Methods(http.MethodGet)
	r.HandleFunc(base+"/history/{id}", s.handleGetHistory).Methods(http.MethodGet)
	r.HandleFunc(base+"/history", s.handleDeleteHistory).Methods(http.MethodDelete)
}

func (s *Server) handleListStubs(w http.ResponseWriter, r *http.Request) {
	filter := parseFilter(r)
	sortOpt := parseSort(r)
	rangeOpt := parseRange(r)
	items, total := s.stubs.List(r.Context(), filter, sortOpt, rangeOpt)

	end := max(rangeOpt.Start+len(items)-1, rangeOpt.Start)

	setListHeaders(w, "stubs", rangeOpt.Start, end, total)
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleSearchStubs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	filter := port.StubFilter{Query: q}
	sortOpt := parseSort(r)
	rangeOpt := parseRange(r)
	items, total := s.stubs.List(r.Context(), filter, sortOpt, rangeOpt)

	end := max(rangeOpt.Start+len(items)-1, rangeOpt.Start)

	setListHeaders(w, "stubs", rangeOpt.Start, end, total)
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleGetStub(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if item, ok := s.stubs.GetByID(r.Context(), id); ok {
		writeJSON(w, http.StatusOK, item)

		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func (s *Server) handleCreateStub(w http.ResponseWriter, r *http.Request) {
	byt, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	defer func() {
		_ = r.Body.Close()
	}()

	var payloads []domain.Stub
	if err := jsondecoder.UnmarshalSlice(byt, &payloads); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if len(payloads) == 0 {
		http.Error(w, "no stubs provided", http.StatusBadRequest)

		return
	}

	// Create all stubs
	created := make([]domain.Stub, 0, len(payloads))

	for _, payload := range payloads {
		// Note: stub type will be determined later when the stub is stored
		// since it requires access to MethodRegistry
		createdStub, err := s.stubs.Create(r.Context(), payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		created = append(created, createdStub)
	}

	// Return single stub if only one was created, otherwise return array
	if len(created) == 1 {
		writeJSON(w, http.StatusCreated, created[0])
	} else {
		writeJSON(w, http.StatusCreated, created)
	}
}

func (s *Server) handleUpdateStub(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var payload domain.Stub
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// Note: stub type will be determined later when the stub is stored
	// since it requires access to MethodRegistry

	updated, err := s.stubs.Update(r.Context(), id, payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if updated.ID == "" {
		http.Error(w, "not found", http.StatusNotFound)

		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteStub(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	_ = s.stubs.Delete(r.Context(), id)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBatchDelete(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if len(ids) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"deleted": 0})

		return
	}

	_ = s.stubs.DeleteMany(r.Context(), ids)
	writeJSON(w, http.StatusOK, map[string]any{"deleted": len(ids)})
}

func (s *Server) handleResetTimes(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	it, ok := s.stubs.GetByID(r.Context(), id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)

		return
	}

	it.Times = 0
	if _, err := s.stubs.Update(r.Context(), id, it); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"id": id, "times": 0})
}

func (s *Server) handleLintStubs(w http.ResponseWriter, r *http.Request) {
	// Read body once and validate with lightweight validator
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// Try v4 first; if it looks like legacy, fall back to legacy validation
	var probe []map[string]any

	_ = json.Unmarshal(raw, &probe)
	looksV4 := false

	for _, it := range probe {
		if _, ok := it["outputs"]; ok {
			looksV4 = true

			break
		}
	}

	if looksV4 {
		if err := validator.ValidateStubV4(raw); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"valid": true, "count": len(probe)})

		return
	}

	if err := validator.ValidateLegacy(raw); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"valid": true, "count": len(probe), "format": "legacy"})
}

// --- Added handlers for analytics, history, and dry-run ---

func (s *Server) handleListAnalytics(w http.ResponseWriter, r *http.Request) {
	if s.analytics == nil {
		setListHeaders(w, "analytics", 0, 0, 0)
		writeJSON(w, http.StatusOK, []any{})

		return
	}

	items := s.analytics.ListAll(r.Context())
	setListHeaders(w, "analytics", 0, len(items), len(items))
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleStubAnalytics(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if s.analytics == nil {
		writeJSON(w, http.StatusOK, map[string]any{"stubId": id})

		return
	}

	if item, ok := s.analytics.GetByStubID(r.Context(), id); ok {
		writeJSON(w, http.StatusOK, item)

		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func (s *Server) handleListHistory(w http.ResponseWriter, r *http.Request) {
	if s.history == nil {
		setListHeaders(w, "history", 0, 0, 0)
		writeJSON(w, http.StatusOK, []any{})

		return
	}

	rangeOpt := parseRange(r)
	items, total := s.history.List(r.Context(), rangeOpt.Start, rangeOpt.End)
	setListHeaders(w, "history", rangeOpt.Start, rangeOpt.End, total)
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	if s.history == nil {
		http.Error(w, "not found", http.StatusNotFound)

		return
	}

	id := mux.Vars(r)["id"]
	if item, ok := s.history.GetByID(r.Context(), id); ok {
		writeJSON(w, http.StatusOK, item)

		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func (s *Server) handleDeleteHistory(w http.ResponseWriter, r *http.Request) {
	if s.history == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	// Delete all history (filters can be added later)
	s.history.Clear(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDryRunMatch(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Service string         `json:"service"`
		Method  string         `json:"method"`
		Headers map[string]any `json:"headers"`
		Data    map[string]any `json:"data"`
	}

	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// Preview with basic matching (no side effects); REST must not mark used
	filter := port.StubFilter{Service: body.Service, Method: body.Method}
	items, _ := s.stubs.List(
		r.Context(),
		filter,
		port.SortOption{Field: "priority", Direction: "DESC"},
		port.RangeOption{Start: 0, End: maxListEnd},
	)

	preview := make([]map[string]any, 0, len(items))
	for _, it := range items {
		if !matchHeaders(it.Headers, body.Headers) {
			continue
		}

		if !matchInputs(it.Inputs, body.Data) {
			continue
		}

		preview = append(preview, map[string]any{
			"id":       it.ID,
			"service":  it.Service,
			"method":   it.Method,
			"priority": it.Priority,
		})
	}

	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handlePlugins(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.plugins)
}

func matchHeaders(m *domain.Matcher, headers map[string]any) bool {
	if m == nil {
		return true
	}

	return matcher.Match(convertMatcher(*m), headers)
}

func matchInputs(ms []domain.Matcher, data map[string]any) bool {
	if len(ms) == 0 {
		return true
	}

	for _, m := range ms {
		if !matcher.Match(convertMatcher(m), data) {
			return false
		}
	}

	return true
}

func convertMatcher(dm domain.Matcher) matcher.Matcher {
	out := matcher.Matcher{
		Equals:           map[string]any{},
		Contains:         map[string]any{},
		Matches:          map[string]any{},
		IgnoreArrayOrder: dm.IgnoreArrayOrder,
	}
	if dm.Equals != nil {
		out.Equals = dm.Equals
	}

	if dm.Contains != nil {
		out.Contains = dm.Contains
	}

	if dm.Matches != nil {
		out.Matches = dm.Matches
	}

	if len(dm.Any) > 0 {
		out.Any = make([]matcher.Matcher, 0, len(dm.Any))
		for _, child := range dm.Any {
			out.Any = append(out.Any, convertMatcher(child))
		}
	}

	return out
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", contentJSON)
	w.WriteHeader(code)

	byt, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	// Write response body - ignore write errors since headers are already sent
	_, _ = w.Write(byt)
}

func setListHeaders(w http.ResponseWriter, resource string, start, end, total int) {
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("Content-Range", resource+" "+strconv.Itoa(start)+"-"+strconv.Itoa(end)+"/"+strconv.Itoa(total))
}

func parseFilter(r *http.Request) port.StubFilter {
	f := r.URL.Query().Get("filter")

	out := port.StubFilter{}
	if f == "" {
		return out
	}

	if err := json.Unmarshal([]byte(f), &out); err != nil {
		// Return empty filter on parse error
		return port.StubFilter{}
	}

	return out
}

func parseSort(r *http.Request) port.SortOption {
	s := r.URL.Query().Get("sort")

	opt := port.SortOption{Field: "id", Direction: "ASC"}
	if s == "" {
		return opt
	}

	var tuple []string
	if err := json.Unmarshal([]byte(s), &tuple); err == nil && len(tuple) == 2 {
		opt.Field = tuple[0]
		opt.Direction = tuple[1]
	}

	return opt
}

func parseRange(r *http.Request) port.RangeOption {
	rng := r.URL.Query().Get("range")

	// Default to full range to match legacy behavior (return all items by default)
	opt := port.RangeOption{Start: 0, End: maxListEnd}

	if rng != "" {
		var rr []int
		if err := json.Unmarshal([]byte(rng), &rr); err == nil && len(rr) == 2 {
			opt.Start = rr[0]
			opt.End = rr[1]

			return opt
		}
	}

	if v := r.URL.Query().Get("_start"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			opt.Start = n
		}
	}

	if v := r.URL.Query().Get("_end"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			opt.End = n
		}
	}

	return opt
}

func handleListServices(w http.ResponseWriter, _ *http.Request) {
	// Return empty list without protobuf registry to keep dependencies minimal in tests
	writeJSON(w, http.StatusOK, []string{})
}

func handleListMethods(w http.ResponseWriter, r *http.Request) {
	// Return empty list for now; can be enhanced with registry if needed
	// service parameter is ignored for now
	_ = r.URL.Query().Get("service")

	writeJSON(w, http.StatusOK, []string{})
}
