# Search Service Example

This example demonstrates server streaming with multiple results and an empty stream (zero messages).

## Service

`SearchService.Search` is a server streaming method that returns a stream of `SearchResult` messages.

## Usage

```bash
gripmock --stub=./stubs.yaml ./service.proto
```

## Stub Configurations

| Query | Results | Description |
|-------|---------|-------------|
| `query: "grpc", category: "tech"` | 3 | Multiple search results |
| `query: "nonexistent"` | 0 | Empty stream (no matches) |
| `query: "specific"` | 1 | Single search result |
| `category: "empty"` | 0 | Empty stream by category |

## Empty Stream

Use `stream: []` to return zero messages and immediately close the stream with OK status. This is useful for scenarios like:
- Search with no matches
- Empty query results
- Filters that exclude all items
