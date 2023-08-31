module github.com/tokopedia/gripmock

go 1.21

require (
	github.com/goccy/go-yaml v1.11.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/lithammer/fuzzysearch v1.1.8
	github.com/tokopedia/gripmock/protogen/example v0.0.0
	golang.org/x/text v0.12.0
	google.golang.org/grpc v1.57.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/fatih/color v1.10.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
)

// this is for generated server to be able to run
replace github.com/tokopedia/gripmock/protogen/example v0.0.0 => ./protogen/example

// this is for example client to be able to run
replace github.com/tokopedia/gripmock/protogen v0.0.0 => ./protogen
