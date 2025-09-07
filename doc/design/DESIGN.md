# GitHub Issue Operator - Design

> **Primary Navigation Hub**

## Standards
- [Coding Standards](../standards/CODING_STANDARDS.md)
- [Testing Strategy](../standards/TESTING_STRATEGY.md) 
- [CI/CD Requirements](../standards/CICD_REQUIREMENTS.md)

## Components
- **API**: [Spec Design](api/design/SPEC_DESIGN.md) | [Status Design](api/design/STATUS_DESIGN.md) → [Spec Implementation](api/implementation/SPEC_IMPLEMENTATION.md) | [Status Implementation](api/implementation/STATUS_IMPLEMENTATION.md)
- **Auth**: [Auth Design](auth/design/AUTH_DESIGN.md) → [Token Implementation](auth/implementation/TOKEN_IMPLEMENTATION.md)
- **Controller**: [Controller Design](controller/design/CONTROLLER_DESIGN.md)
- **GitHub**: [GitHub Design](github/design/GITHUB_DESIGN.md)
- **Finalizer**: [Finalizer Design](finalizer/design/FINALIZER_DESIGN.md)

---

## Overview

Kubernetes operator that manages GitHub issues via Custom Resource Definitions. Each CR maps to one GitHub issue with conflict resolution and state synchronization.

**Current State**: Kubebuilder v4.7.1 scaffold, needs implementation

## Design Principles

- **Declarative**: Users declare desired GitHub issue state
- **Single Issue Per CR**: One-to-one mapping prevents complexity
- **Controller Authority**: Controller enforces CR spec on GitHub
- **Conflict Prevention**: First-come-first-served with clear error messages

## Architecture

```
User Creates CR → Controller Reconciles → GitHub API → Status Updated
                      ↓
                 Environment Variable ← Kubernetes Secret
```

**Key Decisions**:
1. **Auth**: GitHub token from environment variable (mounted from secret)
2. **Validation**: CRD-level patterns + controller business logic
3. **Status**: Standard Kubernetes conditions + GitHub metadata
4. **Conflicts**: Detect duplicate titles, fail with descriptive errors

## Implementation Plan

### Phase 1: API Foundation
- Add CRD validation markers (repo URL, title, description)
- Implement status structure (conditions, GitHub metadata)
- **Design**: [Spec Design](api/design/SPEC_DESIGN.md), [Status Design](api/design/STATUS_DESIGN.md)
- **Implementation**: [Spec Implementation](api/implementation/SPEC_IMPLEMENTATION.md), [Status Implementation](api/implementation/STATUS_IMPLEMENTATION.md)

### Phase 2: Authentication
- Environment variable token retrieval
- Controller integration for GitHub API calls
- **Design**: [Auth Design](auth/design/AUTH_DESIGN.md)
- **Implementation**: [Token Implementation](auth/implementation/TOKEN_IMPLEMENTATION.md)

### Phase 3: GitHub Integration
- GitHub client (list, create, update, close issues)
- Repository URL parsing and error handling
- **Design**: [GitHub Design](github/design/GITHUB_DESIGN.md)

### Phase 4: Controller Logic
- Reconciliation loop (fetch issues, find/create, update status)
- Conflict detection and status conditions
- **Design**: [Controller Design](controller/design/CONTROLLER_DESIGN.md)

### Phase 5: Finalizers
- Cleanup logic for CR deletion
- GitHub issue closure with error handling
- **Design**: [Finalizer Design](finalizer/design/FINALIZER_DESIGN.md)

### Phase 6: Production
- RBAC roles, Helm chart, documentation
- E2E testing and CI/CD integration

## Deployment Requirements

### Helm Chart Structure
- **Chart Components**: Deployment, RBAC, Service, ServiceAccount, ConfigMap, CRDs
- **Configuration**: Environment variables for GitHub token and secret name
- **Resource Management**: CPU/memory limits and requests
- **Security**: RBAC roles for controller and user access (viewer, editor)

## Success Criteria

- ✅ Create/update/close GitHub issues from Kubernetes CRs
- ✅ Conflict detection between multiple CRs
- ✅ Proper status reporting and error handling
- ✅ >90% test coverage with unit/integration/E2E tests
- ✅ Production-ready deployment with Helm
