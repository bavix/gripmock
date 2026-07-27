package app

import (
	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// decodeAndValidateMCPStubs decodes the "stubs" arg, applies the session, and
// validates each stub — the shared preamble of upsert and validate.
func decodeAndValidateMCPStubs(h *RestServer, args map[string]any) ([]*stuber.Stub, error) {
	rawStubs, ok := args["stubs"]
	if !ok || rawStubs == nil {
		return nil, mcpRequiredArgError("stubs")
	}

	stubs, err := decodeMCPStubsArg(rawStubs)
	if err != nil {
		return nil, err
	}

	sessionID, _ := args["session"].(string)

	for _, stub := range stubs {
		stub.Session = sessionID

		if err = h.validateStub(stub); err != nil {
			return nil, mcpInvalidArgErrorWithCause(err.Error(), err)
		}
	}

	return stubs, nil
}

func mcpStubsUpsert(h *RestServer, args map[string]any) (map[string]any, error) {
	stubs, err := decodeAndValidateMCPStubs(h, args)
	if err != nil {
		return nil, err
	}

	ids := h.budgerigar.PutMany(stubs...)

	return map[string]any{"ids": uuidListToStringSlice(ids)}, nil
}

// mcpStubsValidate dry-runs stub validation without persisting, mirroring the
// REST POST /stubs/validate endpoint. Returns the normalized stubs (nil IDs
// stripped) so an agent can preview exactly what an upsert would store.
func mcpStubsValidate(h *RestServer, args map[string]any) (map[string]any, error) {
	stubs, err := decodeAndValidateMCPStubs(h, args)
	if err != nil {
		return nil, err
	}

	normalized, err := normalizeMCPStubs(stubs)
	if err != nil {
		return nil, err
	}

	return map[string]any{"valid": true, "stubs": normalized}, nil
}

// normalizeMCPStubs round-trips stubs through JSON and drops zero-value IDs so
// the preview matches the REST validate response shape.
func normalizeMCPStubs(stubs []*stuber.Stub) ([]map[string]any, error) {
	raw, err := json.Marshal(stubs)
	if err != nil {
		return nil, mcpStubPayloadArgError(err)
	}

	var result []map[string]any
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, mcpStubPayloadArgError(err)
	}

	zeroID := uuid.Nil.String()
	for i := range result {
		if id, ok := result[i]["id"]; ok && id == zeroID {
			delete(result[i], "id")
		}
	}

	return result, nil
}

func mcpStubsList(h *RestServer, args map[string]any) (map[string]any, error) {
	return mcpStubsListResponse(h, h.budgerigar.All(), args)
}

func mcpStubsUsed(h *RestServer, args map[string]any) (map[string]any, error) {
	return mcpStubsListResponse(h, h.budgerigar.Used(), args)
}

func mcpStubsUnused(h *RestServer, args map[string]any) (map[string]any, error) {
	return mcpStubsListResponse(h, h.budgerigar.Unused(), args)
}

// mcpStubsListResponse filters/sorts/paginates, then returns the page decorated
// with the used flag plus total — the pre-pagination count, mirroring the REST
// GET /stubs X-Total-Count header so MCP clients can page large stub sets.
func mcpStubsListResponse(h *RestServer, stubs []*stuber.Stub, args map[string]any) (map[string]any, error) {
	page, total, err := listMCPStubs(stubs, args)
	if err != nil {
		return nil, err
	}

	return map[string]any{"stubs": h.decorateStubsUsed(page), "total": total}, nil
}

// decorateStubsUsed returns shallow copies carrying the response-only Used flag,
// matching the REST GET /stubs contract. Copies keep the shared storage stubs
// unmutated (Used is decoration, not persisted state).
func (h *RestServer) decorateStubsUsed(stubs []*stuber.Stub) []stuber.Stub {
	usedIDs := h.budgerigar.UsedIDs()
	out := make([]stuber.Stub, len(stubs))

	for i, s := range stubs {
		out[i] = *s
		_, out[i].Used = usedIDs[s.ID]
	}

	return out
}

