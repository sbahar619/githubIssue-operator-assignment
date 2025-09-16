# Ownership Implementation

> **Navigation**: [Ownership Design](OWNERSHIP_DESIGN.md) | [Controller Design](CONTROLLER_DESIGN.md)

## 📋 Quick Implementation Guide

### Files to Modify:
1. **`internal/github/client.go`** - ADD 4 new methods
2. **`internal/controller/githubissue_controller.go`** - ADD constants, REPLACE Reconcile method, ADD 10+ new methods
3. **`internal/utils/conditions.go`** - ADD 2 new constants

### Implementation Steps:
1. **Phase 1**: Add GitHub client methods for labels and issue management
2. **Phase 2**: Update controller with ownership logic and new methods
3. **Phase 3**: Add error handling constants

---

## Implementation Plan

### Phase 1: GitHub Client Updates

**📁 File**: `internal/github/client.go`
**🎯 Action**: ADD new methods to existing Client struct

#### New Methods Required
```go
// 🆕 ADD these method signatures to Client struct
// Label management (keep CreateIssue unchanged - SRP)
func (c *Client) ListIssueLabels(ctx context.Context, issueNumber int) ([]string, error)
func (c *Client) AddLabelsToIssue(ctx context.Context, issueNumber int, labels []string) error
func (c *Client) RemoveLabelFromIssue(ctx context.Context, issueNumber int, labelName string) error

// Issue state management
func (c *Client) OpenIssue(ctx context.Context, issueNumber int) (*github.Issue, error)

// ✅ KEEP existing methods unchanged (single responsibility)
// func (c *Client) CreateIssue(ctx context.Context, title, description string) (*github.Issue, error)
// func (c *Client) CloseIssue(ctx context.Context, issueNumber int) (*github.Issue, error)
```

#### Implementation Details
**📁 File**: `internal/github/client.go`
**🎯 Action**: ADD these method implementations at the end of the file
```go
// 🆕 ADD - Label Management Methods
// Copy-paste these methods at the end of internal/github/client.go

func (c *Client) AddLabelsToIssue(ctx context.Context, issueNumber int, labels []string) error {
    _, _, err := c.githubClient.Issues.AddLabelsToIssue(ctx, c.repo.Owner, c.repo.Name, issueNumber, labels)
    if err != nil {
        return c.handleError(err)
    }
    return nil
}

func (c *Client) ListIssueLabels(ctx context.Context, issueNumber int) ([]string, error) {
    issue, _, err := c.githubClient.Issues.Get(ctx, c.repo.Owner, c.repo.Name, issueNumber)
    if err != nil {
        return nil, c.handleError(err)
    }
    
    return extractLabelNames(issue.Labels), nil
}

func (c *Client) RemoveLabelFromIssue(ctx context.Context, issueNumber int, labelName string) error {
    _, err := c.githubClient.Issues.RemoveLabelForIssue(ctx, c.repo.Owner, c.repo.Name, issueNumber, labelName)
    return c.handleError(err)
}

func (c *Client) OpenIssue(ctx context.Context, issueNumber int) (*github.Issue, error) {
    state := IssueStateOpen
    issueRequest := &github.IssueRequest{State: &state}
    
    issue, _, err := c.githubClient.Issues.Edit(ctx, c.repo.Owner, c.repo.Name, issueNumber, issueRequest)
    if err != nil {
        return nil, c.handleError(err)
    }
    return issue, nil
}

// Helper functions for GitHub client (add to internal/github/client.go)
func extractLabelNames(labels []*github.Label) []string {
    labelNames := make([]string, len(labels))
    for i, label := range labels {
        labelNames[i] = label.GetName()
    }
    return labelNames
}
```

### Phase 2: Controller Updates

**📁 File**: `internal/controller/githubissue_controller.go`
**🎯 Action**: ADD import, ADD constants and MODIFY existing Reconcile method

#### Required Imports
**📍 Location**: Add to import section in `internal/controller/githubissue_controller.go`
```go
import (
    // ... existing imports ...
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
    gogithub "github.com/google/go-github/v57/github"  // For *github.Issue types
)
```

