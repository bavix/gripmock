GOLANGCI_LING_IMAGE="golangci/golangci-lint:v1.54.2-alpine"

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
