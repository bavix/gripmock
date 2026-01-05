# Contributing to GripMock

Thank you for your interest in contributing to GripMock! This document provides guidelines for contributing to the project.

## Getting Started

1. **Fork the repository** and clone your fork locally
2. **Set up your development environment**:
   - Install [grpctestify](https://github.com/gripmock/grpctestify) for integration tests (see [grpctestify documentation](https://gripmock.github.io/grpctestify/) for installation instructions)
   - Ensure you have Go installed and configured

## Testing Requirements

### ⚠️ Critical Rules

#### 1. gRPC Server Changes Require Integration Tests

**If you change, add, or fix anything related to the gRPC server functionality, you MUST write integration tests using grpctestify in `.gctf` format.**

Integration tests are located in the `examples/` directory. Example `.gctf` file:

```
--- ENDPOINT ---
helloworld.Greeter/SayHello

--- REQUEST ---
{"name": "Alex"}

--- RESPONSE ---
{"message": "Hello, Alex!"}
```

**Running tests:**
```bash
make test                    # Unit tests
grpctestify examples/  # Integration tests
make lint                    # Linter
```

**Where to place tests:**
- Integration tests: `examples/projects/*/case_*.gctf`
- Unit tests: `internal/app/*_internal_test.go`

#### 2. Every PR Must Include Tests

All Pull Requests must include appropriate tests, especially for bugfixes and new features.

#### 3. Running Tests Locally

Before submitting a PR, ensure all tests pass:

**For integration tests with grpctestify:**
```bash
# Start the server (in a separate terminal)
go run main.go examples -s examples

# Run integration tests
grpctestify examples/
```

**For unit tests:**
```bash
make test
make lint
```

## Backward Compatibility

**All changes MUST be backward compatible** unless explicitly discussed and approved through an issue.

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
- [ ] Your branch is up to date with the main branch

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

- Check existing issues and discussions
- Open a new issue with the `question` label
- Review the [documentation](https://bavix.github.io/gripmock/)

## Additional Resources

- [Project Documentation](https://bavix.github.io/gripmock/)
- [grpctestify Documentation](https://gripmock.github.io/grpctestify/)

Thank you for contributing to GripMock! 🚀
