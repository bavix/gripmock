# Embedded SDK examples

Runnable tests against an in-process GripMock.

```bash
go test ./examples/sdk/...
```

| Example | RPC shape |
|---|---|
| [billing](billing) | unary |
| [telemetry](telemetry) | server stream |
| [ingest](ingest) | client stream |
| [negotiation](negotiation) | bidirectional |
| [onboarding](onboarding) | unary + effects |

Regenerate after changing a `.proto`:

```bash
make gen-sdk-examples
```

See the [Embedded SDK guide](https://bavix.github.io/gripmock/guide/embedded-sdk/).
