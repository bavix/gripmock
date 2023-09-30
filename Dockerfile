ARG BUILD_ARG_GO_VERSION=1.21
ARG BUILD_ARG_ALPINE_VERSION=3.18
FROM golang:${BUILD_ARG_GO_VERSION}-alpine${BUILD_ARG_ALPINE_VERSION} AS builder

# install tools (bash, git, protobuf, protoc-gen-go, protoc-grn-go-grpc)
RUN apk -U --no-cache add bash git protobuf curl &&\
    go install -v google.golang.org/protobuf/cmd/protoc-gen-go@latest &&\
    go install -v google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest &&\
    # cloning well-known-types
    # only use needed files
    git clone --depth=1 https://github.com/protocolbuffers/protobuf.git /protobuf-repo &&\
    mv /protobuf-repo/src/ /protobuf/ &&\
    rm -rf /protobuf-repo &&\
    # cleanup
    apk del git &&\
    apk -v cache clean

COPY . /go/src/github.com/bavix/gripmock

# create necessary dirs and export fix_gopackage.sh
RUN mkdir /proto /stubs &&\
    ln -s /go/src/github.com/bavix/gripmock/fix_gopackage.sh /bin/

RUN cd /go/src/github.com/bavix/gripmock/protoc-gen-gripmock &&\
    go install -v &&\
    cd /go/src/github.com/bavix/gripmock/example/simple/client &&\
    go mod tidy &&\
    cd /go/src/github.com/bavix/gripmock &&\
    go install -v

WORKDIR /go/src/github.com/bavix/gripmock

EXPOSE 4770 4771

HEALTHCHECK CMD curl --fail http://127.0.0.1:4771/api/health/readiness

ENTRYPOINT ["gripmock"]
