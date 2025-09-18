# Controller Design

> **Navigation**: [Design](../../DESIGN.md)

## Goal
Design reconciliation logic for GitHub issue lifecycle management following Kubernetes controller patterns.

## Core Design Decisions

### Reconciliation Strategy
- **Single issue per CR** - One-to-one mapping prevents complexity
- **Label-based ownership** - Only manage issues with operator labels
- **ID-first lookup** - Use stored IssueID, fallback to title search
- **Status synchronization** - Reflect GitHub state in CR status

### Issue State Management
- **Create path** - Issue doesn't exist, create with operator labels, ensure open
- **Update path** - Issue exists with operator labels, apply CR spec changes, ensure open
- **Delete path** - CR deleted, close GitHub issue via finalizer, remove finalizer
- **Ownership verification** - Check labels before taking any action

### Error Classification
- **User-actionable errors** - Update status for user visibility and action
- **Infrastructure errors** - Log only, let controller retry handle
- **Authentication** - Token missing/invalid, user can fix
- **Ownership conflicts** - User needs to resolve external ownership
- **GitHub API errors** - Could be user permissions or service issues

### Status Reporting
- **Final states only** - No intermediate processing states
- **User-actionable errors only** - Status updates when user can take action  
- **Infrastructure errors** - Logged only, controller retries automatically
- **Conditions** - Standard Kubernetes pattern (Ready with specific reasons)
- **GitHub metadata** - IssueID, URL, State, HasPullRequest, LastSyncTime

### Error Handling Pattern
**Status Updates (user-visible):**
- Authentication failures (fix token/secret)
- Ownership conflicts (resolve external ownership)  
- GitHub API errors (check permissions/repo access)

**Log Only (controller-handled):**
- Network timeouts and retryable errors
- Internal status update failures
- Cleanup operation failures during deletion
- Transient infrastructure issues
- **Error details** - Descriptive messages for troubleshooting

### Finalizer Strategy
- **Add on first reconcile** - Ensure cleanup on CR deletion (`github.shahaf.com/finalizer`)
- **Deletion handling** - Close GitHub issue using stored IssueID from status
- **Error handling** - Retry on GitHub API errors during deletion
- **Remove after cleanup** - Standard Kubernetes finalizer pattern

## Integration Points
- **Authentication** - Use `auth.GetGitHubToken()` for token retrieval
- **GitHub client** - Use `github.NewClient()` with token and repo URL
- **API operations** - GetIssueByID, GetIssueByTitle, CreateIssue, UpdateIssue, CloseIssue, OpenIssue
- **Ownership verification** - Check labels: `managed-by: github-issue-operator`
