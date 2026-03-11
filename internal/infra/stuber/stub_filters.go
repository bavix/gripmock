package stuber

func filterByServiceMethod(stubs []*Stub, service, method string) []*Stub {
	filtered := make([]*Stub, 0, len(stubs))
	for _, stub := range stubs {
		if stub.Service == service && stub.Method == method {
			filtered = append(filtered, stub)
		}
	}

	return filtered
}
