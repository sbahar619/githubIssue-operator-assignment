# Phase 2 Implementation: Issue State Synchronization

> **Navigation**: [Design](../../DESIGN.md) | [Controller Design](CONTROLLER_DESIGN.md) | [Controller Plan](CONTROLLER_PLAN.md)

## Goal
Implement issue lookup using `GetIssueByTitle()` for GitHub issue state synchronization, following single responsibility principle and project coding standards.

## Overview
This phase focuses on adding issue state synchronization logic to the controller's `Reconcile` method. The implementation follows the controller authority pattern where the CR spec drives GitHub state.

## Implementation Details

### Step 1: Add Issue Lookup Function

Add this function to `internal/controller/githubissue_controller.go`:

```go
func (r *GithubIssueReconciler) getIssueByTitle(ctx context.Context, githubClient *github.Client, githubIssue *githubv1alpha1.GithubIssue) (*github.Issue, ctrl.Result) {
	log := logf.FromContext(ctx)
	
	existingIssue, err := githubClient.GetIssueByTitle(ctx, githubIssue.Spec.Title)
	if err != nil {
		if githubError, ok := err.(*github.GitHubError); ok {
			if githubError.IsRetryable {
				return nil, r.handleRetryableGitHubError(ctx, githubIssue, githubError)
			}
			return nil, r.handleNonRetryableGitHubError(ctx, githubIssue, githubError)
		}
		return nil, r.handleUnknownError(ctx, githubIssue, err)
	}

	if existingIssue != nil {
		log.Info("Found existing GitHub issue", 
			"issueID", existingIssue.GetID(),
			"issueNumber", existingIssue.GetNumber(),
			"issueURL", existingIssue.GetHTMLURL())
	} else {
		log.Info("No existing GitHub issue found for title", "title", githubIssue.Spec.Title)
	}

	return existingIssue, ctrl.Result{}
}
```

### Step 2: Add GitHub Error Handlers

Add these error handling functions to `internal/controller/githubissue_controller.go`:

```go
func (r *GithubIssueReconciler) handleRetryableGitHubError(ctx context.Context, githubIssue *githubv1alpha1.GithubIssue, githubError *github.GitHubError) ctrl.Result {
	message := fmt.Sprintf("GitHub API temporarily unavailable: %s", githubError.Message)
	return r.setErrorConditionAndRequeue(ctx, githubIssue,
		"GitHubAPIError",
		message,
		time.Minute*2)
}

func (r *GithubIssueReconciler) handleNonRetryableGitHubError(ctx context.Context, githubIssue *githubv1alpha1.GithubIssue, githubError *github.GitHubError) ctrl.Result {
	message := fmt.Sprintf("GitHub API error: %s", githubError.Message)
	return r.setErrorConditionAndRequeue(ctx, githubIssue,
		"GitHubAPIError",
		message,
		time.Minute*5)
}

func (r *GithubIssueReconciler) handleUnknownError(ctx context.Context, githubIssue *githubv1alpha1.GithubIssue, err error) ctrl.Result {
	message := fmt.Sprintf("Unexpected error during GitHub lookup: %s", err.Error())
	return r.setErrorConditionAndRequeue(ctx, githubIssue,
		"UnexpectedError",
		message,
		time.Minute*1)
}
```

### Step 3: Update Reconcile Method

Replace the current `Reconcile` method implementation with:

```go
func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	githubIssue, err := r.fetchGithubIssue(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}

	if githubIssue == nil {
		log.Info("GithubIssue resource not found, skipping reconciliation", "githubissue", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	log.Info("Starting reconciliation", "githubissue", req.NamespacedName)

	githubClient, result := r.createGitHubClient(ctx, githubIssue)
	if !result.IsZero() {
		return result, nil
	}

	existingIssue, result := r.getIssueByTitle(ctx, githubClient, githubIssue)
	if !result.IsZero() {
		return result, nil
	}

	if existingIssue != nil {
		log.Info("Issue synchronization path: existing issue found")
		return r.synchronizeExistingIssue(ctx, githubClient, githubIssue, existingIssue)
	}

	log.Info("Issue creation path: no existing issue found")
	return r.createNewIssue(ctx, githubClient, githubIssue)
}
```

### Step 4: Add Placeholder Functions for Future Phases

Add these placeholder functions to maintain compilation:

```go
func (r *GithubIssueReconciler) synchronizeExistingIssue(ctx context.Context, githubClient *github.Client, githubIssue *githubv1alpha1.GithubIssue, existingIssue *github.Issue) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Synchronization logic not yet implemented", "issueID", existingIssue.GetID())
	
	return r.updateSuccessStatus(ctx, githubIssue)
}

func (r *GithubIssueReconciler) createNewIssue(ctx context.Context, githubClient *github.Client, githubIssue *githubv1alpha1.GithubIssue) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Issue creation logic not yet implemented")
	
	return r.updateSuccessStatus(ctx, githubIssue)
}
```

### Step 5: Add Required Imports

Ensure these imports are present in `internal/controller/githubissue_controller.go`:

```go
import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/github"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
```

## Implementation Strategy

### Code Organization
- **Single Responsibility**: Each function handles one specific concern
- **Error Classification**: Separate handlers for retryable vs non-retryable errors
- **Logging**: Descriptive log messages for debugging and monitoring
- **Future-Proof**: Placeholder functions for subsequent phases

### Error Handling Strategy
1. **GitHub API Errors**: Classified into retryable (5xx, 429) and non-retryable (4xx)
2. **Requeue Delays**: Graduated delays based on error type
3. **Status Conditions**: Clear error messages for troubleshooting

### Integration Points
- **GitHub Client**: Uses existing `GetIssueByTitle()` method
- **Error Handlers**: Reuses existing `setErrorConditionAndRequeue()` function
- **Status Management**: Maintains existing success status pattern

## Testing Considerations

### Unit Tests to Add
- `getIssueByTitle()` with successful issue found
- `getIssueByTitle()` with no issue found
- Error handling for retryable GitHub API errors
- Error handling for non-retryable GitHub API errors
- Integration flow through updated `Reconcile()` method

### Test Patterns
```go
// Example test structure
Describe("getIssueByTitle", func() {
    Context("when issue exists", func() {
        It("should return existing issue", func() {
            // Test implementation
        })
    })
    
    Context("when issue does not exist", func() {
        It("should return nil issue", func() {
            // Test implementation
        })
    })
    
    Context("when GitHub API returns retryable error", func() {
        It("should return requeue result", func() {
            // Test implementation
        })
    })
})
```

## Next Phase Preparation

This implementation provides the foundation for:
- **Phase 2b**: Issue creation using `CreateIssue()`
- **Phase 2c**: Issue updates using `UpdateIssue()`
- **Phase 3**: Advanced status field population

The placeholder functions (`synchronizeExistingIssue`, `createNewIssue`) will be expanded in subsequent implementations to handle the complete issue lifecycle.

## Validation

After implementation, verify:
1. Controller successfully looks up existing issues
2. Error conditions are properly set for GitHub API failures
3. Requeue logic works for transient errors
4. Logging provides useful debugging information
5. Unit tests cover all code paths

## Compliance Checklist

- ✅ **Single Responsibility**: Each function has one clear purpose
- ✅ **Functions < 50 lines**: All functions follow size guidelines
- ✅ **No Code Duplication**: Reuses existing error handling patterns
- ✅ **Clear Variable Names**: Descriptive identifiers throughout
- ✅ **No Inline Comments**: Code is self-explanatory
- ✅ **Error Classification**: Proper GitHub error type handling

