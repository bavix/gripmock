#!/usr/bin/env sh

# this file is used by .github/workflows/integration-test.yml

gripmock --stub=example/simple/stub example/simple/simple.proto &

# wait for generated files to be available and gripmock is up
gripmock check --silent --timeout=30s

go run example/simple/client/*.go