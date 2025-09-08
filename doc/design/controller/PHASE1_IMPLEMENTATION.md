# Phase 1: GitHub Client Integration Implementation

> **Navigation**: [Design](../../DESIGN.md) | [Controller Plan](CONTROLLER_PLAN.md) | [Controller Design](CONTROLLER_DESIGN.md)

## Goal
Implement GitHub client initialization in the controller reconciliation loop using token and repository URL.

## Current State
✅ **Already Complete:**
- Reconcile function structure with CR fetch and error handling
- Authentication integration with `auth.GetGitHubToken()`
- Basic logging and context management
- Authentication error handling with conditions

🔄 **To Implement:**
- GitHub client initialization with token and repo URL

## Implementation Steps

### Step 1: Add GitHub Client Import
Add the GitHub client import to the controller:

```go
// File: internal/controller/githubissue_controller.go
import (
    "context"
    "time"

    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    logf "sigs.k8s.io/controller-runtime/pkg/log"

    githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
    "github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
    "github.com/sbahar619/githubIssue-operator-assignment/internal/github"  // Add this line
    "k8s.io/apimachinery/pkg/api/meta"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
```

### Step 2: Add GitHub Client Error Constants
Add constants for GitHub client errors:

```go
// File: internal/controller/githubissue_controller.go
const (
    ConditionTypeReady         = "Ready"
    ReasonAuthenticationFailed = "AuthenticationFailed"
    ReasonClientError          = "ClientError"          // Add this line
)
```

### Step 3: Integrate GitHub Client in Reconcile Function
Replace the current implementation after token retrieval:

```go
// File: internal/controller/githubissue_controller.go
func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := logf.FromContext(ctx)

    var githubIssue githubv1alpha1.GithubIssue
    if err := r.Get(ctx, req.NamespacedName, &githubIssue); err != nil {
        return ctrl.Result{}, client.IgnoreAlreadyExists(err)
    }

    log.Info("Starting reconciliation", "githubissue", req.NamespacedName)

    // Step 1: Authenticate (existing code)
    token, err := auth.GetGitHubToken()
    if err != nil {
        log.Error(err, "Failed to retrieve GitHub token", "githubissue", req.NamespacedName)
        meta.SetStatusCondition(&githubIssue.Status.Conditions, metav1.Condition{
            Type:    ConditionTypeReady,
            Status:  metav1.ConditionFalse,
            Reason:  ReasonAuthenticationFailed,
            Message: "GitHub token not available: " + err.Error(),
        })
        return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
    }

    // Step 2: Create GitHub client (NEW CODE)
    githubClient, err := github.NewClient(token, githubIssue.Spec.Repo)
    if err != nil {
        log.Error(err, "Failed to create GitHub client", "githubissue", req.NamespacedName, "repo", githubIssue.Spec.Repo)
        meta.SetStatusCondition(&githubIssue.Status.Conditions, metav1.Condition{
            Type:    ConditionTypeReady,
            Status:  metav1.ConditionFalse,
            Reason:  ReasonClientError,
            Message: "Failed to create GitHub client: " + err.Error(),
        })
        return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
    }

    log.Info("Successfully created GitHub client", "repo", githubIssue.Spec.Repo)

    // TODO: Phase 2 - Issue state synchronization will be implemented here
    _ = githubClient

    return ctrl.Result{}, nil
}
```

### Step 4: Update Status on Success
Update the controller to persist status changes:

```go
// File: internal/controller/githubissue_controller.go
// Add this after successful GitHub client creation, before return

// Update status to reflect successful client creation
if err := r.Status().Update(ctx, &githubIssue); err != nil {
    log.Error(err, "Failed to update status", "githubissue", req.NamespacedName)
    return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

return ctrl.Result{}, nil
```

## Complete Implementation

Here's the complete updated `Reconcile` function:

```go
func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := logf.FromContext(ctx)

    var githubIssue githubv1alpha1.GithubIssue
    if err := r.Get(ctx, req.NamespacedName, &githubIssue); err != nil {
        return ctrl.Result{}, client.IgnoreAlreadyExists(err)
    }

    log.Info("Starting reconciliation", "githubissue", req.NamespacedName)

    token, err := auth.GetGitHubToken()
    if err != nil {
        log.Error(err, "Failed to retrieve GitHub token", "githubissue", req.NamespacedName)
        meta.SetStatusCondition(&githubIssue.Status.Conditions, metav1.Condition{
            Type:    ConditionTypeReady,
            Status:  metav1.ConditionFalse,
            Reason:  ReasonAuthenticationFailed,
            Message: "GitHub token not available: " + err.Error(),
        })
        return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
    }

    githubClient, err := github.NewClient(token, githubIssue.Spec.Repo)
    if err != nil {
        log.Error(err, "Failed to create GitHub client", "githubissue", req.NamespacedName, "repo", githubIssue.Spec.Repo)
        meta.SetStatusCondition(&githubIssue.Status.Conditions, metav1.Condition{
            Type:    ConditionTypeReady,
            Status:  metav1.ConditionFalse,
            Reason:  ReasonClientError,
            Message: "Failed to create GitHub client: " + err.Error(),
        })
        return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
    }

    log.Info("Successfully created GitHub client", "repo", githubIssue.Spec.Repo)

    _ = githubClient

    if err := r.Status().Update(ctx, &githubIssue); err != nil {
        log.Error(err, "Failed to update status", "githubissue", req.NamespacedName)
        return ctrl.Result{RequeueAfter: time.Second * 30}, nil
    }

    return ctrl.Result{}, nil
}
```

## Testing
Run the following commands to test the implementation:

```bash
# Build and test
go build ./internal/controller
go test ./internal/controller -v

# Check for linting issues
make lint
```

## Success Criteria
- ✅ GitHub client successfully created with token and repo URL
- ✅ Error handling for invalid repository URLs
- ✅ Proper logging with structured fields
- ✅ Status conditions updated on errors
- ✅ Direct GitHub client creation (YAGNI principle)
- ✅ Ready for Phase 2: Issue State Synchronization
