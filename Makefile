OPENAPI=https://raw.githubusercontent.com/bavix/gripmock-openapi/master/api.yaml

.PHONY: *

version=latest

build:
	docker buildx build --load -t "bavix/gripmock:${version}" .

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
