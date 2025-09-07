# Token Retrieval Implementation

> **Context**: [Authentication Design](../design/AUTH_DESIGN.md) | [Design](../../DESIGN.md)

## Overview

Implement simple GitHub token retrieval from environment variables for controller authentication.

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

func GetGitHubToken() (string, error) {
	token := os.Getenv(GitHubTokenEnvVar)
	if token == "" {
		return "", fmt.Errorf("environment variable %s not configured", GitHubTokenEnvVar)
	}
	return token, nil
}
```

**Why**: Provides simple, secure token access following Kubernetes best practices.

## Step 2: Controller Integration

**Purpose**: Integrate token retrieval into the reconciliation flow.

### Reconciliation Flow Integration

The auth logic integrates at **Step 3** of the reconciliation flow:

1. **Fetch CR**: Get GithubIssue resource from Kubernetes API
2. **Handle Deletion**: Check for deletion timestamp, run finalizer logic  
3. **Authenticate**: Retrieve GitHub token from environment variable ← **Auth integration point**
4. **Create GitHub Client**: Initialize GitHub client with token
5. **Sync Issue State**: Fetch/create/update GitHub issue based on CR spec
6. **Update Status**: Set CR status with GitHub issue details
7. **Handle Conflicts**: Detect and report conflicts with other CRs

### Controller Code Integration

**Constants Addition** (add after package declaration):
```go
package controller

import (...)

// Condition constants for authentication
const (
	ConditionTypeReady          = "Ready"
	ReasonAuthenticationFailed  = "AuthenticationFailed"
)
```

**Import Addition**:
```go
"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
```

**Reconcile Method Integration**:
```go
func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Step 1: Fetch CR
	var githubIssue githubv1alpha1.GithubIssue
	if err := r.Get(ctx, req.NamespacedName, &githubIssue); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Starting reconciliation", "githubissue", req.NamespacedName)

	// Step 2: Handle deletion (TODO: Phase 5)
	
	// Step 3: Authenticate
	_, err := auth.GetGitHubToken()
	if err != nil {
		log.Error(err, "Failed to retrieve GitHub token", "githubissue", req.NamespacedName)
		// Set error condition in CR status
		meta.SetStatusCondition(&githubIssue.Status.Conditions, metav1.Condition{
			Type:    ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonAuthenticationFailed,
			Message: "GitHub token not available: " + err.Error(),
		})
		return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
	}

	log.Info("Successfully retrieved GitHub token")

	// Step 4: Create GitHub client with token (TODO: Phase 3)
	// Steps 5-7: Continue with GitHub operations (TODO: Phase 3-4)
	
	return ctrl.Result{}, nil
}
```

**Import Update** (add to imports section):
```go
import (
	"context"
	"time"  // For RequeueAfter

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/api/meta"        // For meta.SetStatusCondition
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"  // For metav1.Condition
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
package auth

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Suite")
}

var _ = Describe("GetGitHubToken", func() {
	const testToken = "test_token"
	AfterEach(func() {
		Expect(os.Unsetenv(GitHubTokenEnvVar)).To(Succeed())
	})

	It("should return token when set", func() {
		Expect(os.Setenv(GitHubTokenEnvVar, testToken)).To(Succeed())
		token, err := GetGitHubToken()
		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(Equal(testToken))
	})

	It("should error when not set", func() {
		_, err := GetGitHubToken()
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
