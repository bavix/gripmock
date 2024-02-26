package main

import (
	_ "github.com/bavix/gripmock-sdk-go"
	_ "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	_ "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)
