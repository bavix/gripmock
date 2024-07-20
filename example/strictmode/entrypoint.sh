#!/usr/bin/env sh

# this file is used by .github/workflows/integration-test.yml

STRICT_METHOD_TITLE=false gripmock \
    --stub=example/strictmode/stub \
    example/strictmode/method.proto &

# wait for generated files to be available and gripmock is up
gripmock check --silent --timeout=30s

go run example/strictmode/client/*.go