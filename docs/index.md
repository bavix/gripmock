---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: GripMock
  text: Fast. Just. Comfortable.
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
  - title: "Runtime `.pb` Descriptor Loading ⚡"
    details: "Start mocks from compiled protobuf descriptors (`.pb`) instantly, without managing proto source trees."
  - title: "MCP API 🤖"
    details: "Expose Model Context Protocol endpoints for AI agents and tool-driven automation workflows."
  - title: "No-Restart Stub Updates ♻️"
    details: "Create, update, and remove stubs at runtime through API or UI with zero process restarts."
  - title: "Dynamic Templates 🎭"
    details: "Generate realistic responses from request payloads, headers, and stream context in real time."
  - title: "Smart Request Matching 🔍"
    details: "Combine exact, partial, regex, and header matching with priority rules for deterministic stub selection."
  - title: "Full gRPC Streaming Support 🔄"
    details: "Test all gRPC interaction patterns: unary, server streaming, client streaming, and bidirectional streaming."
  - title: "Error & Delay Simulation ❌"
    details: "Simulate production behavior with gRPC status errors and configurable response delays."
  - title: "YAML & JSON Stubs 📝"
    details: "Write stubs in YAML or JSON with JSON Schema validation and IDE autocomplete support."
  - title: "Plugin System 🔌"
    details: "Extend templates with custom Go plugins to implement domain-specific behavior and shared logic."
  - title: "OpenTelemetry + Metrics 📈"
    details: "Export traces via OTLP and scrape `/metrics` for runtime and GripMock metrics in observability stacks."
  - title: "Built-in Faker for Templates 🎲"
    details: "Generate realistic dynamic values in stubs using semantic groups like `faker.Person`, `faker.Geo`, and `faker.Identity`."
  - title: "Upstream Modes (Experimental) 🔁"
    details: "Use `proxy`, `replay`, and `capture` modes to move from live upstream traffic to local deterministic mocks in controlled stages."
  - title: "Docker Ready 🐳"
    details: "Use optimized images in local development, CI pipelines, and containerized test environments."
  - title: "Embedded SDK (Experimental) 🧪"
    details: "Embed GripMock directly in Go tests and internal tooling without external process orchestration."
---
