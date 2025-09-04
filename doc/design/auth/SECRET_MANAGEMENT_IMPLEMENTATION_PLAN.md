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

// NewTokenRetriever creates a new token retriever instance
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

// getSecretName returns the secret name from environment or default
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

```go
// Add to imports
import (
	"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
)

// Add to GithubIssueReconciler struct
type GithubIssueReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	TokenRetriever *auth.TokenRetriever  // NEW
}

// Add helper method
func (r *GithubIssueReconciler) getGitHubToken(ctx context.Context, namespace string) (string, error) {
	return r.TokenRetriever.GetGitHubToken(ctx, namespace)
}
```

### Update: `cmd/main.go`

```go
// Add to imports
import (
	"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
)

// Add before controller setup
func main() {
	// ... existing setup ...
	
	// Create token retriever
	tokenRetriever := auth.NewTokenRetriever(mgr.GetClient())
	
	// Enhanced controller setup
	if err := (&controller.GithubIssueReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		TokenRetriever: tokenRetriever,  // NEW
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GithubIssue")
		os.Exit(1)
	}
}
```

**Why**: Makes token retrieval available in the reconcile loop for GitHub API calls.

## Step 3: RBAC Permissions

**Purpose**: Allow controller to read secrets from namespaces.

### Update: `internal/controller/githubissue_controller.go`

```go
// Add RBAC marker
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
```

**Why**: Controller needs permission to read GitHub token secrets.

## Step 4: Unit Tests

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

**Why**: Ensures all token retrieval scenarios work correctly before GitHub integration.

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