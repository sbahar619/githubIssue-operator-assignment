# Secret Management Implementation Plan

## Overview

Implement secure GitHub token retrieval from Kubernetes secrets before GitHub API calls.

**Dependencies**: Phase 1 (API structure) must be complete  
**Duration**: 1 day (8 hours)

## File Structure

```
internal/auth/
├── secrets.go          # Token retrieval logic
└── secrets_test.go     # Unit tests
```

## Step 1: Token Retrieval Infrastructure

**Purpose**: Securely fetch GitHub tokens from namespace-scoped secrets.

**Why use TokenRetriever struct instead of a simple function?**
- **Controller pattern**: Controllers store services as fields for long-lived usage
- **Dependency injection**: Kubernetes client provided once during startup, reused many times
- **Clean reconcile code**: No need to pass client parameter on every call
- **Future extensibility**: Easy to add features like caching or logging later

### File: `internal/auth/secrets.go`

```go
package auth

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultSecretName = "github-token-secret"

// TokenRetriever handles GitHub token retrieval from Kubernetes secrets
type TokenRetriever struct {
	client client.Client
}

func NewTokenRetriever(client client.Client) *TokenRetriever {
	return &TokenRetriever{client: client}
}

// GetGitHubToken retrieves GitHub token from secret in the given namespace
func (tr *TokenRetriever) GetGitHubToken(ctx context.Context, namespace string) (string, error) {
	secretName := tr.getSecretName()
	
	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}
	
	if err := tr.client.Get(ctx, key, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretName, err)
	}
	
	tokenBytes, exists := secret.Data["token"]
	if !exists {
		return "", fmt.Errorf("token key not found in secret %s/%s", namespace, secretName)
	}
	
	token := string(tokenBytes)
	if token == "" {
		return "", fmt.Errorf("empty token in secret %s/%s", namespace, secretName)
	}
	
	return token, nil
}

func (tr *TokenRetriever) getSecretName() string {
	if name := os.Getenv("GITHUB_TOKEN_SECRET_NAME"); name != "" {
		return name
	}
	return DefaultSecretName
}
```

**Why**: Provides secure, namespace-isolated token retrieval with configurable secret names.

## Step 2: Controller Integration

**Purpose**: Connect token retrieval to the reconciliation loop.

### Update: `internal/controller/githubissue_controller.go`

**Import Addition** (line 28, after existing imports):
```go
"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
```

**Struct Update** (lines 32-36, modify existing struct):
```go
type GithubIssueReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	TokenRetriever *auth.TokenRetriever  // ADD this field
}
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

	// Get GitHub token from secret
	token, err := r.TokenRetriever.GetGitHubToken(ctx, githubIssue.Namespace)
	if err != nil {
		log.Error(err, "failed to retrieve GitHub token")
		// TODO: Set error condition in status (Phase 4)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	log.Info("Successfully retrieved GitHub token", "namespace", githubIssue.Namespace)
	// TODO: Use token for GitHub API calls (Phase 3)

	return ctrl.Result{}, nil
}
```

**Helper Method** (add after line 69, end of file):
```go
func (r *GithubIssueReconciler) getGitHubToken(ctx context.Context, namespace string) (string, error) {
	return r.TokenRetriever.GetGitHubToken(ctx, namespace)
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

### Update: `cmd/main.go`

**Import Addition** (line 41, after existing imports):
```go
"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
```

**Token Retriever Creation** (add after line 204, after manager creation):
```go
// Create token retriever for GitHub authentication
tokenRetriever := auth.NewTokenRetriever(mgr.GetClient())
```

**Controller Setup Update** (lines 206-212, replace existing controller creation):
```go
if err := (&controller.GithubIssueReconciler{
	Client:         mgr.GetClient(),
	Scheme:         mgr.GetScheme(),
	TokenRetriever: tokenRetriever,  // ADD this field
}).SetupWithManager(mgr); err != nil {
	setupLog.Error(err, "unable to create controller", "controller", "GithubIssue")
	os.Exit(1)
}
```

**Why**: Makes token retrieval available in the reconcile loop for GitHub API calls.

## Step 3: RBAC Permissions

**Purpose**: Allow controller to read secrets from namespaces.

### Update: `internal/controller/githubissue_controller.go`

**RBAC Marker Addition** (add after line 40, after existing RBAC markers):
```go
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
```

**Why**: Controller needs permission to read GitHub token secrets.

## Step 4: Code Cleanup (Optional)

**Purpose**: Remove redundant auto-generated comments for cleaner, production-ready code.

### Update: `cmd/main.go`

**Remove verbose auto-generated comments** (lines 25-27):
```go
// DELETE these lines:
// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
// to ensure that exec-entrypoint and run can make use of them.
```

**Remove detailed scaffold comments** (lines 137-140):
```go
// DELETE these lines:
// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
// More info:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/server
// - https://book.kubebuilder.io/reference/metrics.html
```

**Remove production TODO comments** (lines 159-162):
```go
// DELETE these lines:
// TODO(user): If you enable certManager, uncomment the following lines:
// - [METRICS-WITH-CERTS] at config/default/kustomization.yaml to generate and use certificates
// managed by cert-manager for the metrics server.
// - [PROMETHEUS-WITH-CERTS] at config/prometheus/kustomization.yaml for TLS certification.
```

**Remove verbose LeaderElection comments** (lines 189-199):
```go
// DELETE these lines:
// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
// when the Manager ends. This requires the binary to immediately end when the
// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
// speeds up voluntary leader transitions as the new leader don't have to wait
// LeaseDuration time first.
//
// In the default scaffold provided, the program ends immediately after
// the manager stops, so would be fine to enable this option. However,
// if you are doing or is intended to do any operation such as perform cleanups
// after the manager stops then its usage might be unsafe.
```

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

## Step 5: Unit Tests

**Purpose**: Verify token retrieval works correctly with all scenarios.

### File: `internal/auth/secrets_test.go`

```go
package auth_test

