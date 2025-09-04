# GitHub Issue Operator - Complete Design Document with Standards

## Executive Summary

This document outlines the implementation plan for the GitHub Issue Operator, a Kubernetes operator that manages GitHub issues through Custom Resource Definitions (CRDs). Each CRD maps to exactly one GitHub issue, with proper conflict resolution and state synchronization.

## Current State Analysis

### Existing Infrastructure
- **Framework**: Kubebuilder v4.7.1 with Go 1.24.0
- **Domain**: `shahaf.com` 
- **API Group**: `github.shahaf.com/v1alpha1`
- **Resource**: `GithubIssue` (namespaced)
- **Current CRD Spec Fields**:
  - `repo`: string (required) - needs validation
  - `title`: string (required) - needs validation
  - `description`: string (optional) - needs validation
- **Current Status**: Empty status struct - needs comprehensive implementation
- **Controller**: Basic scaffold exists but no business logic implemented

## Coding Standards

### General Principles

1. **Minimal Changes**: Implement with the smallest necessary modifications
   - Avoid redundant code or over-engineering
   - Extend existing structures rather than creating new ones when possible
   - Reuse Kubebuilder generated code patterns

2. **Single Responsibility Principle (SRP)**: Each function should have one clear purpose
   ```go
   // Good: Single responsibility
   func (r *GithubIssueReconciler) fetchGithubIssues(ctx context.Context, repo string) ([]*github.Issue, error)
   func (r *GithubIssueReconciler) findIssueByTitle(issues []*github.Issue, title string) *github.Issue
   func (r *GithubIssueReconciler) updateIssueStatus(ctx context.Context, issue *githubv1alpha1.GithubIssue) error
   
   // Bad: Multiple responsibilities
   func (r *GithubIssueReconciler) handleGithubOperations(ctx context.Context, issue *githubv1alpha1.GithubIssue) error
   ```

3. **Clean Architecture**: Prefer decoupling when functions become complex
   - Split functions when they exceed 50 lines
   - Separate GitHub API operations from Kubernetes operations
   - Create helper functions for common operations

4. **Avoid Code Duplication**: Use helper functions only when justified
   ```go
   // Justified helper function - used in multiple places
   func parseRepoURL(repo string) (owner, name string, err error) {
       parts := strings.Split(repo, "/")
       if len(parts) != 2 {
           return "", "", fmt.Errorf("invalid repo format: %s", repo)
       }
       return parts[0], parts[1], nil
   }
   ```

### Comment Guidelines

1. **No Inline Comments**: Code should be self-explanatory
   ```go
   // Bad
   issueID := *issue.Number // Get the issue ID
   
   // Good
   issueID := *issue.Number
   ```

2. **GoDoc Comments**: Only when truly necessary for exported functions
   ```go
   // NewGithubClient creates a GitHub client with authentication
   func NewGithubClient(ctx context.Context, token string) *github.Client {
       // Implementation
   }
   ```

3. **Code Clarity**: Prefer clear variable names over comments
   ```go
   // Good
   githubIssueTitle := githubIssue.Spec.Title
   existingIssue := findIssueByTitle(allIssues, githubIssueTitle)
   
   // Bad
   title := gi.Spec.Title // title from spec
   issue := find(issues, title) // find existing issue
   ```

### File Organization

```
internal/
├── controller/
│   ├── githubissue_controller.go     # Main reconciliation logic
│   └── githubissue_controller_test.go
├── github/
│   ├── client.go                     # GitHub API operations
│   ├── client_test.go
│   └── types.go                      # GitHub-specific types
└── utils/
    ├── conditions.go                 # Status condition helpers
    └── validation.go                 # Input validation helpers
```

## Testing Strategy

### Progressive Testing Requirements

Every change during development must include:

1. **Lint Checks**: Ensure code quality
   ```bash
   make lint
   golangci-lint run
   ```

2. **Unit Tests**: Verify correctness and prevent regressions
   ```bash
   make test
   go test ./internal/... -v
   ```

### Comprehensive Testing Framework

