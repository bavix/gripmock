package app

import (
	"slices"

	"github.com/google/uuid"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func filterMCPStubs(stubs []*stuber.Stub, service, method, sessionID string) []*stuber.Stub {
	filtered := make([]*stuber.Stub, 0, len(stubs))

	for _, stub := range stubs {
		if !mcpStubMatchesFilters(stub, service, method, sessionID) {
			continue
		}

		filtered = append(filtered, stub)
	}

	return filtered
}

func mcpStubMatchesFilters(stub *stuber.Stub, service, method, sessionID string) bool {
	if service != "" && stub.Service != service {
		return false
	}

	if method != "" && stub.Method != method {
		return false
	}

	return stubVisibleForSession(stub.Session, sessionID)
}

func validateMCPStringSlice(values []string, key string) ([]string, error) {
	if len(values) == 0 {
		return nil, mcpInvalidArgError(key + " cannot be empty")
	}

	if slices.Contains(values, "") {
		return nil, mcpStringListArgError(key)
	}

	return values, nil
}

func convertMCPAnyStringSlice(values []any, key string) ([]string, error) {
	if len(values) == 0 {
		return nil, mcpInvalidArgError(key + " cannot be empty")
	}

	out := make([]string, 0, len(values))
	for _, item := range values {
		value, ok := item.(string)
		if !ok || value == "" {
			return nil, mcpStringListArgError(key)
		}

		out = append(out, value)
	}

	return out, nil
}

func parseMCPInputArg(rawInput any) ([]map[string]any, error) {
	switch input := rawInput.(type) {
	case []map[string]any:
		if len(input) == 0 {
			return nil, mcpInvalidArgError("input cannot be empty")
		}

		return input, nil
	case []any:
		if len(input) == 0 {
			return nil, mcpInvalidArgError("input cannot be empty")
		}

		return convertMCPAnyMapSlice(input)
	default:
		return nil, mcpInvalidArgError("input must be an array")
	}
}

func convertMCPAnyMapSlice(input []any) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(input))
	for _, item := range input {
		message, ok := item.(map[string]any)
		if !ok {
			return nil, mcpInvalidArgError("input must contain JSON objects")
		}

		out = append(out, message)
	}

	return out, nil
}

func uuidListToStringSlice(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))

	for _, id := range ids {
		out = append(out, id.String())
	}

	return out
}

func debugCall(h *RestServer, service, method, session string, historyLimit, stubsLimit int) map[string]any {
	serviceFound, methodFound := lookupServiceAndMethod(h, service, method)
	dynamic := slices.Contains(h.restDescriptors.ServiceIDs(), service)
	stubCount, stubRecords := collectDebugStubs(h, service, method, session, stubsLimit)
	historyRecords := filterHistory(h, history.FilterOpts{Service: service, Method: method, Session: session}, historyLimit)
	errorRecords := extractErrorRecords(historyRecords)
	hints := buildDebugHints(h, serviceFound, methodFound, method, stubCount)

	return map[string]any{
		"service":           service,
		"method":            method,
		"session":           session,
		"serviceRegistered": serviceFound,
		"methodRegistered":  methodFound,
		"dynamicService":    dynamic,
		"stubCount":         stubCount,
		"stubs":             stubRecords,
		"historyCount":      len(historyRecords),
		"errorCount":        len(errorRecords),
		"recentHistory":     historyRecords,
		"recentErrors":      errorRecords,
		"hints":             hints,
	}
}

func lookupServiceAndMethod(h *RestServer, service, method string) (bool, bool) {
	for _, svc := range h.collectAllServices() {
		if svc.Id != service {
			continue
		}

		if method == "" {
			return true, true
		}

		for _, m := range svc.Methods {
			if m.Name == method || m.Id == service+"/"+method {
				return true, true
			}
		}

		return true, false
	}

	return false, false
}

func collectDebugStubs(h *RestServer, service, method, session string, stubsLimit int) (int, []map[string]any) {
	stubRecords := make([]map[string]any, 0)
	stubCount := 0

	for _, stub := range h.budgerigar.All() {
		if stub.Service != service {
			continue
		}

		if !stubVisibleForSession(stub.Session, session) {
			continue
		}

		if method != "" && stub.Method != method {
			continue
		}

		stubCount++

		if stubsLimit > 0 && len(stubRecords) >= stubsLimit {
			continue
		}

		stubRecords = append(stubRecords, map[string]any{
			"id":       stub.ID.String(),
			"service":  stub.Service,
			"method":   stub.Method,
			"session":  stub.Session,
			"priority": stub.Priority,
		})
	}

	return stubCount, stubRecords
}

func stubVisibleForSession(stubSession, querySession string) bool {
	if querySession == "" {
		return stubSession == ""
	}

	return stubSession == "" || stubSession == querySession
}

func extractErrorRecords(records []rest.CallRecord) []rest.CallRecord {
	errorsOnly := make([]rest.CallRecord, 0)

	for _, item := range records {
		if item.Error != nil && *item.Error != "" {
			errorsOnly = append(errorsOnly, item)
		}
	}

	return errorsOnly
}

func buildDebugHints(h *RestServer, serviceFound, methodFound bool, method string, stubCount int) []string {
	hints := make([]string, 0, debugCallHintsCap)

	if !serviceFound {
		hints = append(hints, "Service is not registered. Add descriptors first (MCP descriptors.add).")
	}

	if serviceFound && method != "" && !methodFound {
		hints = append(hints, "Method is not found in service descriptor.")
	}

	if serviceFound && methodFound && stubCount == 0 {
		hints = append(hints, "No stubs found for this service/method. Add one via /api/stubs.")
	}

	if h.history == nil {
		hints = append(hints, "History is disabled; enable HISTORY_ENABLED=true to inspect call traces.")
	}

	return hints
}

func filterHistory(h *RestServer, opts history.FilterOpts, limit int) []rest.CallRecord {
	if h.history == nil {
		return []rest.CallRecord{}
	}

	calls := h.history.Filter(opts)
	if limit > 0 && len(calls) > limit {
		calls = calls[len(calls)-limit:]
	}

	out := make([]rest.CallRecord, len(calls))
	for i, c := range calls {
		out[i] = historyCallRecordToRest(c)
	}

	return out
}

func mcpIntArg(args map[string]any, key string, defaultValue int) (int, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return defaultValue, nil
	}

	switch v := raw.(type) {
	case float64:
		if v < 0 || v != float64(int(v)) {
			return 0, mcpNonNegativeIntegerArgError(key)
		}

		return int(v), nil
	case int:
		if v < 0 {
			return 0, mcpNonNegativeIntegerArgError(key)
		}

		return v, nil
	default:
		return 0, mcpNonNegativeIntegerArgError(key)
	}
}
