# Response Templates <VersionTag version="v3.22.0" />

Leaf templates fill values into a response you already described. When the *shape*
of the response depends on the request — how many items an array holds, how many
messages a stream sends, whether a field is there at all — describe the whole
response with `output.template` instead.

## How it works

`output.template: true` marks `data` or `stream` as a Go template rendered once per
request; what it prints is the response. The slot says what the stub answers with: a
template in `data` fills the single response message, a template in `stream` fills the
stream. `dict` and `list` print themselves as JSON, so building a response is one
expression.

::: v-pre
```yaml
- service: catalog.CatalogService
  method: Search
  input:
    contains: {}
  output:
    template: true
    data: |
      {{ $matched := .Request.catalog | where "stock" "gte" .Request.min_stock }}
      {{ $page := $matched | page .Request.offset .Request.limit }}
      {{ dict "items"     $page
              "matched"   (len $matched)
              "pageTotal" (sum (extract $page "price"))
              "facets"    (countBy "category" $page) }}
```
:::

Query helpers take the collection last, so they chain with `|`; the older `extract`
keeps its own order. Statements may sit on their own lines — the whitespace they
leave behind is ignored, so <code v-pre>{{- -}}</code> trimming is not needed.

| Function | Purpose |
|---|---|
| `where field op value coll` | keep matching items: `eq`, `ne`, `gt`, `gte`, `lt`, `lte` |
| `page offset limit coll` | clamped window; `limit` 0 means until the end |
| `countBy field coll` | count items per field value |
| `extract coll field` | pull one field out of every item, for `sum`/`avg`/`min`/`max` |
| `seq n` | indexes `0..n-1`, for "N messages" |
| `dict "k" v …` / `set obj "k" v` | build and extend an object |
| `list a b …` / `append list items…` | build and grow a list |

Everything from [Dynamic Templates](./dynamic-templates) is available too:
`.Request`, `.Requests`, `.Headers`, `.MessageIndex`, `.RequestTime`, `.StubID`,
`faker`, the math and time helpers. `toJson` still encodes anything explicitly —
plain request data, for instance, which prints as JSON only once a helper has
touched it.

## Streams

Every JSON value the template prints is one message, so a `range` is the whole
story — nothing has to be accumulated first:

::: v-pre
```yaml
output:
  template: true
  stream: |
    {{ range $tick := seq .Request.ticks }}
      {{ range $sku := $.Request.skus }}
        {{ dict "_gripmock" (dict "delay" "50ms")
                "sku"       $sku
                "seq"       $tick }}
      {{ end }}
    {{ end }}
```
:::

A `stream` template that prints nothing sends an empty stream. Printing one array works
too — its elements are the messages — which is what <code v-pre>{{ .Request.items | where … }}</code>
gives you for free.

A generated message may carry the reserved `_gripmock` key, exactly like a literal
stream element: `delay` waits before that message, `error`/`code`/`details` end the
stream at that position. See [Delay](./delay).

In a bidirectional stub the template is rendered for every received message and
everything it prints is sent, so `inputs` still selects the stub but no longer pairs
`inputs[i]` with `stream[i]`. Use a literal `stream` when you need that pairing. For the
same reason a stub that carries `inputs` together with `template` is treated as
bidirectional when stubs are ranked, even if the method is client-streaming.

`output.delay` applies to each rendered message, exactly as it does to literal stream
elements; `_gripmock.delay` on a message overrides it.

## Optional fields and `oneof`

A field whose value is `null` is left out of the response, and a request field the
client did not set — an unset `oneof` branch, an absent message, an empty repeated
field — is absent from `.Request`. Echoing it therefore gives an optional field for
free, and mirroring a `oneof` needs no branching:

::: v-pre
```yaml
output:
  template: true
  data: |
    {{ dict "text"   (index .Request "text")
            "number" (index .Request "number") }}
```
:::

The client sets `text`, the response carries `text`; it sets `number`, the response
carries `number`. Use <code v-pre>{{ if }}</code> when the choice depends on a computed condition
rather than on what the request carried.

