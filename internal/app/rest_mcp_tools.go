package app

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	mcpusecase "github.com/bavix/gripmock/v3/internal/app/usecase/mcp"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
)

func mcpSchemaStub(_ *RestServer, _ map[string]any) (map[string]any, error) {
	return map[string]any{"schemaUrl": stubSchemaURL}, nil
}

func mcpHealthLiveness(_ *RestServer, _ map[string]any) (map[string]any, error) {
	return map[string]any{"message": "ok", "time": time.Now()}, nil
}

func mcpHealthReadiness(h *RestServer, _ map[string]any) (map[string]any, error) {
	ready := h.ok.Load()
	if !ready {
		return map[string]any{"ready": false, "message": "not ready", "time": time.Now()}, nil
	}

	return map[string]any{"ready": true, "message": "ok", "time": time.Now()}, nil
}

func mcpHealthStatus(h *RestServer, _ map[string]any) (map[string]any, error) {
	ready := h.ok.Load()

	readiness := "ok"
	if !ready {
		readiness = "not ready"
	}

	return map[string]any{
		"liveness":  "ok",
		"readiness": readiness,
		"ready":     ready,
		"time":      time.Now(),
	}, nil
}

func mcpDashboard(h *RestServer, args map[string]any) (map[string]any, error) {
	return map[string]any{"dashboard": h.dashboardPayload(mcpSessionRequest(args))}, nil
}

func mcpDashboardOverview(h *RestServer, args map[string]any) (map[string]any, error) {
	payload := h.dashboardPayload(mcpSessionRequest(args))

	return map[string]any{"overview": rest.DashboardOverview{
		TotalServices:      payload.TotalServices,
		TotalStubs:         payload.TotalStubs,
		UsedStubs:          payload.UsedStubs,
		UnusedStubs:        payload.UnusedStubs,
		TotalSessions:      payload.TotalSessions,
		RuntimeDescriptors: payload.RuntimeDescriptors,
		TotalHistory:       payload.TotalHistory,
		HistoryErrors:      payload.HistoryErrors,
	}}, nil
}

func mcpDashboardInfo(h *RestServer, args map[string]any) (map[string]any, error) {
	payload := h.dashboardPayload(mcpSessionRequest(args))

	return map[string]any{"info": rest.DashboardInfo{
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
	}}, nil
}

func mcpSessionsList(h *RestServer, _ map[string]any) (map[string]any, error) {
	return map[string]any{"sessions": h.budgerigar.Sessions()}, nil
}

func mcpGripmockInfo(h *RestServer, _ map[string]any) (map[string]any, error) {
	overview := h.dashboardPayload(nil)

	return map[string]any{
		"appName":            overview.AppName,
		"version":            overview.Version,
		"protocolVersion":    mcpusecase.ProtocolVersion,
		"historyEnabled":     overview.HistoryEnabled,
		"ready":              overview.Ready,
		"totalServices":      overview.TotalServices,
		"totalStubs":         overview.TotalStubs,
		"totalSessions":      overview.TotalSessions,
		"runtimeDescriptors": overview.RuntimeDescriptors,
		"tools":              mcpusecase.ListRuntimeTools(),
	}, nil
}

func mcpReflectInfo(h *RestServer, _ map[string]any) (map[string]any, error) {
	runtimePaths, reflectionPrefixes, reflectionFiles := runtimeDescriptorStats(h)

	globalCount := 0

	protoregistry.GlobalFiles.RangeFiles(func(_ protoreflect.FileDescriptor) bool {
		globalCount++

		return true
	})

	return map[string]any{
		"runtimeDescriptorFiles":    len(runtimePaths),
		"reflectionDescriptorFiles": reflectionFiles,
		"dynamicDescriptorFiles":    len(runtimePaths) - reflectionFiles,
		"reflectionDetected":        reflectionFiles > 0,
		"reflectionSources":         reflectionPrefixes,
		"globalDescriptorFiles":     globalCount,
	}, nil
}

