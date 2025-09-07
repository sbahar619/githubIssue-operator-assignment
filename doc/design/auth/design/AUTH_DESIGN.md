# Authentication Design

> **Navigation**: [Design](../../DESIGN.md) | [Standards](../../../standards/CODING_STANDARDS.md)

## Implementation
📋 [Secret Management Implementation](../implementation/SECRET_IMPLEMENTATION.md)

---

## Design Philosophy

Authentication follows the standard Kubernetes pattern of storing sensitive data in secrets and accessing it via environment variables. This approach provides security, simplicity, and operational familiarity while avoiding complex Kubernetes client interactions.

## Architecture Decisions

### Authentication Strategy

#### Environment Variable Approach
**Decision**: Read GitHub token directly from environment variable
**Pattern**: Secret → Environment Variable → `os.Getenv()` → Token

**Benefits**:
- **Standard Practice**: How most Kubernetes operators handle secrets
- **Simple Implementation**: No Kubernetes client complexity
- **Security**: No additional RBAC permissions required
- **Operations Friendly**: Clear deployment configuration

#### Alternative Rejected: Kubernetes Client Approach
**Pattern**: Secret → Kubernetes Client → TokenRetriever → Token
**Rejected Because**:
- **Unnecessary Complexity**: Requires Kubernetes client and RBAC permissions
- **Performance Overhead**: Additional API calls during reconciliation
- **Maintenance Burden**: More code to test and maintain

### Secret Management Strategy

#### Namespace Isolation
**Decision**: Each namespace has its own GitHub token secret
**Benefits**:
- **Multi-tenancy**: Different teams can use different GitHub tokens
- **Security Isolation**: Namespace boundaries provide security separation
- **Operational Flexibility**: Independent token rotation per namespace

#### Secret Naming Convention
**Default Name**: `github-token-secret`
**Environment Override**: `GITHUB_TOKEN_SECRET_NAME` environment variable
**Key Name**: `token` (standard for token secrets)

### Configuration Approach

#### Environment Variables
- **`GITHUB_TOKEN`**: The actual GitHub Personal Access Token (mounted from secret)
- **`GITHUB_TOKEN_SECRET_NAME`**: Optional override for secret name (defaults to `github-token-secret`)

#### Deployment Configuration
Environment variables are configured in the manager deployment to mount secrets as environment variables.

## Integration Points

### Controller Integration
**Token Retrieval**: Simple function call during reconciliation with graceful error handling

**Error Handling**: Graceful degradation when token is unavailable
- Set error condition in CR status
- Requeue for retry after reasonable interval
- Provide clear error messages to users

### GitHub Client Integration
**Token Usage**: Pass token to GitHub client constructor using OAuth2 token source pattern

### Deployment Integration
**Helm Chart**: Template for secret creation and environment variable mounting
**Documentation**: Clear instructions for token setup and rotation

## Security Considerations

### Token Storage
- **Kubernetes Secrets**: Encrypted at rest (if cluster configured)
- **Base64 Encoding**: Standard Kubernetes secret encoding
- **Namespace Scoped**: Secrets isolated to specific namespaces

### Token Access
- **Environment Variables**: Standard container security model
- **No Logging**: Token never logged or exposed in error messages
- **Memory Only**: Token exists only in process memory

### Token Rotation
- **Zero Downtime**: Update secret, restart controller pods
- **Validation**: Controller validates token on startup
- **Fallback**: Clear error reporting for invalid tokens

## Error Handling Strategy

### Missing Token Scenarios
1. **Secret Not Found**: Clear error message, requeue for retry
2. **Empty Token**: Validation error, don't requeue until fixed
3. **Invalid Token**: GitHub API error, set error condition

### Error Recovery
- **Automatic Retry**: For transient errors (network, temporary GitHub issues)
- **Manual Intervention**: For configuration errors (missing secret, invalid token)
- **Status Reporting**: Clear error messages in CR conditions

## Performance Considerations

### Token Caching
**Decision**: No caching, read from environment on each reconciliation
**Rationale**: 
- **Simplicity**: No cache invalidation complexity
- **Performance**: Environment variable access is extremely fast
- **Security**: No token stored in memory longer than necessary

### Startup Validation
**Decision**: Validate token availability on controller startup
**Benefits**:
- **Fail Fast**: Detect configuration issues early
- **Clear Feedback**: Startup logs indicate authentication status
- **Operational Safety**: Prevent silent failures

## Operational Considerations

### Secret Creation
**Manual Process**: Operators create secrets with GitHub tokens
**Automation**: Helm chart templates for consistent secret structure
**Documentation**: Clear instructions for token generation and setup

### Token Requirements
**GitHub Permissions**: Repository access for issue management
**Token Type**: Personal Access Token (classic or fine-grained)
**Scope Requirements**: `repo` scope for private repositories, `public_repo` for public

### Monitoring and Alerting
**Authentication Failures**: Monitor GitHub API authentication errors
**Token Expiration**: Alert on approaching token expiration (if using fine-grained tokens)
**Secret Availability**: Monitor secret existence and validity

## Future Extensibility

### Multiple Token Support
**Potential**: Different tokens for different repositories
**Current**: Single token per namespace (sufficient for most use cases)
**Migration**: Could extend to repository-specific token mapping

### Token Types
**Current**: Personal Access Token support
**Future**: GitHub App authentication, OAuth flows
**Design**: Interface allows for different authentication methods

### Secret Providers
**Current**: Kubernetes secrets only
**Future**: External secret managers (Vault, AWS Secrets Manager)
**Design**: Environment variable interface allows for different secret sources

## Related Documentation
- 🏗️ [System Design](../../DESIGN.md)
- 📋 [Secret Management Implementation](../implementation/SECRET_IMPLEMENTATION.md)
