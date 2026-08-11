# Schema Validation

How to check stub files against the JSON Schema — from the command line, from an editor, and in CI.

## Command Line Validation

### Basic JSON Validation

```bash
# Validate JSON syntax
python -m json.tool your-stubs.json

# Validate YAML syntax
python -c "import yaml; yaml.safe_load(open('your-stubs.yaml'))"
```

### Schema Validation

Install the required package:

```bash
pip install jsonschema
```

Validate against the schema:

```bash
# Validate JSON file
jsonschema -i your-stubs.json https://bavix.github.io/gripmock/schema/stub.json

# Validate YAML file (convert to JSON first)
python -c "import yaml, json; print(json.dumps(yaml.safe_load(open('your-stubs.yaml'))))" | jsonschema -i - https://bavix.github.io/gripmock/schema/stub.json
```

## IDE Validation

### VS Code

1. Install the "YAML" extension
2. Add schema reference to your files:

```yaml
# yaml-language-server: $schema=https://bavix.github.io/gripmock/schema/stub.json
```

3. Validation and completion then work as you type.

## CI/CD Integration

### GitHub Actions

```yaml
name: Validate Stubs

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      
      - name: Set up Python
        uses: actions/setup-python@v6
        with:
          python-version: '3.9'
          
      - name: Install dependencies
        run: |
          pip install jsonschema pyyaml
          
      - name: Validate JSON stubs
        run: |
          find . -path "*/stubs/*" -name "*.json" -print0 | while IFS= read -r -d "" file; do
            echo "Validating $file"
            jsonschema -i "$file" https://bavix.github.io/gripmock/schema/stub.json
          done
          
      - name: Validate YAML stubs
        run: |
          find . -path "*/stubs/*" \( -name "*.yaml" -o -name "*.yml" \) -print0 |
          while IFS= read -r -d "" file; do
            echo "Validating $file"
            python -c "import yaml, json, sys; json.dump(yaml.safe_load(open(sys.argv[1])), sys.stdout)" "$file" |
              jsonschema -i - https://bavix.github.io/gripmock/schema/stub.json
          done
```

### GitLab CI

```yaml
validate_stubs:
  image: python:3.9
  script:
    - pip install jsonschema pyyaml
    - |
      for file in $(find . -name "*.json" -path "*/stubs/*"); do
        echo "Validating $file"
        jsonschema -i "$file" https://bavix.github.io/gripmock/schema/stub.json
      done
    - |
      for file in $(find . -name "*.yaml" -path "*/stubs/*" -o -name "*.yml" -path "*/stubs/*"); do
        echo "Validating $file"
        python -c "import yaml, json; json.dumps(yaml.safe_load(open('$file')))" | jsonschema -i - https://bavix.github.io/gripmock/schema/stub.json
      done
```

## Common Validation Errors

### Missing Required Fields

A stub with `service` but no `method` and no `output`:

```json
{
  "service": "MyService"
}
```

**Error**: `'method' is a required property`

### Invalid Priority Value

```yaml
priority: "high"  # Should be integer
```

**Error**: `'high' is not of a type(s) 'integer'`

### Invalid Delay Format

```yaml
delay: "2 minutes"  # Invalid format
```

**Error**: `'2 minutes' does not match '^(\\d+(\\.\\d+)?(ms|s|m|h))+$'`

### Invalid Input Matcher

```yaml
input:
  equals: "string"  # Should be object
```

**Error**: `'string' is not of a type(s) 'object'`

## Notes

Schema validation catches structure — a missing `method`, a `code` that is a
string, a matcher key that does not exist. It says nothing about whether a stub
matches the request you expect it to; for that, use
[`POST /api/stubs/search`](../api/stubs/search.md). 