#### Constants
**📍 Location**: Add at the top of `internal/controller/githubissue_controller.go` after imports
```go
const (
    FinalizerName = "github.shahaf.com/finalizer"
    
    // Labels
    OperatorManagedLabel = "operator.github.shahaf.com/managed-by:true"  // Static label (never removed)
    CROwnershipLabel     = "operator.github.shahaf.com/owner"            // Dynamic: "namespace/name" (removed on deletion)
)
```

#### Main Reconcile Logic
**📍 Location**: REPLACE existing Reconcile method in `internal/controller/githubissue_controller.go`
**🎯 Action**: Replace the entire existing Reconcile method (lines ~47-95) with this implementation
```go
func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := logf.FromContext(ctx)
    log.Info("Reconciliation started", "resource", req.NamespacedName)

    var githubIssue githubv1alpha1.GithubIssue
    if err := r.Get(ctx, req.NamespacedName, &githubIssue); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 1. Add finalizer if not present
    if !controllerutil.ContainsFinalizer(&githubIssue, FinalizerName) {
        controllerutil.AddFinalizer(&githubIssue, FinalizerName)
        return ctrl.Result{}, r.Update(ctx, &githubIssue)
    }

    // 2. Handle deletion
    if githubIssue.DeletionTimestamp != nil {
        if err := r.handleDeletion(ctx, &githubIssue); err != nil {
            return ctrl.Result{RequeueAfter: time.Minute}, err
        }
        // Remove finalizer after successful cleanup
        controllerutil.RemoveFinalizer(&githubIssue, FinalizerName)
        if err := r.Update(ctx, &githubIssue); err != nil {
            return ctrl.Result{}, err
        }
        return ctrl.Result{}, nil
    }

    // 3. Normal reconciliation (create/update)
    if err := r.handleCreateOrUpdate(ctx, &githubIssue); err != nil {
        return ctrl.Result{RequeueAfter: time.Minute}, err
    }
    
    return ctrl.Result{}, nil
}
```

#### Ownership Handling
```go
// 🆕 NEW METHOD
// File: internal/controller/githubissue_controller.go
// Action: ADD new ownership validation method
func (r *GithubIssueReconciler) HandleOwnership(ctx context.Context, githubClient *github.Client, issue *github.Issue, cr *githubv1alpha1.GithubIssue) error {
    labels, err := githubClient.ListIssueLabels(ctx, issue.GetNumber())
    if err != nil {
        return fmt.Errorf("failed to retrieve issue labels: %w", err)
    }
    
    // Check if issue is managed by operator
    if !hasOperatorManagedLabel(labels) {
        utils.SetCondition(ctx, r.Client, cr,
            metav1.ConditionFalse,
            utils.ReasonExternallyOwnedIssue,
            "Issue exists but is not managed by operator")
        return nil // Handled internally, don't requeue
    }
    
    // Check specific CR ownership
    expectedOwner := fmt.Sprintf("%s/%s", cr.Namespace, cr.Name)
    actualOwner := extractCROwnerFromLabels(labels)
    
    // If orphaned (no owner) or owned by this CR, we can proceed
    if actualOwner == "" || actualOwner == expectedOwner {
        return nil // Ownership verified, can proceed
    }
    
    // Issue is owned by different CR
    utils.SetCondition(ctx, r.Client, cr,
        metav1.ConditionFalse,
        utils.ReasonConflictedOwnership,
        fmt.Sprintf("Issue owned by different CR: expected %s, actual %s", expectedOwner, actualOwner))
    return nil // Handled internally, don't requeue
}

// Note: Custom error types removed - ownership errors handled internally

// Helper functions
func hasOperatorManagedLabel(labels []string) bool {
    for _, label := range labels {
        if strings.Contains(label, OperatorManagedLabel) {
            return true
        }
    }
    return false
}

func extractCROwnerFromLabels(labels []string) string {
    prefix := CROwnershipLabel + ":"
    for _, label := range labels {
        if strings.HasPrefix(label, prefix) {
            return strings.TrimPrefix(label, prefix)
        }
    }
    return ""
}
```

