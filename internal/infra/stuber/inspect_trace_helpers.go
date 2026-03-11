package stuber

func boolResult(ok bool) string {
	if ok {
		return traceResultPassed
	}

	return traceResultFailed
}

func reasonIf(condition bool, reason string) string {
	if condition {
		return reason
	}

	return ""
}

func isStubVisibleForSession(stubSession, querySession string) bool {
	if querySession == "" {
		return stubSession == ""
	}

	return stubSession == "" || stubSession == querySession
}

func doesQueryMatchStubHeaders(query Query, stub *Stub) bool {
	if stub.Headers.Len() > 0 && len(query.Headers) == 0 {
		return false
	}

	if len(query.Headers) > 0 && !matchHeaders(query.Headers, stub.Headers) {
		return false
	}

	return true
}
