# Ownership Design

> **Navigation**: [Controller Design](CONTROLLER_DESIGN.md) | [Main Design](../../DESIGN.md)

## Problem

Current implementation assumes ownership of any GitHub issue with matching title, creating safety risks:
- May modify human-created issues
- No conflict resolution between multiple CRs
- No way to identify operator-managed issues

## Solution: Label-Based Ownership

### Labels Schema
```yaml
# Primary ownership identifier (permanent)
"managed-by": "github-issue-operator"

# Specific CR ownership (removed on deletion)
"operator.github.shahaf.com/managed-by": "<namespace>/<cr-name>"
```

### Ownership Rules

| Scenario | Action | Reason |
|----------|--------|---------|
| **New CR + No existing issue** | ✅ Create with labels | Safe - operator creates |
| **New CR + External issue exists** | ❌ Error: `ExternallyOwnedIssue` | Respect external ownership |
| **New CR + Operator issue (same CR)** | ✅ Take ownership | Resume management (CR recreated) |
| **New CR + Operator issue (different CR)** | ❌ Error: `ConflictedOwnership` | Prevent conflicts |
| **New CR + Orphaned operator issue** | ✅ Take ownership | Reclaim orphaned |

### Implementation Flow

1. **Add Finalizer** - Ensure finalizer is present on CR
2. **Check Deletion** - Handle deletion if DeletionTimestamp is set
3. **Check Status.IssueID** - Use stored ID if available
4. **Lookup Issue** - GetIssueByID or fallback to GetIssueByTitle  
5. **Verify Ownership** - Check labels for ownership status
6. **Conflict Detection** - Compare CR identity with issue labels
7. **Take Action** - Create, update, or error based on ownership rules
8. **Ensure Open State** - Reopen issue if closed (for create/update paths)

### Simplified Ownership Logic
```go
func (r *GithubIssueReconciler) checkOwnership(issue *github.Issue, cr *githubv1alpha1.GithubIssue) OwnershipStatus {
    labels := getIssueLabels(issue)
    
    // Not operator managed
    if !hasLabel(labels, "managed-by", "github-issue-operator") {
        return ExternallyOwned
    }
    
    // Check specific CR ownership
    expectedOwner := fmt.Sprintf("%s/%s", cr.Namespace, cr.Name)
    actualOwner := getLabelValue(labels, "operator.github.shahaf.com/managed-by")
    
    if actualOwner == "" {
        return Orphaned  // Operator managed but no active owner
    }
    
    if actualOwner == expectedOwner {
        return OwnedByThisCR  // Perfect match
    }
    
    return OwnedByOtherCR  // Different CR owns it
}
```

### Finalizer Logic
```go
const FinalizerName = "github.shahaf.com/finalizer"

func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
        return r.handleDeletion(ctx, &githubIssue)
    }

    // 3. Normal reconciliation (create/update)
    return r.handleCreateOrUpdate(ctx, &githubIssue)
}

func (r *GithubIssueReconciler) handleDeletion(ctx context.Context, cr *githubv1alpha1.GithubIssue) (ctrl.Result, error) {
    if cr.Status.IssueID != nil {
        githubClient, err := r.newGitHubClient(ctx, cr)
        if err != nil {
            return ctrl.Result{RequeueAfter: time.Minute}, err
        }
        
        // Close GitHub issue
        if err := githubClient.CloseIssue(ctx, *cr.Status.IssueID); err != nil {
            return ctrl.Result{RequeueAfter: time.Minute}, err
        }
        
        // Remove CR ownership label (keep managed-by as tombstone)
        ownerLabel := fmt.Sprintf("%s/%s", cr.Namespace, cr.Name)
        if err := githubClient.RemoveLabel(ctx, *cr.Status.IssueID, "operator.github.shahaf.com/managed-by"); err != nil {
            // Log but don't fail - issue is closed
            log.Info("Failed to remove ownership label", "error", err)
        }
    }
    
    // Remove finalizer
    controllerutil.RemoveFinalizer(cr, FinalizerName)
    return ctrl.Result{}, r.Update(ctx, cr)
}

func (r *GithubIssueReconciler) ensureIssueOpen(ctx context.Context, githubClient *github.Client, issue *github.Issue) error {
    if issue.GetState() == "closed" {
        _, err := githubClient.ReopenIssue(ctx, issue.GetNumber())
        return err
    }
    return nil
}
```

## Scenario Examples

### "Same CR" Scenario - Simplified
```bash
# 1. Create CR -> GitHub issue created with labels
kubectl apply -f feature.yaml
# Labels: managed-by=operator, managed-by=team-a/feature

# 2. Delete CR -> Close issue, remove ownership label
kubectl delete -f feature.yaml  
# Labels: managed-by=operator (ownership label removed)

# 3. Recreate same CR -> Reclaim orphaned issue
kubectl apply -f feature.yaml  
# ✅ Detects orphaned issue, reopens it, adds ownership label back
```

### Label Lifecycle
```yaml
# Creation:
"managed-by": "github-issue-operator"                    # Permanent
"operator.github.shahaf.com/managed-by": "team-a/feature" # Added

# Deletion:
"managed-by": "github-issue-operator"                    # Kept as tombstone
# "operator.github.shahaf.com/managed-by" -> REMOVED

# Recreation:
"managed-by": "github-issue-operator"                    # Already present
"operator.github.shahaf.com/managed-by": "team-a/feature" # Re-added
```

## Benefits

- **Safety**: Never modifies externally-created issues
- **Transparency**: Visible ownership in GitHub UI
- **Flexibility**: Supports orphan recovery and CR recreation
- **Real-world ready**: Coexists with external workflows