#### Create/Update Logic
**📍 Location**: ADD new methods at the end of `internal/controller/githubissue_controller.go`
**🎯 Action**: Copy-paste these new methods after the Reconcile method
```go
func (r *GithubIssueReconciler) handleCreateOrUpdate(ctx context.Context, cr *githubv1alpha1.GithubIssue) error {
    githubClient, err := r.newGitHubClient(ctx, cr)
    if err != nil {
        return err
    }
    
    existingIssue, err := r.getExistingIssue(ctx, githubClient, cr)
    if err != nil {
        utils.HandleGitHubAPIError(ctx, r.Client, cr, err)
        return err
    }
    
    if existingIssue != nil {
        if err := r.HandleOwnership(ctx, githubClient, existingIssue, cr); err != nil {
            // API or other errors - requeue (ownership errors handled internally)
            utils.HandleGitHubAPIError(ctx, r.Client, cr, err)
            return err
        }
        
        // Ownership verified, proceed with update
        if err := r.handleUpdateIssue(ctx, githubClient, cr, existingIssue); err != nil {
            return err
        }
        return nil
    }
    
    if err := r.createNewIssue(ctx, githubClient, cr); err != nil {
        return err
    }
    
    return nil
}

// 🆕 ADD - GitHub Client Helper
// Copy-paste this method after handleCreateOrUpdate
func (r *GithubIssueReconciler) newGitHubClient(ctx context.Context, cr *githubv1alpha1.GithubIssue) (*github.Client, error) {
    token, err := auth.GetGitHubToken()
    if err != nil {
        utils.HandleTokenRetrievalError(ctx, r.Client, cr, err)
        return nil, err
    }
    
    githubClient, err := github.NewClient(token, cr.Spec.Repo)
    if err != nil {
        utils.HandleGitHubAPIError(ctx, r.Client, cr, err)
        return nil, err
    }
    
    return githubClient, nil
}
```

#### Issue Creation
**🎯 Action**: Copy-paste these methods after newGitHubClient method
```go
// 🔄 MODIFY EXISTING - Update createNewIssue method
// Replace existing createNewIssue method (lines ~198-232) with this version
func (r *GithubIssueReconciler) createNewIssue(ctx context.Context, githubClient *github.Client, cr *githubv1alpha1.GithubIssue) error {
    log := logf.FromContext(ctx)
    log.Info("Creating new GitHub issue", "title", cr.Spec.Title)
    
    description := ""
    if cr.Spec.Description != nil {
        description = *cr.Spec.Description
    }
    
    // Step 1: Create issue (SRP - single responsibility)
    issue, err := githubClient.CreateIssue(ctx, cr.Spec.Title, description)
    if err != nil {
        utils.HandleGitHubAPIError(ctx, r.Client, cr, err)
        return fmt.Errorf("failed to create GitHub issue: %w", err)
    }
    
    // Step 2: Add operator labels (SRP - separate responsibility)
    labels := []string{
        OperatorManagedLabel,                                             // Static operator label
        fmt.Sprintf("%s:%s/%s", CROwnershipLabel, cr.Namespace, cr.Name), // Dynamic CR ownership
    }
    if err := githubClient.AddLabelsToIssue(ctx, issue.GetNumber(), labels); err != nil {
        log.Info("Failed to add labels to issue", "error", err, "issueID", issue.GetNumber())
        // Continue - issue was created successfully, labels are enhancement
    }
    
    if err := r.updateStatusFromGitHub(cr, issue); err != nil {
        log.Error(err, "Failed to update status after issue creation")
        return fmt.Errorf("failed to update status after issue creation: %w", err)
    }
    
    utils.SetCondition(ctx, r.Client, cr,
        metav1.ConditionTrue,
        utils.ReasonIssueCreated,
        fmt.Sprintf("GitHub issue #%d created successfully", issue.GetNumber()))
    
    log.Info("GitHub issue created successfully", "issueID", issue.GetNumber(), "url", issue.GetHTMLURL())
    return nil
}
```

