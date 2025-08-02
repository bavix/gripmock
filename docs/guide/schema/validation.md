# Schema Validation

So you've written some stub definitions and want to make sure they're correct? This guide shows you how to validate them using our JSON Schema. Think of it as a safety net that catches problems before they cause issues in your tests.

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

3. Get real-time validation and auto-completion



## CI/CD Integration

### GitHub Actions

```yaml
name: Validate Stubs

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Python
        uses: actions/setup-python@v4
        with:
          python-version: '3.9'
          
      - name: Install dependencies
        run: |
          pip install jsonschema pyyaml
          
      - name: Validate JSON stubs
        run: |
          for file in $(find . -name "*.json" -path "*/stubs/*"); do
            echo "Validating $file"
            jsonschema -i "$file" https://bavix.github.io/gripmock/schema/stub.json
          done
          
      - name: Validate YAML stubs
        run: |
          for file in $(find . -name "*.yaml" -path "*/stubs/*" -o -name "*.yml" -path "*/stubs/*"); do
            echo "Validating $file"
            python -c "import yaml, json; json.dumps(yaml.safe_load(open('$file')))" | jsonschema -i - https://bavix.github.io/gripmock/schema/stub.json
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

```json
{
  "service": "MyService"
  // Missing "method" and "output" - required fields
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

## Best Practices

Here are some tips to make validation work better for you:

1. **Always validate** your stubs before deployment - it's like proofreading your code
2. **Use IDE integration** for real-time feedback - catch errors as you type
3. **Set up CI/CD validation** to catch errors early - let the computer do the boring work
4. **Test with real data** to ensure your stubs work as expected - validation catches syntax errors, but you need to test logic
5. **Keep schemas updated** when adding new features - the schema should match what you're actually using 