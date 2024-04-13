FROM golang:1.22-alpine3.19 AS builder

RUN apk --no-cache add git &&\
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
    find /googleapis -not -name "*.proto" -type f -delete

FROM golang:1.22-alpine3.19

LABEL org.opencontainers.image.source=https://github.com/bavix/gripmock
LABEL org.opencontainers.image.description="gRPC Mock Server"
LABEL org.opencontainers.image.licenses=Apache-2.0

ARG version

RUN apk --no-cache add protoc curl

COPY --from=builder /protobuf /protobuf
COPY --from=builder /googleapis /googleapis

COPY --from=builder $GOPATH/bin/protoc-gen-go $GOPATH/bin/protoc-gen-go
COPY --from=builder $GOPATH/bin/protoc-gen-go-grpc $GOPATH/bin/protoc-gen-go-grpc

COPY . /go/src/github.com/bavix/gripmock

WORKDIR /go/src/github.com/bavix/gripmock

RUN cd /go/src/github.com/bavix/gripmock/protoc-gen-gripmock &&\
    go install -v -ldflags "-s -w" &&\
    cd /go/src/github.com/bavix/gripmock &&\
    go install -v -ldflags "-X 'main.version=${version:-dev}' -s -w"

EXPOSE 4770 4771

HEALTHCHECK CMD curl --fail http://127.0.0.1:4771/api/health/readiness

ENTRYPOINT ["gripmock"]
