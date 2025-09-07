# CI/CD Requirements

> **Navigation**: [Design](../design/DESIGN.md) | [Coding Standards](CODING_STANDARDS.md) | [Testing Strategy](TESTING_STRATEGY.md)

## GitHub Actions Pipeline

**File**: `.github/workflows/ci.yml`

```yaml
name: CI
on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    - uses: golangci/golangci-lint-action@v3

  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    - run: make test
    - uses: codecov/codecov-action@v3

  e2e:
    runs-on: ubuntu-latest
    needs: [lint, test]
    steps:
    - uses: actions/checkout@v4
    - uses: helm/kind-action@v1.8.0
    - run: make test-e2e
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_E2E_TOKEN }}
```

## Makefile Targets

```makefile
.PHONY: test-unit
test-unit:
	go test ./internal/... -v -coverprofile=coverage.out

.PHONY: test-e2e  
test-e2e:
	cd test/e2e && ginkgo -v

.PHONY: ci-checks
ci-checks: lint build test-unit
	@echo "All CI checks passed"
```

## Quality Gates

Each PR must pass:
1. **Lint**: golangci-lint with zero issues
2. **Build**: Code compiles successfully  
3. **Unit Tests**: >90% coverage, all tests pass
4. **E2E Tests**: Complete workflow validation

## Development Workflow

```bash
# Before committing
make ci-checks

# Before pushing  
make test-e2e
```

## Branch Protection

- Require PR reviews (minimum 1)
- Require status checks: `lint`, `test`, `e2e`
- Squash and merge only

## Related Documentation
- [Architecture](../design/ARCHITECTURE.md) | [Coding Standards](CODING_STANDARDS.md) | [Testing Strategy](TESTING_STRATEGY.md)
