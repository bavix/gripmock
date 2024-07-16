FROM golang:1.22-alpine3.20 AS builder

RUN apk --no-cache add git &&\
    go install -v -ldflags "-s -w" google.golang.org/protobuf/cmd/protoc-gen-go@latest &&\
    go install -v -ldflags "-s -w" google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest &&\
    # cloning googleapis-types
    git clone --depth=1 https://github.com/googleapis/googleapis.git /googleapis &&\
    # cleanup
    find /googleapis -not -name "*.proto" -type f -delete

COPY . /go/src/github.com/bavix/gripmock

WORKDIR /go/src/github.com/bavix/gripmock/protoc-gen-gripmock

RUN go install -v -ldflags "-s -w"

FROM golang:1.22-alpine3.20 as protoc-builder

ENV PROTOC_VERSION=27.2

RUN apk --no-cache add curl unzip &&\
    if [ `uname -m` = "amd64" ]; then export DL_ARCH=x86_64 ; fi \
    && if [ `uname -m` = "arm64" ]; then export DL_ARCH=aarch_64 ; fi \
    && if [ `uname -m` = "aarch64" ]; then export DL_ARCH=aarch_64 ; fi \
    && curl -f -L -o /tmp/protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${DL_ARCH}.zip \
    && unzip /tmp/protoc.zip \
    && rm /tmp/protoc.zip \
    && mv bin/protoc /usr/bin \
    && mkdir /protobuf \
    && mv include/* /protobuf

FROM golang:1.22-alpine3.20

LABEL org.opencontainers.image.source=https://github.com/bavix/gripmock
LABEL org.opencontainers.image.description="gRPC Mock Server"
LABEL org.opencontainers.image.licenses=Apache-2.0

ARG version

COPY --from=protoc-builder /usr/bin/protoc /usr/bin/protoc
COPY --from=protoc-builder /protobuf /protobuf
COPY --from=builder /googleapis /googleapis

COPY --from=builder $GOPATH/bin/protoc-gen-go $GOPATH/bin/protoc-gen-go
COPY --from=builder $GOPATH/bin/protoc-gen-go-grpc $GOPATH/bin/protoc-gen-go-grpc
COPY --from=builder $GOPATH/bin/protoc-gen-gripmock $GOPATH/bin/protoc-gen-gripmock

COPY . /go/src/github.com/bavix/gripmock

WORKDIR /go/src/github.com/bavix/gripmock

RUN go install -v -ldflags "-X 'github.com/bavix/gripmock/cmd.version=${version:-dev}' -s -w"

EXPOSE 4770 4771

HEALTHCHECK CMD gripmock check --silent

ENTRYPOINT ["gripmock"]