func mcpStubsGet(h *RestServer, args map[string]any) (map[string]any, error) {
	id, err := mcpUUIDArg(args, "id")
	if err != nil {
		return nil, err
	}

	found := h.budgerigar.FindByID(id)

	if found == nil {
		return map[string]any{"found": false, "id": id.String()}, nil
	}

	return map[string]any{"found": true, "stub": found}, nil
}

func mcpStubsDelete(h *RestServer, args map[string]any) (map[string]any, error) {
	id, err := mcpUUIDArg(args, "id")
	if err != nil {
		return nil, err
	}

	deleted := h.budgerigar.DeleteByID(id) > 0

	return map[string]any{"deleted": deleted, "id": id.String()}, nil
}

func mcpStubsBatchDelete(h *RestServer, args map[string]any) (map[string]any, error) {
	idStrings, err := mcpStringSliceArg(args, "ids")
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(idStrings))
	deletedIDs := make([]string, 0, len(idStrings))
	notFoundIDs := make([]string, 0)

	for _, idString := range idStrings {
		id, parseErr := uuid.Parse(idString)
		if parseErr != nil {
			return nil, mcpUUIDArgError("ids", idString, parseErr)
		}

		ids = append(ids, id)

		if h.budgerigar.FindByID(id) == nil {
			notFoundIDs = append(notFoundIDs, idString)
		} else {
			deletedIDs = append(deletedIDs, idString)
		}
	}

	if len(ids) > 0 {
		h.budgerigar.DeleteByID(ids...)
	}

	return map[string]any{
		"deletedIds":  deletedIDs,
		"notFoundIds": notFoundIDs,
	}, nil
}

func mcpStubsPurge(h *RestServer, args map[string]any) (map[string]any, error) {
	sessionID, _ := args["session"].(string)
	if sessionID != "" {
		deletedCount := h.budgerigar.DeleteSession(sessionID)

		return map[string]any{"deletedCount": deletedCount, "session": sessionID}, nil
	}

	deletedCount := len(h.budgerigar.All())
	h.budgerigar.Clear()

	return map[string]any{"deletedCount": deletedCount}, nil
}

func mcpStubsSearch(h *RestServer, args map[string]any) (map[string]any, error) {
	service, _ := args["service"].(string)
	if service == "" {
		return nil, mcpRequiredArgError("service")
	}

	method, _ := args["method"].(string)
	if method == "" {
		return nil, mcpRequiredArgError("method")
	}

	input, err := mcpSearchInput(args)
	if err != nil {
		return nil, err
	}

	headers, err := mcpHeadersArg(args)
	if err != nil {
		return nil, err
	}

	sessionID, _ := args["session"].(string)

	result, searchErr := h.budgerigar.FindByQuery(stuber.Query{
		Service: service,
		Method:  method,
		Session: sessionID,
		Headers: headers,
		Input:   input,
	})
	if searchErr != nil {
		return mcpSearchNotMatchedResponse(searchErr), nil
	}

	found := result.Found()
	if found == nil {
		response := map[string]any{"matched": false}

		if similar := result.Similar(); similar != nil {
			response["similarStubId"] = similar.ID.String()
		}

		return response, nil
	}

	return map[string]any{
		"matched": true,
		"stubId":  found.ID.String(),
		"output":  found.Output,
	}, nil
}

func mcpStubsInspect(h *RestServer, args map[string]any) (map[string]any, error) {
	query, err := mcpInspectQuery(args)
	if err != nil {
		return nil, err
	}

	report := h.budgerigar.InspectQuery(query)

	return map[string]any{"report": toRestInspectReport(report)}, nil
}

func mcpInspectQuery(args map[string]any) (stuber.Query, error) {
	service, _ := args["service"].(string)
	if service == "" {
		return stuber.Query{}, mcpRequiredArgError("service")
	}

	method, _ := args["method"].(string)
	if method == "" {
		return stuber.Query{}, mcpRequiredArgError("method")
	}

	query := stuber.Query{Service: service, Method: method}

	err := mcpInspectQueryOptions(args, &query)
	if err != nil {
		return stuber.Query{}, err
	}

	return query, nil
}

