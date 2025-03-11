FROM alpine:3.21 AS protoc-builder

ENV PROTOC_VERSION=30.0
ARG TARGETARCH

#hadolint ignore=DL3018
RUN apk --no-cache add --virtual .build-deps git unzip coreutils findutils binutils \
    && if [ "$TARGETARCH" = "amd64" ]; then export DL_ARCH=x86_64 ; fi \
    && if [ "$TARGETARCH" = "arm64" ]; then export DL_ARCH=aarch_64 ; fi \
    && wget -qO /tmp/protoc.zip "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${DL_ARCH}.zip" \
    && unzip -q /tmp/protoc.zip -d /tmp/protoc && rm /tmp/protoc.zip \
    && mv /tmp/protoc/bin/protoc /usr/bin/protoc \
    && strip /usr/bin/protoc \
    && rm -rf /tmp/protoc \
    && mkdir -p /usr/include \
    && git clone --depth=1 https://github.com/protocolbuffers/protobuf.git /protobuf-repo \
    && mv /protobuf-repo/src/ /usr/include/protobuf/ \
    && git clone --depth=1 https://github.com/googleapis/googleapis.git /usr/include/googleapis \
    && rm -rf /protobuf-repo \
    && find /usr/include/protobuf -not -name "*.proto" -type f -delete \
    && find /usr/include/googleapis -not -name "*.proto" -type f -delete \
    && find /usr/include/protobuf -empty -type d -delete \
    && find /usr/include/googleapis -empty -type d -delete \
    && apk del --purge .build-deps \
    && rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

FROM golang:1.24-alpine3.21 AS builder

ARG version

RUN apk --no-cache add --virtual .build-deps binutils \
    && go install -v -ldflags "-s -w" google.golang.org/protobuf/cmd/protoc-gen-go@latest \
    && go install -v -ldflags "-s -w" google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest \
    && strip /go/bin/protoc-gen-go /go/bin/protoc-gen-go-grpc \
    && apk del .build-deps \
    && rm -rf /root/.cache /go/pkg /tmp/* /var/cache/apk/*

COPY . /go/src/github.com/bavix/gripmock

WORKDIR /go/src/github.com/bavix/gripmock/protoc-gen-gripmock

RUN apk add --no-cache binutils \
    && go build -o /go/bin/protoc-gen-gripmock -ldflags "-s -w" . \
    && strip /go/bin/protoc-gen-gripmock \
    && apk del binutils \
    && rm -rf /root/.cache /go/pkg/*

WORKDIR /go/src/github.com/bavix/gripmock

RUN apk add --no-cache binutils \
    && go build -o /go/bin/gripmock -ldflags "-X 'github.com/bavix/gripmock/cmd.version=${version:-dev}' -s -w" . \
    && strip /go/bin/gripmock \
    && apk del binutils \
    && rm -rf /root/.cache /go/pkg /tmp/* /var/cache/*

FROM golang:1.24-alpine3.21

LABEL org.opencontainers.image.source="https://github.com/bavix/gripmock"
LABEL org.opencontainers.image.description="gRPC Mock Server"
LABEL org.opencontainers.image.licenses="Apache-2.0"

COPY --from=protoc-builder /usr/bin/protoc /usr/bin/protoc
COPY --from=protoc-builder /usr/include/protobuf /protobuf
COPY --from=protoc-builder /usr/include/googleapis /googleapis

COPY --from=builder $GOPATH/bin/protoc-gen-go $GOPATH/bin/protoc-gen-go
COPY --from=builder $GOPATH/bin/protoc-gen-go-grpc $GOPATH/bin/protoc-gen-go-grpc
COPY --from=builder $GOPATH/bin/protoc-gen-gripmock $GOPATH/bin/protoc-gen-gripmock
COPY --from=builder $GOPATH/bin/gripmock $GOPATH/bin/gripmock

COPY . /go/src/github.com/bavix/gripmock

WORKDIR /go/src/github.com/bavix/gripmock

RUN go build -o /dev/null .

EXPOSE 4770 4771

HEALTHCHECK --start-interval=1s --start-period=30s \
    CMD gripmock check --silent

ENTRYPOINT ["gripmock"]