#### Unit Tests (Using Fake Client)
```go
// File: internal/controller/githubissue_controller_test.go
var _ = Describe("GithubIssue Controller", func() {
    Context("When reconciling a GithubIssue", func() {
        It("Should create a GitHub issue when none exists", func() {
            // Test implementation using fake client
        })
        
        It("Should update description when issue exists", func() {
            // Test implementation
        })
        
        It("Should handle conflicts between multiple CRs", func() {
            // Test implementation
        })
    })
})
```

**Test Coverage Requirements**:
- Controller reconciliation logic: 100%
- GitHub client operations: 90%
- Status condition management: 100%
- Error handling scenarios: 90%

#### Integration Tests (Using GitHub Mock)
```go
// File: internal/github/client_test.go
func TestGithubClient_Integration(t *testing.T) {
    // Setup mock HTTP client
    mockedHTTPClient := mock.NewMockedHTTPClient(
        mock.WithRequestMatch(
            mock.GetReposIssuesByOwnerByRepo,
            []github.Issue{},
        ),
        mock.WithRequestMatch(
            mock.PostReposIssuesByOwnerByRepo,
            github.Issue{
                ID:    github.Int64(123),
                Title: github.Ptr("Test Issue"),
            },
        ),
    )
    
    client := github.NewClient(mockedHTTPClient)
    // Test scenarios
}
```

**Test Scenarios**:
- ✅ Failed attempt to create a GitHub issue
- ✅ Failed attempt to update an issue
- ✅ Create issue when none exists
- ✅ Close issues on CR deletion
- ✅ Handle conflicts between multiple CRs
- ✅ Detect PR associations
- ✅ Handle authentication failures
- ✅ Handle network errors

#### E2E Tests (Using Ginkgo + Real GitHub)
```go
// File: test/e2e/e2e_test.go
var _ = Describe("GithubIssue E2E", func() {
    It("Should manage GitHub issues end-to-end", func() {
        By("Creating a GithubIssue CR")
        // Implementation
        
        By("Verifying GitHub issue creation")
        // Implementation
        
        By("Updating the CR description")
        // Implementation
        
        By("Verifying GitHub issue update")
        // Implementation
        
        By("Deleting the CR")
        // Implementation
        
        By("Verifying GitHub issue closure")
        // Implementation
    })
})
```

## CI/CD Requirements

### GitHub Actions Workflow

Create `.github/workflows/ci.yml`:

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
    - name: Run golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest

  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    - name: Build
      run: make build
    - name: Test compilation
      run: go build ./...

  unit-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    - name: Run unit tests
      run: |
        make test
        go test ./internal/... -v -coverprofile=coverage.out
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

  e2e-tests:
    runs-on: ubuntu-latest
    needs: [lint, build, unit-tests]
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    - name: Create k8s cluster
      uses: helm/kind-action@v1.8.0
      with:
        cluster_name: test-cluster
    - name: Run E2E tests
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_E2E_TOKEN }}
      run: make test-e2e

  docker:
    runs-on: ubuntu-latest
    needs: [lint, build, unit-tests]
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
    - uses: actions/checkout@v4
    - name: Build and push Docker image
      run: |
        make docker-build IMG=ghcr.io/${{ github.repository }}:${{ github.sha }}
        make docker-push IMG=ghcr.io/${{ github.repository }}:${{ github.sha }}
```

### Makefile Enhancements

Add to existing `Makefile`:

```makefile
##@ Testing

.PHONY: test-unit
test-unit: ## Run unit tests
	go test ./internal/... -v -coverprofile=coverage.out

.PHONY: test-e2e
test-e2e: ## Run E2E tests
	cd test/e2e && ginkgo -v

.PHONY: test-integration
test-integration: ## Run integration tests with mocked GitHub
	go test ./internal/github/... -v -tags=integration

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: coverage
coverage: test-unit ## Generate test coverage report
	go tool cover -html=coverage.out -o coverage.html

##@ CI/CD

.PHONY: ci-checks
ci-checks: lint build test-unit ## Run all CI checks locally
	@echo "All CI checks passed"

.PHONY: verify-generate
verify-generate: ## Verify generated code is up to date
	make generate manifests
	git diff --exit-code
```

### Quality Gates

Each PR must pass:

1. **Build Checks**: Code compiles successfully
2. **Lint Checks**: golangci-lint passes with zero issues
3. **Unit Tests**: All unit tests pass with >90% coverage
4. **Integration Tests**: GitHub mock tests pass
5. **E2E Tests**: End-to-end workflow validation
6. **Generated Code**: Verify manifests and generated code are up to date

### Development Workflow

```bash
# Before committing changes
make ci-checks

