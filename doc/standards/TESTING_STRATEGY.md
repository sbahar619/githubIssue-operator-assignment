# Testing Strategy

> **Navigation**: [Design](../design/DESIGN.md) | [Coding Standards](CODING_STANDARDS.md) | [CI/CD Requirements](CICD_REQUIREMENTS.md)

## Requirements

Every change must include:
1. **Lint checks**: `make lint`
2. **Unit tests**: `make test`

## Test Types

### Unit Tests
- **Coverage**: >90% for business logic
- **Framework**: Ginkgo + Gomega
- **Clients**: Fake Kubernetes client + GitHub mock client
- **GitHub Library**: `go-github-mock`

```go
// Controller tests with fake client
var _ = Describe("GithubIssue Controller", func() {
    It("Should create GitHub issue when none exists", func() {
        // Test with fake Kubernetes client
    })
})

// GitHub operations with mock client
mockedHTTPClient := mock.NewMockedHTTPClient(
    mock.WithRequestMatch(mock.PostReposIssuesByOwnerByRepo, github.Issue{ID: github.Int64(123)}),
)
```

### Testing Scope

**Test**: Our business logic, GitHub integration, error handling  
**Don't Test**: CRD validation (Kubernetes handles), framework code (upstream tested)

### E2E Tests
- **Framework**: Ginkgo + real Kubernetes cluster + real GitHub API
- **Coverage**: Complete workflows (create → update → delete)
- **Location**: `test/e2e/`

#### GitHub Client Functions Requiring E2E Testing
Due to `go-github-mock` limitations, these functions cannot be properly unit tested:
- `UpdateIssue` - No mock constant for PATCH operations
- `CloseIssue` - No mock constant for PATCH operations  
- `HasPullRequest` - No mock constant for individual issue GET
- `handleError` - Cannot simulate GitHub API error responses
- Error handling paths in `ListIssues`, `GetIssueByTitle`, `CreateIssue`

## Commands

```bash
# Unit tests
make test

# E2E tests
make test-e2e

# All tests
make ci-checks
```

