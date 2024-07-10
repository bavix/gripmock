#!/usr/bin/env sh

# this file is used by .github/workflows/integration-test.yml

gripmock --stub=example/stream/stub example/stream/stream.proto &

# wait for generated files to be available and gripmock is up
gripmock check --timeout=30s

go run example/stream/client/*.go