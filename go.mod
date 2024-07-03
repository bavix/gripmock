module github.com/bavix/gripmock

go 1.22.2

require (
	github.com/bavix/features v1.0.0
	github.com/bavix/gripmock-sdk-go v1.0.4
	github.com/bavix/gripmock-ui v1.0.0-alpha4
	github.com/bavix/gripmock/protogen v0.0.0
	github.com/goccy/go-yaml v1.11.3
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gripmock/environment v1.0.1
	github.com/gripmock/grpc-interceptors v1.0.1
	github.com/gripmock/shutdown v1.0.0
	github.com/gripmock/stuber v1.0.1
	github.com/jhump/protoreflect v1.16.0
	github.com/oapi-codegen/runtime v1.1.1
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.33.0
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.52.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.52.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.52.0
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0
	go.opentelemetry.io/otel/sdk v1.28.0
	golang.org/x/text v0.16.0
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/bufbuild/protocompile v0.13.0 // indirect
	github.com/caarlos0/env/v10 v10.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/gripmock/deeply v1.0.9 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	golang.org/x/exp v0.0.0-20240531132922-fd00a4e0eefc // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// this is for example client to be able to run
replace github.com/bavix/gripmock/protogen v0.0.0 => ./protogen
