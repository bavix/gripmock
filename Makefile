OPENAPI=api/api.yaml

.PHONY: *

version=latest
GOLANGCI_LINT=go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1

build:
	docker buildx build --load -t bavix/gripmock:${version} .

test:
	go test -tags mock -race -cover ./...

examples-tls12:
	GRPC_PORT=5780 HTTP_PORT=5781 GRPC_TLS_CERT_FILE=tls-suite/tls12/certs/server.crt GRPC_TLS_KEY_FILE=tls-suite/tls12/certs/server.key GRPC_TLS_MIN_VERSION=1.2 go run main.go tls-suite/tls12/service.proto --stub tls-suite/tls12

examples-tls13:
	GRPC_PORT=5782 HTTP_PORT=5783 GRPC_TLS_CERT_FILE=tls-suite/tls13/certs/server.crt GRPC_TLS_KEY_FILE=tls-suite/tls13/certs/server.key GRPC_TLS_MIN_VERSION=1.3 go run main.go tls-suite/tls13/service.proto --stub tls-suite/tls13

examples-mtls:
	GRPC_PORT=5784 HTTP_PORT=5785 GRPC_TLS_CERT_FILE=tls-suite/mtls/certs/server.crt GRPC_TLS_KEY_FILE=tls-suite/mtls/certs/server.key GRPC_TLS_CLIENT_AUTH=true GRPC_TLS_CA_FILE=tls-suite/mtls/certs/ca.crt go run main.go tls-suite/mtls/service.proto --stub tls-suite/mtls

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
