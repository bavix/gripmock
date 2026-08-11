---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: GripMock
  text: Fast, honest gRPC mocks.
  tagline: gRPC-MockServer
  image: https://github.com/bavix/gripmock/assets/5111255/d33740c1-2c53-4c06-a7a7-d3a9cb6e7c00
  actions:
    - theme: brand
      text: Getting started
      link: /guide/introduction/
    - theme: alt
      text: Star on GitHub ⭐
      link: https://github.com/bavix/gripmock

features:
  - title: "Runtime `.pb` descriptor loading"
    details: "Start from compiled protobuf descriptors (`.pb`), with no proto source tree to manage."
  - title: "MCP API"
    details: "Model Context Protocol endpoints, for AI agents and tool-driven automation."
  - title: "No-restart stub updates"
    details: "Create, update and remove stubs at runtime through the API or the UI. The process keeps running."
  - title: "Dynamic templates"
    details: "Build a response out of the request payload, the headers and the stream position."
  - title: "Request matching"
    details: "Exact, partial, regex, glob and header matching, with priority rules deciding which stub answers."
  - title: "Full gRPC streaming"
    details: "Unary, server streaming, client streaming and bidirectional."
  - title: "Errors and delays"
    details: "Return gRPC status errors, and hold a response back long enough to exercise a client timeout."
  - title: "YAML and JSON stubs"
    details: "Both formats, with a JSON Schema for validation and editor autocomplete."
  - title: "Plugin system"
    details: "Add template functions in Go for domain-specific behaviour."
  - title: "OpenTelemetry and metrics"
    details: "Traces over OTLP, and a `/metrics` endpoint to scrape."
  - title: "Built-in faker"
    details: "Generate values from `faker.Person`, `faker.Geo`, `faker.Identity` and seven more domains."
  - title: "Upstream modes (experimental)"
    details: "`proxy`, `replay` and `capture` move a test from live upstream traffic to local mocks in stages."
  - title: "Docker ready"
    details: "19 MB image for local development, CI and containerized test environments."
  - title: "Embedded SDK (experimental)"
    details: "Run GripMock inside a Go test, with no external process to orchestrate."
---
