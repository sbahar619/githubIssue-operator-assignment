# API Spec Design

> **Navigation**: [Design](../../DESIGN.md) | [Status Design](STATUS_DESIGN.md) | [Standards](../../../standards/CODING_STANDARDS.md)

## Implementation
📋 [Spec Implementation](../implementation/SPEC_IMPLEMENTATION.md)

---

## Design Philosophy

The `GithubIssue` spec follows Kubernetes declarative principles where users specify desired state. Each spec represents exactly one GitHub issue with a clear mapping between CR fields and GitHub API properties.

## Architecture Decisions

### Field Design Strategy

#### Spec Fields (User Intent)
- **Repo**: Full GitHub URL for unambiguous repository identification
- **Title**: Exact GitHub issue title for precise matching
- **Description**: Optional GitHub issue body content

### Validation Strategy

#### CRD-Level Validation (OpenAPI Schema)
**Purpose**: Prevent invalid resources from being created
**Scope**: Format, length, pattern validation
**Benefits**: 
- Fast feedback to users
- Reduces controller complexity
- Leverages Kubernetes API server capabilities

**Implementation**: CRD validation markers ensure format compliance

#### Controller-Level Validation (Business Logic)
**Purpose**: Handle complex validation requiring external state
**Scope**: Conflict detection, permission validation, GitHub API verification
**Benefits**:
- Access to external systems
- Complex business rule enforcement
- Dynamic validation based on current state

### Field Type Decisions

**Required Fields**: `repo`, `title` (essential for GitHub issue identification)
**Optional Fields**: `description` (GitHub allows empty issue bodies)
**Value Types**: All spec fields use `string` types for user intent clarity and validation simplicity

### Repository URL Validation

#### Full URL Requirement
**Decision**: Require full GitHub URLs (`https://github.com/owner/repo`)
**Rationale**:
- **Unambiguous**: No confusion about Git providers or protocols
- **Future-proof**: Easy to extend to GitHub Enterprise or other providers
- **User-friendly**: Clear expectation of what constitutes a valid repository

**Validation Pattern**: Regex pattern ensures valid GitHub URL format

**Length Constraints**:
- **MinLength**: 19 characters (shortest possible GitHub URL)
- **MaxLength**: 150 characters (reasonable upper bound for repository URLs)
