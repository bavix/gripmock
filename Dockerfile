FROM golang:1.24-alpine3.21 AS builder

ARG version

COPY . /go/src/github.com/bavix/gripmock

WORKDIR /go/src/github.com/bavix/gripmock

#hadolint ignore=DL3018
RUN apk add --no-cache binutils \
    && go build -o /go/bin/gripmock -ldflags "-X 'github.com/bavix/gripmock/cmd.version=${version:-dev}' -s -w" . \
    && strip /go/bin/gripmock \
    && apk del binutils \
    && rm -rf /root/.cache /go/pkg /tmp/* /var/cache/*

FROM golang:1.24-alpine3.21

LABEL org.opencontainers.image.source="https://github.com/bavix/gripmock"
LABEL org.opencontainers.image.description="gRPC Mock Server"
LABEL org.opencontainers.image.licenses="MIT,Apache-2.0"

COPY --from=builder /go/bin/gripmock /gripmock
COPY --from=builder /go/src/github.com/bavix/gripmock/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && chmod +x /gripmock && ln -s /gripmock /usr/local/bin/gripmock

# for debug. remove it in feature
COPY --from=builder /go/src/github.com/bavix/gripmock /go/src/github.com/bavix/gripmock

WORKDIR /go/src/github.com/bavix/gripmock

EXPOSE 4770 4771

HEALTHCHECK --start-interval=1s --start-period=30s \
    CMD gripmock check --silent

ENTRYPOINT ["/entrypoint.sh"]
