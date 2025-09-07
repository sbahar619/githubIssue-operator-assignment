# GitHub Integration Design

> **Navigation**: [Design](../../DESIGN.md) | [Standards](../../../standards/CODING_STANDARDS.md)

## Implementation
📋 [GitHub Client Implementation](CLIENT_IMPLEMENTATION.md)

---

## Design Philosophy

GitHub integration provides a clean abstraction layer between the controller and GitHub's REST API. The design focuses on reliability, error handling, and maintainability while providing all necessary operations for issue lifecycle management.

## Architecture Decisions

### API Client Design

#### Client Structure
**Pattern**: Repository-scoped client with embedded authentication using a struct that encapsulates the go-github client and repository details.

**Benefits**:
- **Scoped Operations**: All operations automatically target correct repository
- **Authentication Encapsulation**: Token management handled at client level
- **Type Safety**: Compile-time verification of required parameters

#### Authentication Strategy
**Pattern**: OAuth2 token-based authentication using go-github library with static token source for Personal Access Token authentication.

### GitHub Operations Mapping

#### Core CRUD Operations
1. **List Issues**: `GET /repos/{owner}/{repo}/issues` - Find existing issues by title
2. **Create Issue**: `POST /repos/{owner}/{repo}/issues` - Create new GitHub issue
3. **Update Issue**: `PATCH /repos/{owner}/{repo}/issues/{issue_number}` - Update issue description
4. **Close Issue**: `PATCH /repos/{owner}/{repo}/issues/{issue_number}` with `state: "closed"`

#### Extended Operations
5. **Check PR Association**: `GET /repos/{owner}/{repo}/issues/{issue_number}/events` - Detect linked PRs
6. **Get Issue Details**: `GET /repos/{owner}/{repo}/issues/{issue_number}` - Fetch current issue state

### Error Handling Strategy

#### Error Classification
1. **Authentication Errors**: Invalid token, insufficient permissions
2. **Network Errors**: Connection timeouts, DNS failures, service unavailable
3. **Rate Limit Errors**: GitHub API rate limit exceeded
4. **Resource Errors**: Repository not found, issue not found
5. **Validation Errors**: Invalid request parameters, malformed data

#### Error Recovery Patterns
Different error types require different recovery strategies: transient errors use exponential backoff, rate limit errors respect GitHub's reset time, and permanent errors don't retry.

#### GitHub API Response Handling
- **HTTP Status Codes**: Proper interpretation of 2xx, 4xx, 5xx responses
- **Rate Limit Headers**: Extract and respect `X-RateLimit-*` headers
- **Error Messages**: Parse GitHub API error responses for user-friendly messages

### Repository URL Parsing

#### URL Format Support
**Supported Format**: `https://github.com/{owner}/{repo}`
**Validation**: CRD-level validation ensures format compliance
**Parsing Logic**: URL parsing removes the GitHub prefix and splits the path into owner and repository components with validation.

#### Error Handling
- **Malformed URLs**: Clear error messages for invalid formats
- **Missing Components**: Specific errors for missing owner or repo
- **Validation**: Consistent with CRD validation patterns

### Pull Request Detection

#### Detection Strategy
**Approach**: Analyze GitHub issue events for PR associations
**API Endpoint**: `/repos/{owner}/{repo}/issues/{issue_number}/events`
**Event Types**: Look for `connected`, `cross-referenced`, or `mentioned` events from PRs

#### Implementation Considerations
- **Performance**: Cache PR detection results during reconciliation
- **Accuracy**: Handle edge cases (closed PRs, draft PRs, external references)
- **Reliability**: Graceful degradation if event API is unavailable

## Integration Points

### Controller Integration
Client creation during reconciliation involves URL parsing and GitHub client instantiation, followed by issue lifecycle management (list, create, update operations).

### Authentication Integration
Token retrieval from the auth layer with proper error handling before client creation.

### Status Integration
GitHub API responses are mapped to CR status fields with proper type conversion and timestamp tracking.

## Performance Considerations

### API Call Optimization
- **Batch Operations**: Minimize API calls per reconciliation cycle
- **Conditional Requests**: Use ETags and If-Modified-Since headers when available
- **Pagination**: Handle large issue lists efficiently
- **Caching**: Cache issue lists during single reconciliation cycle

### Rate Limit Management
- **Respect Limits**: Honor GitHub's rate limit headers
- **Backoff Strategy**: Exponential backoff with jitter for rate limit errors
- **Monitoring**: Track rate limit usage and remaining quota

### Connection Management
- **HTTP Client Reuse**: Reuse HTTP connections for multiple requests
- **Timeout Configuration**: Appropriate timeouts for different operation types
- **Connection Pooling**: Leverage go-github's built-in connection management

## Security Considerations

### Token Handling
- **No Logging**: Never log GitHub tokens in any form
- **Memory Safety**: Clear tokens from memory when possible
- **Scope Validation**: Verify token has necessary permissions

### Request Security
- **HTTPS Only**: All GitHub API requests use HTTPS
- **Certificate Validation**: Proper TLS certificate verification
- **User Agent**: Identify requests with appropriate User-Agent header

## Testing Strategy

### Unit Testing
- **Mock HTTP Client**: Test GitHub operations with mocked responses
- **Error Scenarios**: Comprehensive error condition testing
- **Edge Cases**: Handle unusual GitHub API responses

### Integration Testing
**GitHub Mock Library**: Use `go-github-mock` for realistic API simulation with configurable request matching and response mocking.

### E2E Testing
- **Real GitHub API**: Test against actual GitHub repository
- **Rate Limit Testing**: Verify rate limit handling
- **Error Recovery**: Test network failures and service outages

## Future Extensibility

### Additional GitHub Features
- **Issue Labels**: Label management and synchronization
- **Issue Assignees**: Assignment and team management
- **Issue Milestones**: Milestone association and tracking
- **Issue Comments**: Comment management and synchronization

### Enhanced Operations
- **Bulk Operations**: Batch issue creation and updates
- **Webhook Integration**: Real-time GitHub event processing
- **Advanced Search**: Complex issue queries and filtering

### Alternative Authentication
- **GitHub Apps**: App-based authentication for enhanced permissions
- **Fine-grained Tokens**: Support for fine-grained personal access tokens
- **OAuth Flows**: Interactive authentication for user-specific operations

## Error Types and Handling

### Custom Error Types
Custom error types provide structured error handling with retry information for transient errors, reset time for rate limits, and clear categorization for permanent errors.

### Error Context
- **Request Details**: Include request URL, method, and parameters in error context
- **Response Information**: Include HTTP status code and GitHub error messages
- **Retry Information**: Provide clear guidance on retry strategy

## Related Documentation
- 🏗️ [System Architecture](../ARCHITECTURE.md)
- 📋 [Implementation Phases](../PHASES.md)
- 📋 [GitHub Client Implementation](CLIENT_IMPLEMENTATION.md)
- 🔐 [Authentication Design](../auth/AUTH_DESIGN.md)