import (
	"context"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/metav1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
)

func TestAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Suite")
}

var _ = Describe("TokenRetriever", func() {
	var (
		ctx            context.Context
		tokenRetriever *auth.TokenRetriever
		testNamespace  = "test-namespace"
		testToken      = "github_pat_test_token_123"
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("GetGitHubToken", func() {
		It("should retrieve token from existing secret", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "github-token-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"token": []byte(testToken),
				},
			}
			
			fakeClient := fake.NewClientBuilder().WithObjects(secret).Build()
			tokenRetriever = auth.NewTokenRetriever(fakeClient)
			
			token, err := tokenRetriever.GetGitHubToken(ctx, testNamespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(token).To(Equal(testToken))
		})

		It("should handle missing secret", func() {
			fakeClient := fake.NewClientBuilder().Build()
			tokenRetriever = auth.NewTokenRetriever(fakeClient)
			
			_, err := tokenRetriever.GetGitHubToken(ctx, testNamespace)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get secret"))
		})

		It("should handle missing token key", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "github-token-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"wrong-key": []byte("value"),
				},
			}
			
			fakeClient := fake.NewClientBuilder().WithObjects(secret).Build()
			tokenRetriever = auth.NewTokenRetriever(fakeClient)
			
			_, err := tokenRetriever.GetGitHubToken(ctx, testNamespace)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("token key not found"))
		})

		It("should handle empty token", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "github-token-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"token": []byte(""),
				},
			}
			
			fakeClient := fake.NewClientBuilder().WithObjects(secret).Build()
			tokenRetriever = auth.NewTokenRetriever(fakeClient)
			
			_, err := tokenRetriever.GetGitHubToken(ctx, testNamespace)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty token"))
		})

		It("should use custom secret name from environment", func() {
			os.Setenv("GITHUB_TOKEN_SECRET_NAME", "custom-secret")
			defer os.Unsetenv("GITHUB_TOKEN_SECRET_NAME")
			
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "custom-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"token": []byte(testToken),
				},
			}
			
			fakeClient := fake.NewClientBuilder().WithObjects(secret).Build()
			tokenRetriever = auth.NewTokenRetriever(fakeClient)
			
			token, err := tokenRetriever.GetGitHubToken(ctx, testNamespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(token).To(Equal(testToken))
		})
	})
})
```

**Why**: Comprehensive test coverage ensures reliability before GitHub API integration.

## Completion Criteria

### Verification Commands
```bash
# Run tests with coverage
go test ./internal/auth/... -v -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total

# Verify RBAC generation
make manifests
grep -A 5 "secrets" config/rbac/role.yaml

# Lint check
make lint
```

### Success Checklist
- ✅ Token retrieval works with valid secrets
- ✅ Error handling for missing/invalid secrets
- ✅ Environment configuration working
- ✅ Unit tests achieve 90%+ coverage
- ✅ RBAC permissions generated correctly
- ✅ Controller integration ready for Phase 3

## Secret Format Reference

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-token-secret
  namespace: target-namespace
type: Opaque
data:
  token: <base64-encoded-github-personal-access-token>
```

**Environment Variables**:
- `GITHUB_TOKEN_SECRET_NAME`: Secret name (default: "github-token-secret")

This implementation provides secure, testable GitHub token management ready for Phase 3 GitHub API integration.