#### Issue Updates
**🎯 Action**: Copy-paste these NEW methods after createNewIssue method
```go
func (r *GithubIssueReconciler) handleUpdateIssue(ctx context.Context, githubClient *github.Client, cr *githubv1alpha1.GithubIssue, existingIssue *gogithub.Issue) error {
    if err := r.prepareIssueForUpdate(ctx, githubClient, existingIssue, cr); err != nil {
        return fmt.Errorf("failed to prepare issue for update: %w", err)
    }
    
    if r.isUpdateNeeded(cr, existingIssue) {
        // Content differs, update required
        log.Info("Content differs, update required", "issueID", existingIssue.GetNumber())
        
        description := ""
        if cr.Spec.Description != nil {
            description = *cr.Spec.Description
        }
        updatedIssue, err := githubClient.UpdateIssue(ctx, existingIssue.GetNumber(), cr.Spec.Title, description)
        if err != nil {
            utils.HandleGitHubAPIError(ctx, r.Client, cr, err)
            return fmt.Errorf("failed to update GitHub issue: %w", err)
        }
        
        if err := r.updateStatusFromGitHub(cr, updatedIssue); err != nil {
            log.Error(err, "Failed to update status after issue update")
            return fmt.Errorf("failed to update status after issue update: %w", err)
        }
        
        utils.SetCondition(ctx, r.Client, cr,
            metav1.ConditionTrue,
            utils.ReasonIssueSynchronized,
            fmt.Sprintf("Issue #%d updated and synchronized successfully", updatedIssue.GetNumber()))
        log.Info("Issue updated successfully", "issueID", updatedIssue.GetNumber())
        
        return nil
    }
    
    // Update status from GitHub issue
    if err := r.updateStatusFromGitHub(cr, existingIssue); err != nil {
        logf.FromContext(ctx).Error(err, "Failed to update status")
        return fmt.Errorf("failed to update status: %w", err)
    }
    
    // Set synchronized condition
    utils.SetCondition(ctx, r.Client, cr,
        metav1.ConditionTrue,
        utils.ReasonIssueSynchronized,
        fmt.Sprintf("Issue #%d content matches desired state", existingIssue.GetNumber()))
    return nil
}

func (r *GithubIssueReconciler) prepareIssueForUpdate(ctx context.Context, githubClient *github.Client, issue *gogithub.Issue, cr *githubv1alpha1.GithubIssue) error {
    // Ensure issue is open
    if issue.GetState() == "closed" {
        if _, err := githubClient.OpenIssue(ctx, issue.GetNumber()); err != nil {
            utils.HandleGitHubAPIError(ctx, r.Client, cr, err)
            return err
        }
    }
    
    // Add ownership labels
    labels := []string{
        OperatorManagedLabel,                                             // Static operator label
        fmt.Sprintf("%s:%s/%s", CROwnershipLabel, cr.Namespace, cr.Name), // Dynamic CR ownership
    }
    if err := githubClient.AddLabelsToIssue(ctx, issue.GetNumber(), labels); err != nil {
        logf.FromContext(ctx).Info("Failed to update ownership labels", "error", err)
    }
    
    return nil
}
```

#### Deletion Logic
**🎯 Action**: Copy-paste these NEW methods after the update methods
```go
func (r *GithubIssueReconciler) handleDeletion(ctx context.Context, cr *githubv1alpha1.GithubIssue) error {
    log := logf.FromContext(ctx)
    log.Info("Handling CR deletion", "cr", cr.Name)
    
    if cr.Status.IssueID != nil {
        githubClient, err := r.newGitHubClient(ctx, cr)
        if err != nil {
            return err
        }
        
        if err := r.closeGitHubIssue(ctx, githubClient, cr); err != nil {
            return err
        }
    }
    
    return nil
}

func (r *GithubIssueReconciler) closeGitHubIssue(ctx context.Context, githubClient *github.Client, cr *githubv1alpha1.GithubIssue) error {
    log := logf.FromContext(ctx)
    
    // Close the GitHub issue
    if _, err := githubClient.CloseIssue(ctx, *cr.Status.IssueID); err != nil {
        log.Error(err, "Failed to close GitHub issue", "issueID", *cr.Status.IssueID)
        return err
    }
    
    // Remove dynamic ownership label (keep static managed-by label)
    ownershipLabel := fmt.Sprintf("%s:%s/%s", CROwnershipLabel, cr.Namespace, cr.Name)
    if err := githubClient.RemoveLabelFromIssue(ctx, *cr.Status.IssueID, ownershipLabel); err != nil {
        log.Info("Failed to remove ownership label", "error", err, "issueID", *cr.Status.IssueID)
    }
    // Note: OperatorManagedLabel stays to indicate this issue was managed by our operator
    
    log.Info("GitHub issue closed and ownership label removed", "issueID", *cr.Status.IssueID)
    return nil
}
```

