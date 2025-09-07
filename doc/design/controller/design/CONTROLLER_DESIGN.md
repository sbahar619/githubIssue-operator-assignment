# Controller Design

> **Navigation**: [Design](../../DESIGN.md) | [Standards](../../../standards/CODING_STANDARDS.md)

## Implementation
📋 [Reconciliation Implementation](RECONCILE_IMPLEMENTATION.md)

---

## Design Philosophy

The controller follows Kubernetes declarative principles: continuously reconcile actual state (GitHub issues) with desired state (GithubIssue CRs). The controller acts as the authoritative source of truth, enforcing CR specifications on GitHub rather than syncing GitHub changes back to CRs.

## Architecture Decisions

### Reconciliation Philosophy

#### Controller as Source of Truth
**Decision**: CR spec defines desired state, controller enforces it on GitHub
**Pattern**: Kubernetes CR → Controller → GitHub API (unidirectional)

**Benefits**:
- **Predictable Behavior**: Users know CR changes will be applied to GitHub
- **Conflict Prevention**: No confusion about which system is authoritative
- **Operational Clarity**: Clear ownership model for issue management

#### Declarative State Management
**Pattern**: Continuous reconciliation loop ensuring desired state
**Implementation**: Standard controller-runtime reconcile pattern with clear phases: fetch CR, fetch GitHub state, calculate diff, apply changes, update status

### Error Handling Strategy

#### Error Classification
1. **Transient Errors**: Network issues, GitHub API rate limits, temporary service unavailability
2. **Configuration Errors**: Missing secrets, invalid tokens, malformed repository URLs
3. **Business Logic Errors**: Conflicts between CRs, permission denied, repository not found
4. **System Errors**: Kubernetes API failures, controller bugs

#### Error Recovery Patterns
- **Transient Errors**: Exponential backoff retry with jitter
- **Configuration Errors**: Set error condition, don't requeue until manual fix
- **Business Logic Errors**: Set specific condition, provide clear user guidance
- **System Errors**: Log for debugging, requeue with standard interval

#### Status Condition Management
Standard Kubernetes condition types are used for consistent status reporting across different scenarios (ready, synced, error, conflict states).

### Conflict Resolution Approach

#### Single Issue Per CR Design
**Decision**: Each CR manages exactly one GitHub issue (identified by repo + title)
**Benefits**:
- **Simple Model**: Easy to understand and implement
- **Predictable Behavior**: Clear ownership boundaries
- **Conflict Prevention**: Built-in uniqueness constraint

#### Conflict Detection Strategy
**Trigger**: Multiple CRs request same GitHub issue (same repo + title)
**Resolution**: First-come-first-served with clear error reporting
**Implementation**:
1. Controller attempts to claim issue during reconciliation
2. If issue already managed by another CR, set Conflict condition
3. Provide descriptive error message with conflicting CR details

#### Conflict Error Messages
Descriptive error messages include conflicting CR details and namespace information for clear user guidance.

### Status Synchronization Strategy

#### GitHub State Mapping
**Purpose**: Reflect current GitHub issue state in CR status
**Timing**: After every successful GitHub API operation
**Fields**:
- `IssueID`: GitHub issue number for future operations
- `URL`: Direct link for user convenience
- `State`: Current GitHub issue state (open/closed)
- `HasPullRequest`: PR association detection
- `LastSyncTime`: Synchronization timestamp

#### Condition Updates
**Success Path**: Set Ready=True, Synced=True conditions
**Error Path**: Set appropriate error conditions with descriptive messages
**Conflict Path**: Set Conflict=True condition with conflicting CR details

### Performance Considerations

#### Resync Strategy
**Forced Reconciliation**: Regular resync period to detect external changes
**Purpose**: Handle GitHub issues modified outside the operator
**Behavior**: Controller detects changes and re-applies CR spec to GitHub

#### GitHub API Optimization
- **Batch Operations**: Minimize API calls per reconciliation
- **Conditional Requests**: Use ETags when available
- **Rate Limit Respect**: Built-in rate limiting and backoff

#### Memory Efficiency
- **Minimal State**: Controller maintains no persistent state between reconciliations
- **Garbage Collection**: Proper cleanup of temporary objects
- **Resource Limits**: Appropriate memory and CPU limits in deployment

## Integration Points

### Authentication Integration
Simple token retrieval during reconciliation with proper error handling and condition setting.

### GitHub Client Integration
Repository URL parsing and GitHub client creation with token authentication.

### Finalizer Integration
Standard Kubernetes finalizer pattern for deletion handling and cleanup coordination.

## Reconciliation Logic Flow

### Standard Reconciliation Path
The reconciliation follows a clear sequence: fetch CR, handle deletion, manage finalizers, authenticate, create GitHub client, sync issue state, update status, and handle conflicts.

### Error Handling Flow
Errors are classified by type (transient, configuration, business logic) with appropriate condition setting, logging, and requeue strategies based on error category.

## Helper Function Design

### Single Responsibility Functions
Following SRP, complex operations are broken into focused functions for authentication, GitHub operations, status management, and conflict detection.

### Utility Functions
Helper functions handle repository URL parsing, issue comparison, and pull request detection with clear single responsibilities.

## Testing Strategy

### Unit Testing Focus
- **Helper Functions**: 100% coverage for all helper functions
- **Error Scenarios**: Comprehensive error handling testing
- **Status Updates**: Condition management testing
- **Conflict Detection**: Multiple CR scenario testing

### Integration Testing
- **GitHub Mock**: Full reconciliation loop with mocked GitHub API
- **Error Simulation**: Network failures, authentication errors, API errors
- **State Transitions**: Complete lifecycle testing

### E2E Testing
- **Real GitHub**: End-to-end workflow with actual GitHub repository
- **Conflict Scenarios**: Multiple controller instances, concurrent operations
- **Failure Recovery**: Controller restart, network partition recovery

## Future Extensibility

### Additional GitHub Operations
- **Issue Labels**: Label management support
- **Issue Assignment**: Assignee management
- **Issue Milestones**: Milestone association

### Enhanced Conflict Resolution
- **Automatic Resolution**: Configurable conflict resolution strategies
- **Ownership Transfer**: Ability to transfer issue ownership between CRs
- **Conflict Notifications**: Alert mechanisms for conflicts

### Performance Optimizations
- **Caching**: GitHub issue caching for reduced API calls
- **Webhooks**: GitHub webhook integration for real-time updates
- **Batch Processing**: Multiple CR processing optimization

## Related Documentation
- 🏗️ [System Architecture](../ARCHITECTURE.md)
- 📋 [Implementation Phases](../PHASES.md)
- 📋 [Reconciliation Implementation](RECONCILE_IMPLEMENTATION.md)
- 🧪 [Testing Strategy](../../standards/TESTING_STRATEGY.md)
