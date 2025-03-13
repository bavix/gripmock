OPENAPI=https://raw.githubusercontent.com/bavix/gripmock-openapi/master/api.yaml

.PHONY: *

version=latest
target_image=bavix/gripmock:${version}

build:
	docker buildx build --load -t ${target_image} .

build-slim:
	mint --report=off slim \
		--obfuscate-metadata \
		--preserve-path "/go/src/github.com/bavix/gripmock/example" \
		--include-path /usr/local/go \
		--mount ./example/well_known_types/entrypoint.sh:/go/src/github.com/bavix/gripmock/entrypoint.sh \
		--tag "${target_image}-slim" \
		--target "${target_image}"

test:
	go test -tags mock -race -cover ./...

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.6 run --color always ${args}

lint-fix:
	make lint args=--fix

intgr-test: build
	docker compose -f deployments/docker-compose/docker-compose.yml up

gen:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest -generate gorilla,types -package rest ${OPENAPI} > internal/domain/rest/api.gen.go