func mcpInspectQueryOptions(args map[string]any, query *stuber.Query) error {
	if query == nil {
		return nil
	}

	if idValue, ok := args["id"]; ok && idValue != nil {
		id, err := mcpUUIDArg(args, "id")
		if err != nil {
			return err
		}

		query.ID = &id
	}

	if sessionID, _ := args["session"].(string); sessionID != "" {
		query.Session = sessionID
	}

	headers, err := mcpHeadersArg(args)
	if err != nil {
		return err
	}

	query.Headers = headers

	if rawInput, ok := args["input"]; ok && rawInput != nil {
		input, err := parseMCPInputArg(rawInput)
		if err != nil {
			return err
		}

		query.Input = input
	}

	return nil
}

func decodeMCPStubsArg(raw any) ([]*stuber.Stub, error) {
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, mcpStubPayloadArgError(err)
	}

	var items []*stuber.Stub
	if err = jsondecoder.UnmarshalSlice(payload, &items); err != nil {
		return nil, mcpStubPayloadArgError(err)
	}

	if len(items) == 0 {
		return nil, mcpInvalidArgError("stubs cannot be empty")
	}

	return items, nil
}

func stringArg(args map[string]any, key string) string {
	v, _ := args[key].(string)

	return v
}

// listMCPStubs filters, sorts and paginates the given stub set. It returns the
// requested page plus the total count before pagination.
func listMCPStubs(stubs []*stuber.Stub, args map[string]any) ([]*stuber.Stub, int, error) {
	filter := mcpStubListFilter{
		service: stringArg(args, "service"),
		method:  stringArg(args, "method"),
		session: stringArg(args, "session"),
		source:  stringArg(args, "source"),
		query:   stringArg(args, "q"),
	}

	limit, err := mcpIntArg(args, "limit", 0)
	if err != nil {
		return nil, 0, err
	}

	offset, err := mcpIntArg(args, "offset", 0)
	if err != nil {
		return nil, 0, err
	}

	filtered := filterMCPStubs(stubs, filter)

	// Sort before paginating so offset/limit page a stable order, matching
	// the REST GET /stubs contract. Unknown sort modes fall back to priority_desc.
	stuber.SortStubs(filtered, stringArg(args, "sort"))

	total := len(filtered)

	if offset >= len(filtered) {
		return []*stuber.Stub{}, total, nil
	}

	filtered = filtered[offset:]

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, total, nil
}

func mcpUUIDArg(args map[string]any, key string) (uuid.UUID, error) {
	value, _ := args[key].(string)
	if value == "" {
		return uuid.Nil, mcpRequiredArgError(key)
	}

	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, mcpUUIDArgError(key, value, err)
	}

	return id, nil
}

func mcpStringSliceArg(args map[string]any, key string) ([]string, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil, mcpRequiredArgError(key)
	}

	switch values := raw.(type) {
	case []string:
		return validateMCPStringSlice(values, key)
	case []any:
		return convertMCPAnyStringSlice(values, key)
	default:
		return nil, mcpStringListArgError(key)
	}
}

func mcpSearchInput(args map[string]any) ([]map[string]any, error) {
	if rawInput, ok := args["input"]; ok && rawInput != nil {
		return parseMCPInputArg(rawInput)
	}

	payload, ok := args["payload"].(map[string]any)
	if !ok || payload == nil {
		return nil, mcpRequiredArgError("payload")
	}

	return []map[string]any{payload}, nil
}

func mcpHeadersArg(args map[string]any) (map[string]any, error) {
	rawHeaders, ok := args["headers"]
	if !ok || rawHeaders == nil {
		return map[string]any{}, nil
	}

	headers, ok := rawHeaders.(map[string]any)
	if !ok {
		return nil, mcpInvalidArgError("headers must be an object")
	}

	return headers, nil
}

func mcpSearchNotMatchedResponse(searchErr error) map[string]any {
	return map[string]any{"matched": false, "error": searchErr.Error()}
}
