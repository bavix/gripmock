FROM golang:1.26-alpine3.23

LABEL org.opencontainers.image.title="GripMock Builder"
LABEL org.opencontainers.image.description="Builder image for GripMock runtime and Go plugins"
LABEL org.opencontainers.image.source="https://github.com/bavix/gripmock"
LABEL org.opencontainers.image.documentation="https://bavix.github.io/gripmock/"
LABEL org.opencontainers.image.authors="Babichev Maxim <info@babichev.net>"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.vendor="bavix"

# hadolint ignore=DL3018
RUN apk add --no-cache git build-base binutils

WORKDIR /work
