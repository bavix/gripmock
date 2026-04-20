# Dump

`gripmock dump` exports stubs from a running GripMock instance into files.

Use it to:

- persist runtime/captured stubs,
- convert API-created stubs into static fixtures,
- snapshot current mock state for CI.

## Prerequisites

1. GripMock is running.
2. HTTP admin API is reachable (default `127.0.0.1:4771`).

## Usage

```bash
gripmock dump
```

By default this writes YAML files to `./stubs_export`.

## Options

| Flag | Short | Default | Description |
|---|---|---|---|
| `--output` | `-o` | `stubs_export` | Output directory. |
| `--format` | — | `yaml` | Output format: `yaml` or `json`. |
| `--source` | — | *(empty)* | Filter by source: `file`, `rest`, `mcp`, `proxy`. |

## Source behavior

- If `--source` is omitted: exports all stubs except `file` source.
- If `--source` is set: exports only stubs of that source.

## Examples

Export default set (all except `file`):

```bash
gripmock dump --output ./stubs_export --format yaml
```

Export only proxy-captured stubs:

```bash
gripmock dump --source proxy --output ./captured_stubs
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
