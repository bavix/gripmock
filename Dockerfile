FROM golang:1.27-alpine3.23 AS builder

ARG version
ARG commit
ARG date

WORKDIR /gripmock-src

#hadolint ignore=DL3018
RUN apk add --no-cache binutils

# Module download is its own layer so a source-only change reuses it.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /usr/local/bin/gripmock -ldflags "-X 'github.com/bavix/gripmock/v3/internal/infra/build.Version=${version:-dev}' -X 'github.com/bavix/gripmock/v3/internal/infra/build.Commit=${commit:-unknown}' -X 'github.com/bavix/gripmock/v3/internal/infra/build.Date=${date:-}' -s -w" . \
    && strip /usr/local/bin/gripmock \
    && chmod +x /usr/local/bin/gripmock

FROM alpine:3.24

LABEL org.opencontainers.image.title="GripMock" 
LABEL org.opencontainers.image.description="Mock server for gRPC services with dynamic stubbing capabilities"
LABEL org.opencontainers.image.source="https://github.com/bavix/gripmock"
LABEL org.opencontainers.image.documentation="https://bavix.github.io/gripmock/"
LABEL org.opencontainers.image.authors="Babichev Maxim <info@babichev.net>"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.vendor="bavix"

COPY --from=builder /usr/local/bin/gripmock /usr/local/bin/gripmock

EXPOSE 4770 4771

HEALTHCHECK --start-interval=1s --start-period=30s \
    CMD ["/usr/local/bin/gripmock", "check", "--silent"]

ENTRYPOINT ["/usr/local/bin/gripmock"]
