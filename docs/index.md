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
  - title: "Automatic gRPC Server Generation 🚀"
    details: "Generates gRPC server implementation from your .proto files instantly."
  - title: "Dynamic Stub Management 🛠️"
    details: "Add, delete, and search stubs via REST API for on-the-fly mocking."
  - title: "Flexible Input Matching 🔍"
    details: "Supports equals, contains, matches rules with ignoreArrayOrder option for arrays."
  - title: "Header Matching Support 📦"
    details: "Validate and match gRPC request headers with regex and exact rules."
  - title: "gRPC Error Simulation ❌"
    details: "Return custom errors with specific gRPC status codes (e.g., NotFound, Internal)."
  - title: "Healthcheck Endpoints ❤️"
    details: "Built-in /health/liveness and /health/readiness for production readiness."
  - title: "Docker Integration 🐳"
    details: "Lightweight Docker image with minimal footprint for CI/CD workflows."
  - title: "Static Stub Initialization 📄"
    details: "Load predefined stubs from YAML/JSON files at startup."
  - title: "JSON Schema Validation 📋"
    details: "Comprehensive schema for validating stub definitions with IDE support."
---


