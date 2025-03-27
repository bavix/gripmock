#!/usr/bin/env sh

# this file is used by .github/workflows/integration-test.yml

gripmock --stub=example/multi-files/stub example/multi-files/file1.proto \
  example/multi-files/file2.proto \
  example/multi-files/nested/file3.proto &

# wait for generated files to be available and gripmock is up
gripmock check --silent --timeout=30s

go run example/multi-files/client/*.go