# During development
make test-unit        # Quick feedback loop
make test-integration # Test GitHub integration
make lint            # Check code quality

# Before pushing
make test-e2e        # Full end-to-end validation
```

## Implementation Plan

### Phase 1: API Design and Enhancement (1 day)

#### 1.1 Enhanced GithubIssueStatus Structure
```go
type GithubIssueStatus struct {
    // Conditions represent the latest available observations
    Conditions []metav1.Condition `json:"conditions,omitempty"`
    
    // IssueID is the GitHub issue number/ID
    IssueID *int `json:"issueID,omitempty"`
    
    // URL is the direct link to the GitHub issue
    URL string `json:"url,omitempty"`
    
    // State represents the current GitHub issue state (open/closed)
    State string `json:"state,omitempty"`
    
    // HasPullRequest indicates if the issue has an associated pull request
    HasPullRequest bool `json:"hasPullRequest,omitempty"`
    
    // LastSyncTime is when the issue was last synchronized
    LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}
```

#### 1.2 CRD-Level Validation
```go
type GithubIssueSpec struct {
    // Repo must be a valid GitHub repository in format "owner/repo"
    // +kubebuilder:validation:Pattern=`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`
    // +kubebuilder:validation:MinLength=3
    // +kubebuilder:validation:MaxLength=100
    Repo string `json:"repo"`
    
    // Title of the GitHub issue
    // +kubebuilder:validation:MinLength=1
    // +kubebuilder:validation:MaxLength=256
    Title string `json:"title"`
    
    // Description of the GitHub issue
    // +kubebuilder:validation:MaxLength=65536
    // +optional
    Description *string `json:"description,omitempty"`
}
```

#### 1.3 Finalizer Implementation
```go
const GithubIssueFinalizer = "github.shahaf.com/finalizer"

func (r *GithubIssueReconciler) addFinalizer(ctx context.Context, issue *githubv1alpha1.GithubIssue) error
func (r *GithubIssueReconciler) removeFinalizer(ctx context.Context, issue *githubv1alpha1.GithubIssue) error
func (r *GithubIssueReconciler) hasFinalizer(issue *githubv1alpha1.GithubIssue) bool
```

### Phase 2: GitHub Integration (2 days)

#### 2.1 GitHub Client Structure
```go
// File: internal/github/client.go
type Client struct {
    client *github.Client
    owner  string
    repo   string
}

func NewClient(token, owner, repo string) *Client
func (c *Client) ListIssues(ctx context.Context) ([]*github.Issue, error)
func (c *Client) CreateIssue(ctx context.Context, title, description string) (*github.Issue, error)
func (c *Client) UpdateIssue(ctx context.Context, issueNumber int, description string) (*github.Issue, error)
func (c *Client) CloseIssue(ctx context.Context, issueNumber int) error
func (c *Client) HasPullRequest(ctx context.Context, issueNumber int) (bool, error)
```

#### 2.2 Authentication Implementation
```go
// File: internal/github/auth.go
func GetTokenFromSecret(ctx context.Context, client client.Client, namespace string) (string, error) {
    secretName := os.Getenv("GITHUB_TOKEN_SECRET_NAME")
    if secretName == "" {
        secretName = "github-token"
    }
    // Implementation
}
```

#### 2.3 GitHub Operations
- **List Issues**: GET `/repos/{owner}/{repo}/issues` - to find existing issues by title
- **Create Issue**: POST `/repos/{owner}/{repo}/issues`
- **Update Issue**: PATCH `/repos/{owner}/{repo}/issues/{issue_number}`
- **Close Issue**: PATCH `/repos/{owner}/{repo}/issues/{issue_number}` with `state: "closed"`
- **Check PR Association**: GET `/repos/{owner}/{repo}/issues/{issue_number}/events` - check for linked PRs

### Phase 3: Controller Implementation (3 days)

#### 3.1 Main Reconciliation Logic
```go
func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch GithubIssue resource
    // 2. Handle deletion (finalizer logic)
    // 3. Parse repository URL
    // 4. Authenticate with GitHub
    // 5. Fetch all GitHub issues
    // 6. Find or create issue
    // 7. Update status
    // 8. Handle conflicts
}
```

#### 3.2 Helper Functions (SRP compliance)
```go
func (r *GithubIssueReconciler) handleDeletion(ctx context.Context, issue *githubv1alpha1.GithubIssue) (ctrl.Result, error)
func (r *GithubIssueReconciler) ensureGithubIssue(ctx context.Context, ghClient *github.Client, issue *githubv1alpha1.GithubIssue) error
func (r *GithubIssueReconciler) updateStatus(ctx context.Context, issue *githubv1alpha1.GithubIssue, ghIssue *github.Issue) error
func (r *GithubIssueReconciler) detectConflicts(ctx context.Context, issue *githubv1alpha1.GithubIssue) error
```

#### 3.3 Corrected Reconciliation Logic Flow
```
1. Fetch GithubIssue resource from Kubernetes
2. Parse repository URL (owner/repo) 
3. Authenticate with GitHub API using namespace secret
4. Fetch ALL GitHub issues from the repository
5. Find issue with exact matching title
6. If issue doesn't exist:
   - Create new GitHub issue with title and description
   - Get created issue details
