package app

import (
	"net/http"
	"strconv"

	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func (h *RestServer) AddStub(w http.ResponseWriter, r *http.Request) {
	byt, err := httputil.RequestBody(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	var inputs []*stuber.Stub

	if err := jsondecoder.UnmarshalSlice(byt, &inputs); err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	sess := muxmiddleware.FromRequest(r)
	for _, stub := range inputs {
		stub.Session = sess
		stub.Source = stuber.SourceRest

		if err := h.validateStub(stub); err != nil {
			h.validationError(r.Context(), w, err)

			return
		}
	}

	h.writeResponse(r.Context(), w, h.budgerigar.PutMany(inputs...))
}

// ListDescriptors returns service IDs of descriptors added via POST /descriptors.

func (h *RestServer) DeleteStubByID(w http.ResponseWriter, _ *http.Request, uuid rest.ID) {
	h.budgerigar.DeleteByID(uuid)

	w.WriteHeader(http.StatusNoContent)
}

// BatchStubsDelete removes multiple stubs by ID.
func (h *RestServer) BatchStubsDelete(w http.ResponseWriter, r *http.Request) {
	byt, err := httputil.RequestBody(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	var inputs []uuid.UUID

	if err := jsondecoder.UnmarshalSlice(byt, &inputs); err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	if len(inputs) > 0 {
		h.budgerigar.DeleteByID(inputs...)
	}
}

// ListUsedStubs returns stubs that have been matched.
func (h *RestServer) ListUsedStubs(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, h.budgerigar.Used())
}

// ListUnusedStubs returns stubs that have never been matched.
func (h *RestServer) ListUnusedStubs(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, h.budgerigar.Unused())
}

// ValidateStub dry-runs validation over the posted stubs without persisting them.
func (h *RestServer) ValidateStub(w http.ResponseWriter, r *http.Request) {
	byt, err := httputil.RequestBody(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	var inputs []*stuber.Stub

	if err := jsondecoder.UnmarshalSlice(byt, &inputs); err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	for _, stub := range inputs {
		if err := h.validateStub(stub); err != nil {
			h.validationError(r.Context(), w, err)

			return
		}
	}

	raw, err := json.Marshal(inputs)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	var result []map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	zeroID := uuid.Nil.String()
	for i := range result {
		if id, ok := result[i]["id"]; ok && id == zeroID {
			delete(result[i], "id")
		}
	}

	h.writeResponse(r.Context(), w, result)
}

// ListStubs returns all stubs, optionally filtered by source.
func (h *RestServer) ListStubs(w http.ResponseWriter, r *http.Request, params rest.ListStubsParams) {
	stubs, total := h.budgerigar.List(listOptionsFromParams(params))
	w.Header().Set("X-Total-Count", strconv.Itoa(total))

	// Decorate shallow copies with the used flag — storage stays untouched.
	usedIDs := h.budgerigar.UsedIDs()
	out := make([]stuber.Stub, len(stubs))

	for i, s := range stubs {
		out[i] = *s
		_, out[i].Used = usedIDs[s.ID]
	}

	h.writeResponse(r.Context(), w, out)
}

func listOptionsFromParams(params rest.ListStubsParams) stuber.ListOptions {
	options := stuber.ListOptions{
		Source:  stringFromPtr(params.Source),
		Service: stringFromPtr(params.Service),
		Method:  stringFromPtr(params.Method),
		Query:   stringFromPtr(params.Q),
		Sort:    stringFromPtr(params.Sort),
		Limit:   intFromPtr(params.Limit),
		Offset:  intFromPtr(params.Offset),
	}

	if params.Session != nil {
		options.Session = *params.Session
		options.SessionSet = true
	}

	return options
}

// PurgeStubs removes all stubs.
func (h *RestServer) PurgeStubs(w http.ResponseWriter, _ *http.Request) {
	h.budgerigar.Clear()

	w.WriteHeader(http.StatusNoContent)
}

// SearchStubs finds a stub matching the query.
func (h *RestServer) SearchStubs(w http.ResponseWriter, r *http.Request) {
	query, err := stuber.NewQuery(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	defer func() {
		_ = r.Body.Close()
	}()

	if sess := muxmiddleware.FromRequest(r); sess != "" {
		query.Session = sess
	}

	result, err := h.budgerigar.FindByQuery(query)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(r.Context(), w, err)

		return
	}

	if result.Found() == nil {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(r.Context(), w, h.errorFormatter.FormatStubNotFoundError(query, result))

		return
	}

	h.writeResponse(r.Context(), w, result.Found().Output)
}

// InspectStubs returns detailed matching report for a query.
func (h *RestServer) InspectStubs(w http.ResponseWriter, r *http.Request) {
	var req rest.InspectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	query := stuber.Query{
		Service: req.Service,
		Method:  req.Method,
		Input:   req.Input,
		Headers: req.Headers,
	}

	if req.Id != nil {
		id := *req.Id
		query.ID = &id
	}

	if req.Session != nil {
		query.Session = *req.Session
	}

	report := h.budgerigar.InspectQuery(query)
	h.writeResponse(r.Context(), w, toRestInspectReport(report))
}

func toRestInspectReport(report stuber.InspectReport) rest.InspectReport {
	stages := make([]rest.InspectStage, len(report.Stages))
	for i, stage := range report.Stages {
		stages[i] = rest.InspectStage{
			Name:    stage.Name,
			Before:  stage.Before,
			After:   stage.After,
			Removed: stage.Removed,
		}
	}

	candidates := make([]rest.InspectCandidate, len(report.Candidates))
	for i, candidate := range report.Candidates {
		events := make([]rest.InspectCandidateEvent, len(candidate.Events))
		for j, event := range candidate.Events {
			reason := event.Reason
			events[j] = rest.InspectCandidateEvent{
				Stage:  event.Stage,
				Result: event.Result,
				Reason: nilIfEmpty(reason),
			}
		}

		candidates[i] = rest.InspectCandidate{
			Id:               candidate.ID.String(),
			Service:          candidate.Service,
			Method:           candidate.Method,
			Session:          candidate.Session,
			Priority:         candidate.Priority,
			Times:            candidate.Times,
			Used:             candidate.Used,
			Specificity:      candidate.Specificity,
			Score:            candidate.Score,
			VisibleBySession: candidate.VisibleBySession,
			WithinTimes:      candidate.WithinTimes,
			HeadersMatched:   candidate.HeadersMatched,
			InputMatched:     candidate.InputMatched,
			Matched:          candidate.Matched,
			ExcludedBy:       candidate.ExcludedBy,
			Events:           events,
		}
	}

	return rest.InspectReport{
		Service:          report.Service,
		Method:           report.Method,
		Session:          report.Session,
		MatchedStubId:    stringFromUUIDPtr(report.MatchedStubID),
		SimilarStubId:    stringFromUUIDPtr(report.SimilarStubID),
		FallbackToMethod: report.FallbackToMethod,
		Error:            stringFromPtr(report.Error),
		Stages:           stages,
		Candidates:       candidates,
	}
}
