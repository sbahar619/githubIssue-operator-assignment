# Controller Design

> **Navigation**: [Design](../../DESIGN.md)

## Goal
Design reconciliation logic for GitHub issue lifecycle management following Kubernetes controller patterns.

## Core Design Decisions

### Reconciliation Strategy
- **Single issue per CR** - One-to-one mapping prevents complexity
- **Controller authority** - CR spec enforces state on GitHub
- **Search-based lookup** - Use `GetIssueByTitle` for conflict detection
- **Status synchronization** - Reflect GitHub state in CR status

### Issue State Management
- **Create path** - Issue doesn't exist, create new one
- **Update path** - Issue exists, apply CR spec changes
- **Delete path** - CR deleted, close GitHub issue via finalizer
- **Conflict path** - Multiple CRs claim same issue, fail with error

### Error Classification
- **Authentication** - Token missing/invalid, requeue with delay
- **Retryable** - GitHub API 5xx/429, exponential backoff
- **Non-retryable** - Permission denied, invalid repo, set error condition
- **Conflicts** - Duplicate title/repo, set conflict condition

### Status Reporting
- **Conditions** - Standard Kubernetes pattern (Ready, Synced, Conflict)
- **GitHub metadata** - IssueID, URL, State, HasPullRequest, LastSyncTime
- **Error details** - Descriptive messages for troubleshooting

### Finalizer Strategy
- **Add on creation** - Ensure cleanup on CR deletion
- **Close GitHub issue** - Use stored IssueID from status
- **Remove after cleanup** - Standard Kubernetes finalizer pattern

## Integration Points
- **Authentication** - Use `auth.GetGitHubToken()` for token retrieval
- **GitHub client** - Use `github.NewClient()` with token and repo URL
- **API operations** - GetIssueByTitle, CreateIssue, UpdateIssue, CloseIssue
- **Conflict detection** - Search by title, check ownership via metadata
