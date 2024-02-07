ARG BUILD_ARG_GO_VERSION=1.22
ARG BUILD_ARG_ALPINE_VERSION=3.19
FROM golang:${BUILD_ARG_GO_VERSION}-alpine${BUILD_ARG_ALPINE_VERSION} AS builder

LABEL org.opencontainers.image.source=https://github.com/bavix/gripmock
LABEL org.opencontainers.image.description="gRPC Mock Server"
LABEL org.opencontainers.image.licenses=Apache-2.0

ARG version

# install tools (git, protobuf, protoc-gen-go, protoc-grn-go-grpc)
RUN apk -U --no-cache add git protobuf curl &&\
    go install -v -ldflags "-s -w" google.golang.org/protobuf/cmd/protoc-gen-go@latest &&\
    go install -v -ldflags "-s -w" google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest &&\
    # cloning well-known-types
    git clone --depth=1 https://github.com/protocolbuffers/protobuf.git /protobuf-repo &&\
    mv /protobuf-repo/src/ /protobuf/ &&\
    # cloning googleapis-types
    git clone --depth=1 https://github.com/googleapis/googleapis.git /googleapis &&\
    # cleanup
    rm -rf /protobuf-repo &&\
    find /protobuf -not -name "*.proto" -type f -delete &&\
    find /googleapis -not -name "*.proto" -type f -delete &&\
    apk del git &&\
    apk -v cache clean

COPY . /go/src/github.com/bavix/gripmock

RUN cd /go/src/github.com/bavix/gripmock/protoc-gen-gripmock &&\
    go install -v -ldflags "-s -w" &&\
    cd /go/src/github.com/bavix/gripmock &&\
    go install -v -ldflags "-X 'main.version=${version:-dev}' -s -w"

WORKDIR /go/src/github.com/bavix/gripmock

EXPOSE 4770 4771

HEALTHCHECK CMD curl --fail http://127.0.0.1:4771/api/health/readiness

ENTRYPOINT ["gripmock"]
