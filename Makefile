OPENAPI=api/api.yaml

.PHONY: *

version=latest

build:
	docker buildx build --load -t bavix/gripmock:${version} .

test:
	go test -tags mock -race -cover ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0 run --color always ${args}

lint-fix:
	make lint args=--fix

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
