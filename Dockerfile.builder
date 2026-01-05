# GripMock Builder Image
# This image provides a stable and consistent toolchain for building GripMock binaries and plugins.
# It ensures reproducible builds with pinned Go version and system libraries, preventing runtime
# incompatibilities such as plugin version mismatches, CGO symbol resolution errors, and libc issues.

FROM golang:1.25-alpine3.23

LABEL org.opencontainers.image.title="GripMock Builder"
LABEL org.opencontainers.image.description="Builder image for GripMock with consistent toolchain for reproducible builds"
LABEL org.opencontainers.image.source="https://github.com/bavix/gripmock"
LABEL org.opencontainers.image.documentation="https://bavix.github.io/gripmock/"
LABEL org.opencontainers.image.authors="Bavix Team <info@babichev.net>"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.vendor="bavix"

# Install build essentials
# - binutils: provides 'strip' for binary optimization
# - gcc: C compiler for CGO support
# - musl-dev: C standard library development files for Alpine (musl libc)
# - git: version control for go modules
#hadolint ignore=DL3018,DL3003
RUN apk update && apk add --no-cache \
    binutils \
    gcc \
    musl-dev \
    git || \
    (sleep 10 && apk update && apk add --no-cache binutils gcc musl-dev git)

# Set CGO environment variables for consistent builds
ENV CGO_ENABLED=1
ENV CC=gcc
ENV CXX=g++

# Set working directory
WORKDIR /workspace

# Verify Go version
RUN go version

# Default command
CMD ["/bin/sh"]