7. If issue exists:
   - Compare description and update if different
   - Get current issue details
8. Check for associated pull requests
9. Update Kubernetes resource status with real GitHub issue state
10. Handle conflicts (multiple CRs with same title)
```

#### 3.4 Error Handling Strategy
- **Authentication Errors**: Set error condition, requeue after 30 seconds
- **Repository Not Found**: Set failed condition, don't requeue
- **Permission Denied**: Set error condition, requeue after 5 minutes
- **Network Errors**: Retry with exponential backoff (max 3 retries)
- **Conflict Detection**: Set error condition for duplicate title conflicts

#### 3.5 Status Conditions
```go
const (
    ConditionTypeReady          = "Ready"
    ConditionTypeSynced         = "Synced" 
    ConditionTypeError          = "Error"
    ConditionTypeConflict       = "Conflict"
)
```

### Phase 4: Testing Implementation (2 days)

#### 4.1 Unit Test Structure
```go
// File: internal/controller/githubissue_controller_test.go
var _ = Describe("GithubIssue Controller", func() {
    var (
        ctx       context.Context
        k8sClient client.Client
        reconciler *GithubIssueReconciler
    )
    
    BeforeEach(func() {
        // Setup fake client and reconciler
    })
    
    Context("Creating GitHub Issues", func() {
        It("Should create issue when none exists", func() {})
        It("Should update description when issue exists", func() {})
        It("Should handle authentication failures", func() {})
    })
    
    Context("Conflict Resolution", func() {
        It("Should detect title conflicts", func() {})
        It("Should set appropriate error conditions", func() {})
    })
    
    Context("Deletion Handling", func() {
        It("Should close GitHub issue on CR deletion", func() {})
        It("Should remove finalizer after closing issue", func() {})
    })
})
```

#### 4.2 Integration Tests with Mock
```go
// File: internal/github/client_test.go
func TestGithubClient_CreateIssue(t *testing.T) {
    mockedHTTPClient := mock.NewMockedHTTPClient(
        mock.WithRequestMatch(
            mock.PostReposIssuesByOwnerByRepo,
            github.Issue{
                ID:    github.Int64(123),
                Title: github.Ptr("Test Issue"),
            },
        ),
    )
    // Test implementation
}
```

### Phase 5: CI/CD Setup (1 day)

#### 5.1 GitHub Actions Configuration
- Create comprehensive CI workflow
- Add quality gates for all checks
- Configure E2E testing with kind cluster
- Set up code coverage reporting

#### 5.2 Makefile Enhancement
- Add testing targets
- Create CI check aggregation
- Add coverage reporting

### Phase 6: Documentation and Deployment (1 day)

#### 6.1 Minimal Documentation
Single `README.md` with:
- Installation via Helm
- Basic usage example
- Configuration options

#### 6.2 Helm Chart
```
charts/github-issue-operator/
├── Chart.yaml
├── values.yaml
└── templates/
    ├── deployment.yaml
    ├── rbac.yaml
    └── configmap.yaml