func mcpReflectSources(h *RestServer, args map[string]any) (map[string]any, error) {
	runtimePaths, reflectionPrefixes, _ := runtimeDescriptorStats(h)
	reflectionPaths, dynamicPaths, _ := splitRuntimeDescriptorPaths(runtimePaths)

	kind, _ := args["kind"].(string)
	if kind == "" {
		kind = "all"
	}

	if kind != "all" && kind != "reflection" && kind != "dynamic" {
		return nil, mcpInvalidArgError("kind must be one of: all, reflection, dynamic")
	}

	offset, err := mcpIntArg(args, "offset", 0)
	if err != nil {
		return nil, err
	}

	limit, err := mcpIntArg(args, "limit", 0)
	if err != nil {
		return nil, err
	}

	filtered := runtimePaths

	switch kind {
	case "reflection":
		filtered = reflectionPaths
	case "dynamic":
		filtered = dynamicPaths
	}

	total := len(filtered)
	filtered = paginateStringSlice(filtered, offset, limit)

	return map[string]any{
		"kind":   kind,
		"paths":  filtered,
		"total":  total,
		"offset": offset,
		"limit":  limit,
		"groups": map[string]any{
			"reflection": map[string]any{"count": len(reflectionPaths)},
			"dynamic":    map[string]any{"count": len(dynamicPaths)},
		},
		"reflectionSources": reflectionPrefixes,
	}, nil
}

func runtimeDescriptorStats(h *RestServer) ([]string, []string, int) {
	runtimePaths := h.restDescriptors.Paths()
	reflectionPaths, _, prefixes := splitRuntimeDescriptorPaths(runtimePaths)

	return runtimePaths, prefixes, len(reflectionPaths)
}

func splitRuntimeDescriptorPaths(runtimePaths []string) ([]string, []string, []string) {
	reflectionPrefixes := make(map[string]struct{})
	reflectionPaths := make([]string, 0, len(runtimePaths))
	dynamicPaths := make([]string, 0, len(runtimePaths))

	for _, path := range runtimePaths {
		if !strings.HasPrefix(path, "grpc_reflect_") {
			dynamicPaths = append(dynamicPaths, path)

			continue
		}

		reflectionPaths = append(reflectionPaths, path)

		prefix := path
		if idx := strings.Index(prefix, "/"); idx > 0 {
			prefix = prefix[:idx]
		}

		reflectionPrefixes[prefix] = struct{}{}
	}

	prefixes := make([]string, 0, len(reflectionPrefixes))
	for prefix := range reflectionPrefixes {
		prefixes = append(prefixes, prefix)
	}

	sort.Strings(prefixes)

	return reflectionPaths, dynamicPaths, prefixes
}

func paginateStringSlice(items []string, offset int, limit int) []string {
	if offset >= len(items) {
		return []string{}
	}

	items = items[offset:]

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items
}

func mcpSessionRequest(args map[string]any) *http.Request {
	req := &http.Request{Header: make(http.Header)}
	if sessionID, _ := args["session"].(string); sessionID != "" {
		req.Header.Set(muxmiddleware.HeaderName, sessionID)
	}

	return req
}

func mcpDescriptorsAdd(h *RestServer, args map[string]any) (map[string]any, error) {
	descriptorSetBase64, _ := args["descriptorSetBase64"].(string)
	if descriptorSetBase64 == "" {
		return nil, mcpRequiredArgError("descriptorSetBase64")
	}

	payload, err := base64.StdEncoding.DecodeString(descriptorSetBase64)
	if err != nil {
		return nil, mcpDescriptorSetBase64ArgError(err)
	}

	serviceIDs, err := registerDescriptorBytes(h, payload)
	if err != nil {
		return nil, mcpDescriptorRegistrationArgError(err)
	}

	return map[string]any{"serviceIDs": serviceIDs}, nil
}

func mcpDescriptorsList(h *RestServer, _ map[string]any) (map[string]any, error) {
	return map[string]any{"serviceIDs": h.restDescriptors.ServiceIDs()}, nil
}

func mcpServicesList(h *RestServer, _ map[string]any) (map[string]any, error) {
	return map[string]any{"services": h.collectAllServices()}, nil
}

func mcpServicesDelete(h *RestServer, args map[string]any) (map[string]any, error) {
	serviceID, _ := args["serviceID"].(string)
	if serviceID == "" {
		return nil, mcpRequiredArgError("serviceID")
	}

	removed := unregisterService(h, serviceID)

	return map[string]any{"removed": removed > 0, "serviceID": serviceID}, nil
}

func mcpServicesGet(h *RestServer, args map[string]any) (map[string]any, error) {
	serviceID, _ := args["serviceID"].(string)
	if serviceID == "" {
		return nil, mcpRequiredArgError("serviceID")
	}

	service, ok := h.findServiceDetailed(serviceID)
	if !ok {
		return nil, mcpInvalidArgError(errServiceNotFound.Error() + ": " + serviceID)
	}

	return map[string]any{"service": service}, nil
}

