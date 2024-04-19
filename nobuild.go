//go:build nobuild

package main

import (
	_ "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	_ "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	_ "github.com/bavix/gripmock-sdk-go"
)
