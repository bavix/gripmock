#!/usr/bin/env sh

# this file is used by .github/workflows/integration-test.yml

gripmock --stub=example/stub-subfolders/stub example/stub-subfolders/stub-subfolders.proto &

# wait for generated files to be available and gripmock is up
gripmock check --timeout=30s

go run example/stub-subfolders/client/*.go