## In JSON

Stub files are YAML or JSON, and the REST API takes JSON. Nothing changes but the
quoting — a template is one string, and it stays readable on one line:

```json
[
  {
    "service": "catalog.CatalogService",
    "method": "WatchStock",
    "input": { "contains": {} },
    "output": {
      "template": true,
      "stream": "{{ range $i := seq .Request.ticks }}{{ dict \"seq\" $i }}{{ end }}"
    }
  }
]
```

## Rules and limits

- A `data` template must print exactly one JSON value; a `stream` template may print any
  number of them, or a single array of messages. `_gripmock` belongs to stream messages
  only — in a `data` template it is an error.
- Values taken from the request are never re-rendered, so a <code v-pre>{{ }}</code> arriving in
  request data is inert.
- The math helpers answer with numbers, and `index` wants an integer, so cast a computed
  position: <code v-pre>{{ index $items (int (sub (len $items) 1)) }}</code>.
- A template failure or invalid JSON fails the call with `Internal` and echoes the
  beginning of the rendered document. A document that is valid JSON but does not fit the
  response message fails exactly as a literal `data` with that shape would — the
  conversion error, with `Unknown`.
- Limits: at most 10 000 messages per stream document, 8 MiB of rendered text, and
  `seq` counts to 10 000. Rendering is not interrupted by the client going away, so a
  runaway template runs to one of those limits.
- `health` stubs do not support templates: such a stub is rejected when it is added, so
  every transport agrees. Use a literal `data`/`stream`.
- Without a proto descriptor (Connect/grpc-web fallback) a template stub answers
  `Unimplemented`: there is nothing to encode the rendered document into.
- The template is parsed when the stub is loaded or added over the REST/MCP API, so a
  syntax error surfaces there rather than on the first call. A runtime failure — say
  `dict` with an odd number of arguments — still surfaces on the call.
- What `GET /api/stubs`, `/api/stubs/search`, the MCP `stubs_*` tools and the UI show
  for such a stub is the template text; only MCP `mock_call` renders it. `gripmock dump`
  writes the template as a block scalar and reloads it unchanged.
- A stub captured by the proxy is inserted next to your stubs with an exact-match
  matcher, so for that one request it can outrank a template stub.

## Context per message

A server-streaming document is rendered **once**, so <code v-pre>{{ .MessageIndex }}</code> is `0`
throughout and <code v-pre>{{ now }}</code>, `uuid` and `faker` give the same value in every message of
that response. A literal `stream` is rendered per message and varies. In a
bidirectional stub the document is rendered for every received message, so
`.MessageIndex` advances there.

## Load time

Nothing written under `output` is evaluated when a stub file is loaded — that is what
keeps a template alive until the request arrives. The rule is enforced by a line scanner
that works on the text under the key, so anchoring the key or the document itself is
safe (`output: &out`, `data: &doc |`, `<<: *base`), and so is a quoted one-liner broken
across two lines. JSON stub files skip the scanner entirely.

Anchored documents work as well: the scanner pre-collects which aliases are
referenced from `output` and defers their source blocks too, so this renders per
request like a document written inline:

::: v-pre
```yaml
- service: catalog.CatalogService
  method: Search
  x-doc: &shared |
    {{ dict "id" (uuid) }}
  output:
    template: true
    data: *shared     # rendered per request: a fresh uuid every call
```
:::

Actions that need the request defer anyway and stay dynamic; it is the ones that can be
resolved without it — `uuid`, `now`, `faker`, plain constants — that would freeze only
outside `output`. Write the document under `output` itself, or anchor it anywhere and
alias it in, and it always behaves.

See also: [Dynamic Templates](./dynamic-templates), [Output](./output-stream),
[Streaming](./streaming). A runnable example lives in `examples/projects/catalog`: it
filters and pages a catalog, counts categories into a map, and streams one event per
requested sku per tick.
