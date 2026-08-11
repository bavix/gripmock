# Contributing to GripMock

**Languages:** English | [简体中文](CONTRIBUTING.zh-CN.md) | [日本語](CONTRIBUTING.ja-JP.md) | [Deutsch](CONTRIBUTING.de.md) | [Español](CONTRIBUTING.es.md)

## Getting Started

1. **Fork the repository** and clone your fork locally
2. **Set up your development environment**:
   - Install [grpctestify](https://github.com/gripmock/grpctestify-rust) for integration tests (see [grpctestify documentation](https://gripmock.github.io/grpctestify-rust/) for installation instructions)
   - Ensure you have Go installed and configured

### ConnectRPC Testing

**HTTP Client tests** (`.http` files) for the ConnectRPC server live in `examples/projects/*/connectrpc-tests/`.

**Running with httpyac:**
```bash
npx httpyac run examples/projects/greeter/connectrpc-tests/ --all
```

You can also open `.http` files in JetBrains IDEs (GoLand, IntelliJ) and click the run icon next to each request.

## Testing Requirements

### 1. gRPC server changes require integration tests

Anything that changes gRPC server behaviour needs an integration test written with grpctestify, in `.gctf` format.

Integration tests are located in the `examples/` directory. Example `.gctf` file:

```
--- ENDPOINT ---
helloworld.Greeter/SayHello

--- REQUEST ---
{"name": "Alex"}

--- RESPONSE ---
{"message": "Hello, Alex!"}
```

**Where to place tests:**
- Integration tests: `examples/projects/*/case_*.gctf`
- Unit tests: `internal/app/*_internal_test.go`

### 2. Every PR includes tests

Bugfixes and new features both need them.

### 3. Running tests locally

```bash
make test    # Unit tests
make lint    # Linter
```

Integration tests need the server running in a separate terminal:

```bash
go run main.go examples -s examples
grpctestify examples/
```

## Backward Compatibility

All changes must be backward compatible unless a breaking change has been discussed and approved through an issue.

### Breaking Changes Process

If you need to introduce a breaking change:

1. **Create an Issue First**: Open an issue with a detailed proposal that includes:
   - Description of the problem you're trying to solve
   - Why the breaking change is necessary
   - Proposed migration path for existing users

2. **Wait for Approval**: Do not implement breaking changes without discussion and approval from maintainers

3. **Provide Migration Guide**: If approved, include clear migration instructions in your PR

## Pull Request Process

### Before Submitting

- [ ] All tests pass locally
- [ ] Code follows the project's style guidelines (`make lint`)
- [ ] Documentation is updated if needed
- [ ] Your branch is up to date with `master`

### PR Description

When creating a PR, please include:
- Description of changes
- Type of change (bug fix, new feature, etc.)
- Testing information (unit tests, integration tests if gRPC server changes)
- Backward compatibility status
- Related issues

## Code Style

- Follow standard Go formatting: `gofmt` and `goimports`
- Run the linter: `make lint`
- Use meaningful variable and function names
- Add comments for exported functions and types
- Place new code in appropriate packages under `internal/`

## Documentation

Update documentation when:
- Adding new features
- Changing existing behavior
- Fixing bugs that affect user workflows

Documentation locations:
- User docs: `docs/guide/`
- Examples: `examples/` directory
- Main README: `README.md`

## Questions?

Check the existing issues and discussions first, then open a new issue with the
`question` label.

- [Project documentation](https://bavix.github.io/gripmock/)
- [grpctestify documentation](https://gripmock.github.io/grpctestify-rust/)
