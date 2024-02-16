package protogen

import (
	_ "github.com/goccy/go-json"
	_ "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	_ "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	_ "google.golang.org/grpc"
	_ "google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/status"
	_ "google.golang.org/protobuf/reflect/protoreflect"
	_ "google.golang.org/protobuf/runtime/protoimpl"
	_ "google.golang.org/protobuf/types/known/apipb"
	_ "google.golang.org/protobuf/types/known/emptypb"
)