func mcpServicesMethods(h *RestServer, args map[string]any) (map[string]any, error) {
	serviceID, _ := args["serviceID"].(string)
	if serviceID == "" {
		return nil, mcpRequiredArgError("serviceID")
	}

	serviceDescriptor, ok := h.findServiceDescriptor(serviceID)
	if !ok {
		return nil, mcpInvalidArgError(errServiceNotFound.Error() + ": " + serviceID)
	}

	return map[string]any{"methods": h.serviceFromDescriptor(serviceDescriptor, false).Methods}, nil
}

func mcpServicesMethod(h *RestServer, args map[string]any) (map[string]any, error) {
	serviceID, _ := args["serviceID"].(string)
	if serviceID == "" {
		return nil, mcpRequiredArgError("serviceID")
	}

	methodID, _ := args["methodID"].(string)
	if methodID == "" {
		return nil, mcpRequiredArgError("methodID")
	}

	service, ok := h.findServiceDetailed(serviceID)
	if !ok {
		return nil, mcpInvalidArgError(errServiceNotFound.Error() + ": " + serviceID)
	}

	for _, method := range service.Methods {
		if method.Id == methodID || method.Name == methodID {
			return map[string]any{"method": method}, nil
		}
	}

	return nil, mcpInvalidArgError(errMethodNotFound.Error() + " " + methodID + " in service " + serviceID)
}

func mcpHistoryList(h *RestServer, args map[string]any) (map[string]any, error) {
	service, _ := args["service"].(string)
	method, _ := args["method"].(string)
	session, _ := args["session"].(string)

	limit, err := mcpIntArg(args, "limit", 0)
	if err != nil {
		return nil, err
	}

	records := filterHistory(h, history.FilterOpts{
		Service: service,
		Method:  method,
		Session: session,
	}, limit)

	return map[string]any{"records": records}, nil
}

func mcpHistoryErrors(h *RestServer, args map[string]any) (map[string]any, error) {
	session, _ := args["session"].(string)

	limit, err := mcpIntArg(args, "limit", 0)
	if err != nil {
		return nil, err
	}

	errorsOnly := extractErrorRecords(filterHistory(h, history.FilterOpts{Session: session}, 0))
	if limit > 0 && len(errorsOnly) > limit {
		errorsOnly = errorsOnly[len(errorsOnly)-limit:]
	}

	return map[string]any{"records": errorsOnly}, nil
}

func mcpVerifyCalls(h *RestServer, args map[string]any) (map[string]any, error) {
	service, _ := args["service"].(string)
	if service == "" {
		return nil, mcpRequiredArgError("service")
	}

	method, _ := args["method"].(string)
	if method == "" {
		return nil, mcpRequiredArgError("method")
	}

	expectedCount, err := mcpIntArg(args, "expectedCount", -1)
	if err != nil {
		return nil, err
	}

	if expectedCount < 0 {
		return nil, mcpRequiredArgError("expectedCount")
	}

	if h.history == nil {
		return map[string]any{"verified": false, "message": "history is disabled", "expected": expectedCount, "actual": 0}, nil
	}

	session, _ := args["session"].(string)
	calls := h.history.Filter(history.FilterOpts{Service: service, Method: method, Session: session})
	actual := len(calls)

	if actual != expectedCount {
		return map[string]any{
			"verified": false,
			"message":  fmt.Sprintf("expected %s/%s to be called %d times, got %d", service, method, expectedCount, actual),
			"expected": expectedCount,
			"actual":   actual,
		}, nil
	}

	return map[string]any{"verified": true, "message": "ok", "expected": expectedCount, "actual": actual}, nil
}

func mcpDebugCall(h *RestServer, args map[string]any) (map[string]any, error) {
	service, _ := args["service"].(string)
	if service == "" {
		return nil, mcpRequiredArgError("service")
	}

	method, _ := args["method"].(string)
	session, _ := args["session"].(string)

	limit, err := mcpIntArg(args, "limit", debugCallDefaultLimit)
	if err != nil {
		return nil, err
	}

	stubsLimit, err := mcpIntArg(args, "stubsLimit", debugCallDefaultLimit)
	if err != nil {
		return nil, err
	}

	return debugCall(h, service, method, session, limit, stubsLimit), nil
}
