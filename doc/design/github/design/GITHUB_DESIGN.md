# GitHub Integration Design

> **Navigation**: [Design](../../DESIGN.md)

## Implementation
📋 [GitHub Client Implementation](../implementation/CLIENT_IMPLEMENTATION.md)

---

## Goal

Provide GitHub API integration for issue lifecycle management (create, update, close, list).

## Design

### Client Structure
**Pattern**: Repository-scoped client with embedded authentication
**Benefits**: Scoped operations, encapsulated auth, type safety

### Core Operations
1. **List Issues** - Find existing issues by title
2. **Create Issue** - Create new GitHub issue
3. **Update Issue** - Update issue description
4. **Close Issue** - Close issue on CR deletion
5. **Check PR Association** - Detect linked pull requests

### Repository URL Parsing
**Format**: `https://github.com/{owner}/{repo}`
**Validation**: CRD-level validation ensures format compliance
**Parsing**: Extract owner/repo from URL path

### Error Handling
**Error Types**: Authentication, network, rate limit, resource not found
**Recovery**: Exponential backoff for transient errors, respect rate limits
**Permanent Errors**: No retry for auth failures or missing resources

### Pull Request Detection
**Method**: Analyze issue events API for PR associations
**Caching**: Cache results during reconciliation cycle
