---
title: gRPC Testify
description: Declarative gRPC Testing Framework
---

# gRPC Testify 🛠️  
[![GitHub Repo](https://img.shields.io/badge/GitHub-Repo-blue?logo=github)](https://github.com/gripmock/grpctestify)

**Declarative testing for gRPC services** using simple `.gctf` configuration files.

## Features
- 🔄 PLAINTEXT communication only
- 🎯 Request/response validation
- 🚦 Error code assertions
- 📄 Human-readable reports
- 📂 Recursive directory processing

## Installation

### Recommended Installation (macOS/Linux)
```bash
brew tap gripmock/tap
brew install grpctestify
```

In this case, `grpctestify` will automatically install `grpcurl` and `jq` as dependencies.

### Manual Installation

#### Install Dependencies
```bash
# macOS
brew install grpcurl jq

# Ubuntu/Debian
sudo apt install -y grpcurl jq

# Verify installation
grpcurl --version
jq --version
```

#### Download the Script
Use `curl` or `wget` to download the `grpctestify.sh` script from the latest release:
```bash
# Using curl
curl -LO https://github.com/gripmock/grpctestify/releases/latest/download/grpctestify.sh

# Using wget
wget https://github.com/gripmock/grpctestify/releases/latest/download/grpctestify.sh
```

#### Make the Script Executable
After downloading, make the script executable:
```bash
chmod +x grpctestify.sh
```

#### Move the Script to a Directory in Your PATH (Optional)
For easier access, move the script to a directory in your `PATH`:
```bash
sudo mv grpctestify.sh /usr/local/bin/grpctestify
```

#### Verify Installation
Check that the script is working correctly:
```bash
grpctestify --version
```

## Test File Format
```php
--- ADDRESS ---
localhost:50051

--- ENDPOINT ---
package.service/Method

--- REQUEST ---
{
  "key": "value"
}

--- RESPONSE ---
{
  "result": "OK"
}

--- ERROR ---
```

## Key Concepts
### 1. Address Handling
- **Default address**: `localhost:4770`
- Override via test file:
  ```yaml
  --- ADDRESS ---
  localhost:50051
  ```
- Override via environment (bash):
  ```bash
  export DEFAULT_ADDRESS=localhost:50051
  ```

### 2. Response/Expectation Rules
- **Exactly one** of `RESPONSE` or `ERROR` must be present
- `RESPONSE` validates successful responses (status code 0)
- `ERROR` validates gRPC error codes/messages (non-zero status)

### 3. Communication Protocol
- **Only PLAINTEXT** supported (no TLS)
- Uses `grpcurl -plaintext` for all requests

## Command-Line Options
```bash
$ ./grpctestify.sh --help

▶ gRPC Testify v0.0.3 - gRPC Server Testing Tool
 INF Usage: ./grpctestify.sh [options] <test_file_or_directory>
 INF Options:
    --no-color   Disable colored output
    --verbose    Show debug output (request/response details)
    --version    Print version
    -h, --help   Show help
```

## Practical Examples
### 1. Calculator Service Tests
**math_add.gctf**
```yaml
--- ENDPOINT ---
calculator.Math/Add

--- REQUEST ---
{
    "a": 5,
    "b": 3
}

--- RESPONSE ---
{
    "result": 8
}
```

**math_divide_by_zero.gctf**
```yaml
--- ENDPOINT ---
calculator.Math/Divide

--- REQUEST ---
{
    "a": 10,
    "b": 0
}

--- ERROR ---
{
    "code": 3,    # INVALID_ARGUMENT (see status codes reference)
    "message": "division by zero"
}
```

### 2. User Management Tests
**user_create.gctf**
```yaml
--- ADDRESS ---
localhost:50052

--- ENDPOINT ---
user.Manager/Create

--- REQUEST ---
{
    "name": "Alice",
    "email": "alice@example.com"
}

--- RESPONSE ---
{
    "id": 123,
    "status": "created"
}
```

**user_invalid_email.gctf**
```yaml
--- ENDPOINT ---
user.Manager/Create

--- REQUEST ---
{
    "name": "Bob",
    "email": "invalid-email"
}

--- ERROR ---
{
    "code": 16,   # UNAUTHENTICATED (example code)
    "message": "invalid email format"
}
```

## CI/CD Integration
### GitHub Actions Example
```yaml
name: gRPC Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install dependencies
        run: |
          sudo apt update
          sudo apt install -y grpcurl jq
      - name: Run tests
        run: |
          curl -sSL https://raw.githubusercontent.com/gripmock/grpctestify/master/grpctestify.sh -o grpctestify.sh
          chmod +x grpctestify.sh
          ./grpctestify.sh tests/
```

## Best Practices
1. **Test Organization:**
```
tests/
├── math/
│   ├── add_valid.gctf
│   └── divide_by_zero.gctf
└── user/
    ├── create_success.gctf
    └── invalid_email.gctf
```

2. **Validation Rules:**
- Use `jq`-compatible JSON in all sections
- Match error codes from [gRPC status codes](https://grpc.github.io/grpc/core/md_doc_statuscodes.html)
- Keep test files focused (one operation per test)

## Limitations
- No TLS support (PLAINTEXT only)
- Requires valid JSON formatting
- Error messages must exactly match server responses

## Advanced Usage
### Verbose Output Example
```bash
$ ./grpctestify.sh --verbose math_add.gctf
ℹ️ Starting test suite
🔍 Processing file: math_add.gctf
 INF Configuration:
  ADDRESS: localhost:4770
  ENDPOINT: calculator.Math/Add
  REQUEST: {"a":5,"b":3}
  RESPONSE: {"result":8}
🔍 Executing gRPC request to localhost:4770...
✅ TEST PASSED: math_add
```

### Directory Execution
```bash
$ ./grpctestify.sh tests/
...
✅ TEST PASSED: add
...
✅ TEST PASSED: divide_by_zero
...
✅ TEST PASSED: create_success
...
✅ TEST PASSED: invalid_email
```

## Editor Support
Install the [VS Code Extension](https://marketplace.visualstudio.com/items?itemName=gripmock.grpctestify)  
[GitHub Repo for Extension](https://github.com/gripmock/grpctestify-vscode)  
Features:
- Syntax highlighting
- Snippet auto-completion
- Section folding
- Real-time validation
