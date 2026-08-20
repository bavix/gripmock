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

// VisibleForSession reports whether a stub is visible to a caller in querySession:
// global stubs are visible to everyone, session stubs only to their own session.
func VisibleForSession(stubSession, querySession string) bool {
	if querySession == "" {
		return stubSession == ""
	}

	return stubSession == "" || stubSession == querySession
}
