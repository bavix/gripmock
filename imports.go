package main

import (
	_ "github.com/gripmock/grpc-interceptors"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	_ "google.golang.org/grpc/health"

	_ "github.com/bavix/gripmock-sdk-go"
)
