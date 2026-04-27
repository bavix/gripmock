# Glob Matching Example

Demonstrates glob pattern matching for flexible stub routing.

## Service

`FileService.GetFile` - retrieves files by filename and path patterns.

## Stubs

| Pattern | Matches | Response |
|---------|---------|----------|
| `report_*.pdf` | `report_2024.pdf`, `report_final.pdf` | PDF report content |
| `*.json` | `config.json`, `data.json` | JSON file content |
| `/reports/*` | `/reports/2024`, `/reports/monthly` | Reports folder content |
| `data[12].csv` | `data1.csv`, `data2.csv` | Data file 1 or 2 |
| `test_?` | `test_a`, `test_1`, `test_x` | Single char wildcard |

## Run

```bash
go run main.go examples/projects/glob/service.proto --stub examples/projects/glob
```

## Test

```bash
grpctestify examples/projects/glob/
```

Or manually:

```bash
grpcurl -plaintext -d '{"filename": "report_2024.pdf", "path": "/reports/annual"}' localhost:4770 glob.FileService/GetFile
```
