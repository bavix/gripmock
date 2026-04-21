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

PROTOBUF_REPO=/tmp/gm-protobuf-repo
GOOGLEAPIS_REPO=/tmp/gm-googleapis-sdk

PROTOBUF_EXCLUDE_ARGS=\
	--exclude 'google/protobuf/compiler/**' \
	--exclude '**/*test*.proto' \
	--exclude '**/test_protos/**' \
	--exclude '**/sample_*.proto' \
	--exclude '**/internal_*.proto' \
	--exclude '**/late_loaded_*.proto'

GOOGLEAPIS_EXCLUDE_ARGS=\
	--exclude 'preview/**'

gen-imports:
	rm -rf $(PROTOBUF_REPO) $(GOOGLEAPIS_REPO)
	touch internal/pbs/protobuf.pbs internal/pbs/googleapis.pbs
	git clone --depth=1 https://github.com/protocolbuffers/protobuf.git $(PROTOBUF_REPO)
	git clone --depth=1 https://github.com/googleapis/googleapis.git $(GOOGLEAPIS_REPO)
	go run main.go proto export \
		--root $(PROTOBUF_REPO)/src \
		$(PROTOBUF_EXCLUDE_ARGS) \
		--out internal/pbs/protobuf.pbs
	go run main.go proto export \
		--root $(GOOGLEAPIS_REPO) \
		--import-root $(PROTOBUF_REPO)/src \
		$(GOOGLEAPIS_EXCLUDE_ARGS) \
		--out internal/pbs/googleapis.pbs
	rm -rf $(PROTOBUF_REPO) $(GOOGLEAPIS_REPO)

gen-sdk-examples:
	rm -rf pkg/sdk/internal/examplefds/gen
	mkdir -p pkg/sdk/internal/examplefds/gen
	protoc --proto_path=. --go_out=pkg/sdk/internal/examplefds/gen --go_opt=paths=source_relative --go_opt=Mexamples/projects/chat/service.proto=github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/chat --go-grpc_out=pkg/sdk/internal/examplefds/gen --go-grpc_opt=paths=source_relative --go-grpc_opt=Mexamples/projects/chat/service.proto=github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/chat examples/projects/chat/service.proto
	protoc --proto_path=. --go_out=pkg/sdk/internal/examplefds/gen --go_opt=paths=source_relative --go-grpc_out=pkg/sdk/internal/examplefds/gen --go-grpc_opt=paths=source_relative examples/projects/multiverse/service.proto
