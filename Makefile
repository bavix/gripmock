OPENAPI=https://raw.githubusercontent.com/bavix/gripmock-openapi/master/api.yaml

.PHONY: *

version=latest

build:
	docker buildx build --load -t bavix/gripmock:${version} .

test:
	go test -tags mock -race -cover ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.0 run --color always ${args}

lint-fix:
	make lint args=--fix

semgrep:
	docker run --rm -v $$(pwd):/src bavix/semgrep:master semgrep scan --error --config=p/golang -f /semgrep-go

gen-rest:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest -generate gorilla,types -package rest ${OPENAPI} > internal/domain/rest/api.gen.go
	sed -i '' 's/interface{}/any/g' internal/domain/rest/api.gen.go

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
	rm -rf /tmp/gm-protobuf-sdk/google/protobuf/unittest_lite_edition_2024.proto
	# Remove test files that have invalid syntax
	find /tmp/gm-protobuf-sdk -name "*unittest*" -type f -delete
	find /tmp/gm-protobuf-sdk -name "*test*" -type f -delete
	# Remove files that depend on unittest files
	find /tmp/gm-protobuf-sdk -name "json_format*.proto" -type f -delete
	find /tmp/gm-protobuf-sdk -name "late_loaded_option*.proto" -type f -delete
	# Remove sample_messages_edition.proto which has invalid import syntax
	find /tmp/gm-protobuf-sdk -name "sample_messages_edition.proto" -type f -delete
	# Remove files with edition 2024 which is not supported
	find /tmp/gm-protobuf-sdk -name "internal_options.proto" -type f -delete
	find /tmp/gm-protobuf-sdk -name "*edition*.proto" -type f -delete
	mkdir -p internal/pbs 
	protoc --proto_path=/tmp/gm-googleapis-sdk --descriptor_set_out=internal/pbs/googleapis.pb --include_imports $$(find /tmp/gm-googleapis-sdk -name '*.proto')
	protoc --proto_path=/tmp/gm-protobuf-sdk --descriptor_set_out=internal/pbs/protobuf.pb --include_imports $$(find /tmp/gm-protobuf-sdk -name '*.proto')
	rm -rf /tmp/gm-protobuf-sdk /tmp/gm-googleapis-sdk
