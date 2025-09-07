# API Status Design

> **Navigation**: [Design](../../DESIGN.md) | [Spec Design](SPEC_DESIGN.md) | [Standards](../../../standards/CODING_STANDARDS.md)

## Implementation
📋 [Status Implementation](../implementation/STATUS_IMPLEMENTATION.md)

---

## Design Philosophy

The `GithubIssue` status reflects current reality from GitHub API responses. Status fields provide observability into the controller's synchronization state and GitHub issue metadata.

## Architecture Decisions

### Field Design Strategy

#### Status Fields (Current Reality)
- **Conditions**: Standard Kubernetes condition pattern for state reporting
- **IssueID**: GitHub issue number/ID returned from GitHub API
- **URL**: Direct link to the GitHub issue
- **State**: Current GitHub issue state (open/closed)
- **HasPullRequest**: Indicates if the issue has an associated pull request
- **LastSyncTime**: When the issue was last synchronized with GitHub

### Field Type Decisions

**Pointer Types**: All status fields use pointers (`*string`, `*int`, `*bool`) for clear nil semantics and Kubernetes pattern compliance

### Status Field Architecture

#### Standard Kubernetes Patterns
Status follows standard Kubernetes condition patterns with GitHub-specific metadata fields using pointer types for clear nil semantics.

#### Status Structure Design
```go
type GithubIssueStatus struct {
    Conditions     []metav1.Condition `json:"conditions,omitempty"`     // Standard K8s conditions
    IssueID        *int               `json:"issueID,omitempty"`        // GitHub issue number
    URL            *string            `json:"url,omitempty"`            // Direct GitHub link
    State          *string            `json:"state,omitempty"`          // GitHub issue state
    HasPullRequest *bool              `json:"hasPullRequest,omitempty"` // PR association
    LastSyncTime   *metav1.Time       `json:"lastSyncTime,omitempty"`   // Sync timestamp
}
```

#### Condition Types Strategy
**Generic Documentation**: Avoid hardcoding specific condition types in comments
**Rationale**: Allows flexibility as requirements evolve without API changes
**Pattern**: Use descriptive but implementation-agnostic documentation

