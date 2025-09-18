# API Status Design

## Goal
Reflect observed state from GitHub API and controller synchronization.

## Status Fields
- **Conditions** (`[]metav1.Condition`): Standard Kubernetes condition pattern
- **IssueID** (`*int`): GitHub issue number from API response
- **URL** (`*string`): Direct GitHub issue link
- **State** (`*string`): Current GitHub issue state (open/closed)
- **HasPullRequest** (`*bool`): Whether issue has associated pull request
- **LastSyncTime** (`*metav1.Time`): Last synchronization timestamp

## Field Design Decisions
- **Pointer Types**: All status fields use pointers for clear nil semantics
- **Kubernetes Patterns**: Follows standard condition-based status reporting
- **GitHub Metadata**: Direct mapping from GitHub API responses

## Status Structure
```go
type GithubIssueStatus struct {
    Conditions     []metav1.Condition `json:"conditions,omitempty"`
    IssueID        *int               `json:"issueID,omitempty"`
    URL            *string            `json:"url,omitempty"`
    State          *string            `json:"state,omitempty"`
    HasPullRequest *bool              `json:"hasPullRequest,omitempty"`
    LastSyncTime   *metav1.Time       `json:"lastSyncTime,omitempty"`
}
```

