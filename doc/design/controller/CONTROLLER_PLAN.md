# Controller Implementation Plan

> **Navigation**: [Design](../../DESIGN.md) | [Controller Design](CONTROLLER_DESIGN.md)

## Goal
High-level phases for implementing controller reconciliation logic based on design decisions.

## Implementation Phases

### Phase 1: GitHub Client Integration ✅ **Complete**
- ~~Enhance Reconcile function with CR fetch and error handling~~ ✅ **Complete**
- ~~Add authentication integration with `auth.GetGitHubToken()`~~ ✅ **Complete**
- ~~Create GitHub client initialization with token and repo URL~~ ✅ **Complete**
- ~~Basic logging and context management~~ ✅ **Complete**

### Phase 2: Issue State Synchronization
- Implement issue lookup using `GetIssueByTitle()`
- Add create path for new issues with `CreateIssue()`
- Add update path for existing issues with `UpdateIssue()`
- Basic status field population (IssueID, URL, State)

### Phase 3: Status Management
- Implement Kubernetes condition patterns (Ready, Synced)
- Add GitHub metadata synchronization (HasPullRequest, LastSyncTime)
- Error condition setting with descriptive messages
- Status update persistence

### Phase 4: Error Handling & Classification ✅ **Authentication Errors Done**
- ~~Authentication error handling with requeue delays~~ ✅ **Complete**
- GitHub API error classification (retryable vs non-retryable)
- Exponential backoff for transient errors
- ~~Error condition management~~ ✅ **Basic Complete**

### Phase 5: Finalizer Integration
- Add finalizer on CR creation
- Implement deletion detection and GitHub issue cleanup
- Issue closure using stored IssueID from status
- Finalizer removal after successful cleanup

### Phase 6: Conflict Detection
- Multi-CR conflict detection via title/repo lookup
- Conflict condition setting with detailed error messages
- First-come-first-served ownership model
- Conflict resolution error handling

## Dependencies
- **Auth module** - Token retrieval functionality
- **GitHub client** - All API operations (GetIssueByTitle, CreateIssue, UpdateIssue, CloseIssue)
- **API types** - CR spec and status structure
- **Finalizer design** - Cleanup strategy and patterns