```

## Configuration and Security

### Environment Configuration
```go
type Config struct {
    GitHubTokenSecretName  string // Default: "github-token"
    ResyncPeriod          time.Duration // 1 minute for forced reconciliation
}
```

### RBAC Enhancement
Add user management roles:

**Viewer Role** (`githubissue_viewer_role.yaml`):
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: githubissue-viewer
rules:
- apiGroups: ["github.shahaf.com"]
  resources: ["githubissues"]
  verbs: ["get", "list", "watch"]
```

**Editor Role** (`githubissue_editor_role.yaml`):
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: githubissue-editor
rules:
- apiGroups: ["github.shahaf.com"]
  resources: ["githubissues"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

### Security Implementation
**GitHub Token Storage**: Read from namespace-scoped secret via environment variable
```go
tokenSecretName := os.Getenv("GITHUB_TOKEN_SECRET_NAME")
if tokenSecretName == "" {
    tokenSecretName = "github-token"
}
```

**Secret Format** (per namespace):
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-token
  namespace: <target-namespace>
type: Opaque
data:
  token: <base64-encoded-github-token>
```

## Advanced Features

### Issue Lifecycle Management
- **Creation**: New GithubIssue CR → Create GitHub issue
- **Updates**: Spec changes → Update GitHub issue description
- **Deletion**: CR deletion → Close GitHub issue → Remove finalizer

### State Synchronization and Conflict Resolution
**Resync Period**: 1 minute forced reconciliation to detect external changes

**External Modification Handling**: When GitHub issue is modified externally, controller will:
1. Detect changes during reconciliation
2. Force update GitHub issue to match CR spec (controller is source of truth)
3. Update status with current state

**Conflict Resolution**: When multiple CRs request the same GitHub issue (same repo + title):
- First CR succeeds and claims the issue
- Subsequent CRs fail with `Conflict` condition and descriptive error message
- Status message: "GitHub issue with title 'X' already managed by GithubIssue 'Y' in namespace 'Z'"

### Single Issue Per CR Design
**Design Decision**: Each CR manages exactly one GitHub issue
- **Benefit**: Simple, predictable, easy to manage
- **Multi-issue Support**: Users create multiple CRs as needed
- **Conflict Prevention**: Built-in detection and reporting

## Dependencies

### Required Go Modules
```go
// Add to go.mod
github.com/google/go-github/v57 v57.0.0
golang.org/x/oauth2 v0.15.0
github.com/migueleliasweb/go-github-mock v1.4.0 // for testing
```

## Success Criteria

### Code Quality
- ✅ All functions follow SRP
- ✅ Zero code duplication (justified helpers only)
- ✅ Clear, readable code without unnecessary comments
- ✅ Minimal, focused changes

### Testing Coverage
- ✅ Unit tests: >90% coverage
- ✅ Integration tests: All GitHub operations
- ✅ E2E tests: Complete workflow validation
- ✅ All tests pass in CI

### CI/CD Pipeline
- ✅ Automated build checks
- ✅ Lint validation
- ✅ Unit and integration tests
- ✅ E2E testing with real Kubernetes cluster
- ✅ Quality gates prevent broken code from merging

### Functional Requirements
- ✅ Create GitHub issues from Kubernetes resources
- ✅ Update GitHub issue descriptions when CR changes
- ✅ Close GitHub issues when CR is deleted
- ✅ Detect and report conflicts between CRs
- ✅ Track PR associations in status

### Non-Functional Requirements
- ✅ CRD-level validation prevents invalid resources
- ✅ Secure token management via namespace secrets
- ✅ Comprehensive testing with mock and real GitHub APIs
- ✅ RBAC roles for user management

### Documentation Requirements
- ✅ Minimal, concise README.md
- ✅ Helm chart for easy deployment

## Timeline Estimate

**Total Duration**: 10 working days

- **Phase 1**: API Enhancement (1 day)
- **Phase 2**: GitHub Integration (2 days)
- **Phase 3**: Controller Implementation (3 days)
- **Phase 4**: Testing Implementation (2 days)
- **Phase 5**: CI/CD Setup (1 day)
- **Phase 6**: Documentation & Helm Chart (1 day)

This comprehensive design ensures high code quality, thorough testing, and reliable CI/CD processes while maintaining simplicity and avoiding over-engineering.
