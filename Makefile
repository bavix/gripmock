GOLANGCI_LING_IMAGE="golangci/golangci-lint:v1.55.1-alpine"

.PHONY: *

version=latest

build:
	docker buildx build --load -t "bavix/gripmock:${version}" --platform linux/arm64 .

test:
	go test -tags mock -race -cover ./...

lint:
	docker run --rm -v ./:/app -w /app $(GOLANGCI_LING_IMAGE) golangci-lint run --color always ${args}

lint-fix:
	make lint args=--fix

intgr-test: build
	docker compose -f deployments/docker-compose/docker-compose.yml up

gen:
	go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest -generate gorilla,types -package rest ./api/openapi/api.yaml > internal/domain/rest/api.gen.go
	go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest -generate client,types -package sdk ./api/openapi/api.yaml | sed -e 's/json\.Marshal/Marshal/g' -e 's/json\.Unmarshal/Unmarshal/g' > pkg/sdk/api.gen.go
