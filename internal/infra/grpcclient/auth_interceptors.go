package grpcclient

import "google.golang.org/grpc"

func bearerValue(token string) string {
	if token == "" {
		return ""
	}

	return "Bearer " + token
}

// UnaryBearerInterceptor injects Authorization bearer token for unary client calls.
func UnaryBearerInterceptor(token string) grpc.UnaryClientInterceptor {
	return unaryHeaderInterceptor("authorization", bearerValue(token))
}

// StreamBearerInterceptor injects Authorization bearer token for stream client calls.
func StreamBearerInterceptor(token string) grpc.StreamClientInterceptor {
	return streamHeaderInterceptor("authorization", bearerValue(token))
}
