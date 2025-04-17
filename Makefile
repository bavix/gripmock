OPENAPI=https://raw.githubusercontent.com/bavix/gripmock-openapi/master/api.yaml

.PHONY: *

version=latest
target_image=bavix/gripmock:${version}

build:
	docker buildx build --load -t ${target_image} .

build-manifest:
	make build-slim arch=arm64
	make build-slim arch=amd64
	docker push "${target_image}-slim-amd64"
	docker push "${target_image}-slim-arm64"
	docker manifest create "${target_image}-slim" \
		--amend "${target_image}-slim-arm64" \
		--amend "${target_image}-slim-amd64"

build-slim:
	mint --report=off slim \
		--image-build-arch=${arch} \
		--preserve-path "/go/src/github.com/bavix/gripmock/example,/protobuf,/googleapis" \
		--include-path /usr/local/go \
		--mount ./example/well_known_types/entrypoint.sh:/go/src/github.com/bavix/gripmock/entrypoint.sh \
		--tag "${target_image}-slim-${arch}" \
		--target "${target_image}"

test:
	go test -tags mock -race -cover ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.1 run --color always ${args}

lint-fix:
	make lint args=--fix

intgr-test: build
	docker compose -f deployments/docker-compose/docker-compose.yml up

gen:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest -generate gorilla,types -package rest ${OPENAPI} > internal/domain/rest/api.gen.go

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
	mkdir -p internal/pbs 
	protoc --proto_path=/tmp/gm-googleapis-sdk --descriptor_set_out=internal/pbs/googleapis.pb --include_imports $$(find /tmp/gm-googleapis-sdk -name '*.proto')
	protoc --proto_path=/tmp/gm-protobuf-sdk --descriptor_set_out=internal/pbs/protobuf.pb --include_imports $$(find /tmp/gm-protobuf-sdk -name '*.proto')
	rm -rf /tmp/gm-protobuf-sdk /tmp/gm-googleapis-sdk