### Phase 3: Error Handling Updates

**📁 File**: `internal/utils/conditions.go`
**🎯 Action**: ADD new constants and modify existing file

#### New Error Constants
**📍 Location**: Add to existing const block in `internal/utils/conditions.go` (after line 30)
```go
const (
    // 🆕 ADD these constants to existing const block
    // Existing constants remain unchanged...
    // ReasonIssueCreated        = "IssueCreated"
    // ReasonIssueSynchronized   = "IssueSynchronized"
    
    // ADD these new ownership error constants:
    ReasonExternallyOwnedIssue = "ExternallyOwnedIssue"
    ReasonConflictedOwnership  = "ConflictedOwnership"
)
```

#### Error Handling
Note: Use `utils.SetCondition` directly instead of creating wrapper functions.

## Testing Strategy

### Unit Tests (Ginkgo + Gomega)
```go
var _ = Describe("Ownership Logic", func() {
    Describe("hasOperatorManagedLabel", func() {
        It("should detect operator label", func() {
            labels := []string{"operator.github.shahaf.com/managed-by:team-a/feature", "other:label"}
            Expect(hasOperatorManagedLabel(labels)).To(BeTrue())
        })
        
        It("should return false for non-operator labels", func() {
            labels := []string{"bug", "enhancement"}
            Expect(hasOperatorManagedLabel(labels)).To(BeFalse())
        })
    })
    
    Describe("extractCROwnerFromLabels", func() {
        It("should extract owner from labels", func() {
            labels := []string{"operator.github.shahaf.com/managed-by:team-a/feature"}
            owner := extractCROwnerFromLabels(labels)
            Expect(owner).To(Equal("team-a/feature"))
        })
        
        It("should return empty string when no owner label", func() {
            labels := []string{"bug", "enhancement"}
            owner := extractCROwnerFromLabels(labels)
            Expect(owner).To(Equal(""))
        })
    })
    
    Describe("HandleOwnership", func() {
        It("should handle externally owned issues", func() {
            // Test external ownership rejection
        })
        
        It("should handle ownership conflicts", func() {
            // Test CR conflict detection
        })
        
        It("should proceed with owned/orphaned issues", func() {
            // Test successful ownership verification
        })
    })
})
```

### E2E Tests (Real GitHub API)
```go
var _ = Describe("Ownership E2E", func() {
    It("should create issue with operator labels", func() {
        // Test label creation and verification
    })
    
    It("should reclaim orphaned issues", func() {
        // Test orphan recovery scenario
    })
    
    It("should reject externally owned issues", func() {
        // Test conflict detection
    })
    
    It("should cleanup labels on deletion", func() {
        // Test deletion cleanup
    })
})
```

## Benefits of Simplified Approach

### Advantages Over Switch/Enum Pattern
- ✅ **Simpler Logic**: No intermediate enum types or complex switch statements
- ✅ **Direct Flow**: Ownership check and handling in single function
- ✅ **Fewer Abstractions**: Less cognitive overhead, easier to understand
- ✅ **Better Error Handling**: Direct error propagation without status confusion
- ✅ **Single Responsibility**: Each function has one clear purpose
- ✅ **Consistent Pattern**: All business logic functions return error only

### Code Reduction
- **Removed**: `OwnershipStatus` enum and constants
- **Removed**: Complex switch statement with multiple cases
- **Removed**: Separate `checkOwnership` + `handleExistingIssue` functions
- **Added**: Single `HandleOwnership` function (returns error only)
- **Added**: Custom error types for ownership decisions
- **Consistent**: All business logic functions return error only
- **Simplified**: Removed redundant `ManagedByOperatorLabel` - using single ownership label

### Maintainability
- **Easier to modify**: All ownership logic in one place
- **Clearer intent**: Function name describes exactly what it does
- **Less indirection**: No intermediate status values to track

## Migration Notes

### Breaking Changes
- Issues created before this implementation won't have operator labels
- Need migration strategy for existing issues

### Backward Compatibility
- Existing issues without labels will be treated as externally owned
- No automatic migration - manual intervention required
