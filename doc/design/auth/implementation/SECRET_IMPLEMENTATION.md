# Secret Management Implementation

> **Context**: [Authentication Design](../design/AUTH_DESIGN.md) | [Design](../../DESIGN.md)

## Overview

Implement secure GitHub token retrieval from environment variables, with tokens stored in Kubernetes secrets and mounted as environment variables in the controller pod.

## File Structure

```
internal/auth/
├── token.go           # Environment variable token retrieval
└── token_test.go      # Unit tests
config/manager/
└── manager.yaml       # Environment variable configuration
```

## Step 1: Token Retrieval from Environment

**Purpose**: Read GitHub token from environment variable mounted from Kubernetes secret.

**Why this approach?**
- **Follows instruction**: "use it in the code by reading an env variable"
- **Standard pattern**: How most Kubernetes operators handle secrets
- **Simple and secure**: No Kubernetes client complexity
- **No RBAC needed**: No secret read permissions required

### File: `internal/auth/token.go`

```go
package auth

import (
	"fmt"
	"os"
)

const GitHubTokenEnvVar = "GITHUB_TOKEN"

// GetGitHubToken retrieves GitHub token from environment variable
func GetGitHubToken() (string, error) {
	token := os.Getenv(GitHubTokenEnvVar)
	if token == "" {
		return "", fmt.Errorf("environment variable %s not set", GitHubTokenEnvVar)
	}
	return token, nil
}

// IsTokenConfigured checks if GitHub token is available
func IsTokenConfigured() bool {
	return os.Getenv(GitHubTokenEnvVar) != ""
}
```

**Why**: Provides simple, secure token access following Kubernetes best practices.

## Step 2: Controller Integration

**Purpose**: Use token retrieval in the reconciliation loop.

### Update: `internal/controller/githubissue_controller.go`

**Import Addition** (line 28, after existing imports):
```go
"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
```

**Reconcile Method Update** (replace lines 52-57, the empty reconcile logic):
```go
func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the GithubIssue resource
	var githubIssue githubv1alpha1.GithubIssue
	if err := r.Get(ctx, req.NamespacedName, &githubIssue); err != nil {
		log.Error(err, "unable to fetch GithubIssue")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get GitHub token from environment variable
	token, err := auth.GetGitHubToken()
	if err != nil {
		log.Error(err, "failed to retrieve GitHub token")
		// TODO: Set error condition in status (Phase 4)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	log.Info("Successfully retrieved GitHub token")
	// TODO: Use token for GitHub API calls (Phase 3)

	return ctrl.Result{}, nil
}
```

**Import Update** (add to imports section):
```go
import (
	"context"
	"time"  // ADD this import for RequeueAfter

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
)
```

**Why**: Integrates token retrieval into reconcile loop for GitHub API calls in Phase 3.

## Step 3: Environment Variable Configuration

**Purpose**: Mount GitHub token secret as environment variable in controller pod.

### Update: `config/manager/manager.yaml`

**Environment Variable Addition** (add to container spec, around line 65):
```yaml
spec:
  template:
    spec:
      containers:
      - name: manager
        # ... existing configuration ...
        env:
        - name: GITHUB_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-token-secret
              key: token
```

**Why**: Mounts secret as environment variable, making it available to controller code.

## Step 4: Code Cleanup (Optional)

**Purpose**: Remove redundant auto-generated comments for cleaner, production-ready code.

### Update: `internal/controller/githubissue_controller.go`

**Remove scaffold comment** (line 31):
```go
// DELETE this line:
// GithubIssueReconciler reconciles a GithubIssue object
```

**Remove verbose Reconcile comments** (lines 43-51):
```go
// DELETE these lines:
// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GithubIssue object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
```

**Remove placeholder comments** (lines 53, 55):
```go
// DELETE these lines:
_ = logf.FromContext(ctx)
// TODO(user): your logic here
```

**Remove method comment** (line 60):
```go
// DELETE this line:
// SetupWithManager sets up the controller with the Manager.
```

**Why**: Removes unnecessary verbosity while keeping essential functionality clear and professional.

### Unit Tests

**File**: `internal/auth/token_test.go`

```go
var _ = Describe("GetGitHubToken", func() {
	AfterEach(func() {
		os.Unsetenv(auth.GitHubTokenEnvVar)
	})

	It("should return token when set", func() {
		os.Setenv(auth.GitHubTokenEnvVar, "test_token")
		token, err := auth.GetGitHubToken()
		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(Equal("test_token"))
	})

	It("should error when not set", func() {
		_, err := auth.GetGitHubToken()
		Expect(err).To(HaveOccurred())
	})
})
```

## Verification

```bash
# Test
go test ./internal/auth/... -v

# Lint
make lint

# Local test
export GITHUB_TOKEN="test_token"
go run cmd/main.go
```

## Secret Setup

```bash
kubectl create secret generic github-token-secret \
  --from-literal=token=<your-github-token> \
  --namespace=githubissue-operator-system
```