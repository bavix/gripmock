FROM golang:1.24-alpine3.21 AS protoc-builder

ENV PROTOC_VERSION=29.3
ARG TARGETARCH

#hadolint ignore=DL3018
RUN apk --no-cache add git unzip \
    && if [ $TARGETARCH = "amd64" ]; then export DL_ARCH=x86_64 ; fi \
    && if [ $TARGETARCH = "arm64" ]; then export DL_ARCH=aarch_64 ; fi \
    && wget --no-verbose -O /tmp/protoc.zip "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${DL_ARCH}.zip" \
    && unzip /tmp/protoc.zip && rm /tmp/protoc.zip \
    && mv bin/protoc /usr/bin && rm -rf bin include \
    && mkdir -p /usr/include \
    # cloning well-known-types
    && git clone --depth=1 https://github.com/protocolbuffers/protobuf.git /protobuf-repo \
    && mv /protobuf-repo/src/ /usr/include/protobuf/ \
    # cloning googleapis-types
    && git clone --depth=1 https://github.com/googleapis/googleapis.git /usr/include/googleapis \
    # cleanup
    && rm -rf /protobuf-repo \
    && find /usr/include/protobuf -not -name "*.proto" -type f -delete \
    && find /usr/include/googleapis -not -name "*.proto" -type f -delete

FROM golang:1.24-alpine3.21 AS builder

ARG version

RUN go install -v -ldflags "-s -w" google.golang.org/protobuf/cmd/protoc-gen-go@latest  \
    && go install -v -ldflags "-s -w" google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest \
    && rm -rf /root/.cache

COPY . /go/src/github.com/bavix/gripmock

WORKDIR /go/src/github.com/bavix/gripmock/protoc-gen-gripmock

RUN go install -v -ldflags "-s -w" \
    && rm -rf /root/.cache

WORKDIR /go/src/github.com/bavix/gripmock

RUN go install -v -ldflags "-X 'github.com/bavix/gripmock/cmd.version=${version:-dev}' -s -w" \
    && rm -rf /root/.cache

FROM golang:1.24-alpine3.21

LABEL org.opencontainers.image.source=https://github.com/bavix/gripmock
LABEL org.opencontainers.image.description="gRPC Mock Server"
LABEL org.opencontainers.image.licenses=Apache-2.0

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

HEALTHCHECK CMD gripmock check --silent

ENTRYPOINT ["gripmock"]
