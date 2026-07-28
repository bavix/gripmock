package grpcclient

import "google.golang.org/grpc"

const sessionHeader = "x-gripmock-session"

// UnarySessionInterceptor injects gripmock session header into unary calls.
func UnarySessionInterceptor(session string) grpc.UnaryClientInterceptor {
	return unaryHeaderInterceptor(sessionHeader, session)
}

// StreamSessionInterceptor injects gripmock session header into stream calls.
func StreamSessionInterceptor(session string) grpc.StreamClientInterceptor {
	return streamHeaderInterceptor(sessionHeader, session)
}
