FROM golang:1.24-alpine3.21 AS builder

ARG version

COPY . /gripmock-src

WORKDIR /gripmock-src

#hadolint ignore=DL3018
RUN apk add --no-cache binutils \
    && go build -o /usr/local/bin/gripmock -ldflags "-X 'github.com/bavix/gripmock/v3/cmd.version=${version:-dev}' -s -w" . \
    && strip /usr/local/bin/gripmock \
    && apk del binutils \
    && rm -rf /root/.cache /go/pkg /tmp/* /var/cache/*

RUN chmod +x /gripmock-src/entrypoint.sh && chmod +x /usr/local/bin/gripmock

FROM alpine:3.21

LABEL org.opencontainers.image.source="https://github.com/bavix/gripmock"
LABEL org.opencontainers.image.description="gRPC Mock Server"
LABEL org.opencontainers.image.licenses="MIT"

COPY --from=builder /usr/local/bin/gripmock /usr/local/bin/gripmock
COPY --from=builder /gripmock-src/entrypoint.sh /entrypoint.sh

EXPOSE 4770 4771

HEALTHCHECK --start-interval=1s --start-period=30s \
    CMD gripmock check --silent

ENTRYPOINT ["/entrypoint.sh"]
