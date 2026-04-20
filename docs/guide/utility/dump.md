# Dump <VersionTag version="v3.10.1" />

`gripmock dump` exports stubs from a running GripMock instance into files.

Use it to:

- persist runtime/captured stubs,
- convert API-created stubs into static fixtures,
- snapshot current mock state for CI.

## Prerequisites

1. GripMock is running.
2. Admin API is reachable at the address from environment variables.

## Usage

```bash
gripmock dump
```

By default writes YAML files to `./stubs_export`, connecting to `http://` + `HTTP_ADDR`.

## Options

| Flag | Short | Default | Description |
|---|---|---|---|
| `--output` | `-o` | `stubs_export` | Output directory. |
| `--format` | — | `yaml` | Output format: `yaml` or `json`. |
| `--scheme` | — | `http` | URL scheme: `http` or `https`. |
| `--source` | — | *(empty)* | Filter by source: `file`, `rest`, `mcp`, `proxy`. |

## Source behavior

- If `--source` is omitted: exports all stubs except `file` source.
- If `--source` is set: exports only stubs of that source.

## Examples

Export default set (all except `file`):

```bash
gripmock dump --output ./stubs_export
```

Export only proxy-captured stubs:

```bash
gripmock dump --source proxy --output ./captured_stubs
```

Export from HTTPS endpoint:

```bash
gripmock dump --scheme https --output ./secure_stubs
```

Export from a custom address via environment:

```bash
HTTP_ADDR=10.0.0.5:4771 gripmock dump --output ./remote_stubs
```

Export only API-created stubs in JSON:

```bash
gripmock dump --source rest --format json --output ./api_stubs
```

## Output details

- Stubs are grouped by `service + method` into files.
- File names are sanitized (`.`, `/`, `\\`, `:` become `_`).
- Records include:
  - `service`, `method`
  - `input`/`inputs`
  - `headers`
  - `output`
  - optional `_meta.source`

After export, command prints:

```text
total: <files> files, <stubs> stubs
```
