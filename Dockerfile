ARG BUILDER_IMAGE=golang:1.26-alpine3.23

FROM ${BUILDER_IMAGE} AS builder

ARG version

COPY . /gripmock-src

WORKDIR /gripmock-src

#hadolint ignore=DL3018
RUN apk add --no-cache binutils \
    && go build -o /usr/local/bin/gripmock -ldflags "-X 'github.com/bavix/gripmock/v3/internal/infra/build.Version=${version:-dev}' -s -w" . \
    && strip /usr/local/bin/gripmock \
    && apk del binutils \
    && rm -rf /root/.cache /go/pkg /tmp/* /var/cache/*

RUN chmod +x /usr/local/bin/gripmock

FROM alpine:3.23

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
    CMD gripmock check --silent

ENTRYPOINT ["/usr/local/bin/gripmock"]
