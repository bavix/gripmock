#!/usr/bin/env sh

# this file is used by .github/workflows/integration-test.yml

gripmock --stub=example/well_known_types/stub example/well_known_types/wkt.proto &

# wait for generated files to be available and gripmock is up
gripmock check --silent --timeout=30s

go run example/well_known_types/client/*.go