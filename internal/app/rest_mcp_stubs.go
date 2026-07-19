package app

import (
	"github.com/goccy/go-json"
	"github.com/google/uuid"

	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func mcpStubsUpsert(h *RestServer, args map[string]any) (map[string]any, error) {
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

	ids := h.budgerigar.PutMany(stubs...)

	return map[string]any{"ids": uuidListToStringSlice(ids)}, nil
}

func mcpStubsList(h *RestServer, args map[string]any) (map[string]any, error) {
	stubs, err := listMCPStubs(h.budgerigar.All(), args)
	if err != nil {
		return nil, err
	}

	return map[string]any{"stubs": stubs}, nil
}

func mcpStubsUsed(h *RestServer, args map[string]any) (map[string]any, error) {
	stubs, err := listMCPStubs(h.budgerigar.Used(), args)
	if err != nil {
		return nil, err
	}

	return map[string]any{"stubs": stubs}, nil
}

func mcpStubsUnused(h *RestServer, args map[string]any) (map[string]any, error) {
	stubs, err := listMCPStubs(h.budgerigar.Unused(), args)
	if err != nil {
		return nil, err
	}

	return map[string]any{"stubs": stubs}, nil
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

func listMCPStubs(stubs []*stuber.Stub, args map[string]any) ([]*stuber.Stub, error) {
	service, _ := args["service"].(string)
	method, _ := args["method"].(string)
	sessionID, _ := args["session"].(string)

	limit, err := mcpIntArg(args, "limit", 0)
	if err != nil {
		return nil, err
	}

	offset, err := mcpIntArg(args, "offset", 0)
	if err != nil {
		return nil, err
	}

	filtered := filterMCPStubs(stubs, service, method, sessionID)

	if offset >= len(filtered) {
		return []*stuber.Stub{}, nil
	}

	filtered = filtered[offset:]

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
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
