module github.com/bavix/gripmock

go 1.22.1

require (
	github.com/bavix/features v1.0.0
	github.com/bavix/gripmock-sdk-go v1.0.4
	github.com/bavix/gripmock-ui v1.0.0-alpha1
	github.com/bavix/gripmock/protogen v0.0.0
	github.com/goccy/go-yaml v1.11.3
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gripmock/environment v1.0.1
	github.com/gripmock/shutdown v1.0.0
	github.com/gripmock/stuber v1.0.0
	github.com/jhump/protoreflect v1.16.0
	github.com/oapi-codegen/runtime v1.1.1
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.32.0
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.51.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.51.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.50.0
	go.opentelemetry.io/otel v1.26.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.25.0
	go.opentelemetry.io/otel/sdk v1.26.0
	golang.org/x/text v0.14.0
	google.golang.org/grpc v1.63.2
	google.golang.org/protobuf v1.33.1-0.20240408130810-98873a205002
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/bufbuild/protocompile v0.12.0 // indirect
	github.com/caarlos0/env/v10 v10.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/gripmock/deeply v1.0.8 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.25.0 // indirect
	go.opentelemetry.io/otel/metric v1.26.0 // indirect
	go.opentelemetry.io/otel/trace v1.26.0 // indirect
	go.opentelemetry.io/proto/otlp v1.2.0 // indirect
	golang.org/x/exp v0.0.0-20240416160154-fe59bbe5cc7f // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240415180920-8c6c420018be // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240415180920-8c6c420018be // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// this is for example client to be able to run
replace github.com/bavix/gripmock/protogen v0.0.0 => ./protogen
