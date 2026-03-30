OPENAPI=api/api.yaml

.PHONY: *

version=latest
GOLANGCI_LINT=go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4

build:
	docker buildx build --load -t bavix/gripmock:${version} .

test:
	go test -tags mock -race -cover ./...

lint:
	$(GOLANGCI_LINT) run --color always

lint-fix:
	$(GOLANGCI_LINT) run --color always --fix

lint-clean:
	$(GOLANGCI_LINT) cache clean

plugins:
	mkdir -p plugins; \
	for dir in examples/plugins/*; do \
		[ -d $$dir ] && go build -buildmode=plugin -o plugins/$$(basename $$dir).so $$dir/*.go; \
	done

semgrep:
	docker run --rm -v $$(pwd):/src bavix/semgrep:master semgrep scan --error --config=p/golang -f /semgrep-go

gen-rest:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest -generate gorilla,types -package rest ${OPENAPI} > internal/domain/rest/api.gen.go
	sed -i '' 's/interface{}/any/g' internal/domain/rest/api.gen.go
	gofmt -w internal/domain/rest/api.gen.go
	goimports -w internal/domain/rest/api.gen.go

gen-imports:
	rm -rf /tmp/gm-protobuf-repo /tmp/gm-googleapis-sdk /tmp/gm-protobuf-sdk
	git clone --depth=1 https://github.com/protocolbuffers/protobuf.git /tmp/gm-protobuf-repo
	mv /tmp/gm-protobuf-repo/src/ /tmp/gm-protobuf-sdk
	rm -rf /tmp/gm-protobuf-repo
	git clone --depth=1 https://github.com/googleapis/googleapis.git /tmp/gm-googleapis-sdk
	find /tmp/gm-protobuf-sdk -not -name "*.proto" -type f -delete
	find /tmp/gm-googleapis-sdk -not -name "*.proto" -type f -delete
	find /tmp/gm-protobuf-sdk -empty -type d -delete
	find /tmp/gm-googleapis-sdk -empty -type d -delete
	rm -rf /tmp/gm-protobuf-sdk/google/protobuf/compiler
	mkdir -p internal/pbs 
	protoc --proto_path=/tmp/gm-googleapis-sdk --descriptor_set_out=internal/pbs/googleapis.pb --include_imports $$(find /tmp/gm-googleapis-sdk -name '*.proto')
	protoc --proto_path=/tmp/gm-protobuf-sdk --descriptor_set_out=internal/pbs/protobuf.pb --include_imports $$(find /tmp/gm-protobuf-sdk -name '*.proto')
	rm -rf /tmp/gm-protobuf-sdk /tmp/gm-googleapis-sdk

gen-sdk-examples:
	rm -rf pkg/sdk/internal/examplefds/gen
	mkdir -p pkg/sdk/internal/examplefds/gen
	protoc --proto_path=. --go_out=pkg/sdk/internal/examplefds/gen --go_opt=paths=source_relative --go_opt=Mexamples/projects/chat/service.proto=github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/chat --go-grpc_out=pkg/sdk/internal/examplefds/gen --go-grpc_opt=paths=source_relative --go-grpc_opt=Mexamples/projects/chat/service.proto=github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/chat examples/projects/chat/service.proto
	protoc --proto_path=. --go_out=pkg/sdk/internal/examplefds/gen --go_opt=paths=source_relative --go-grpc_out=pkg/sdk/internal/examplefds/gen --go-grpc_opt=paths=source_relative examples/projects/multiverse/service.proto
