# Authentication Design

> **Navigation**: [Design](../../DESIGN.md)

## Implementation
📋 [Token Implementation](../implementation/TOKEN_IMPLEMENTATION.md)

---

## Goal

Add logic to controller to retrieve GitHub token from environment variable.

## Design

### Controller Logic
**Simple Pattern**: `os.Getenv("GITHUB_TOKEN")` → Controller

**Why Environment Variables**:
- Controller doesn't care about the source (secret, configmap, direct value)
- Simple implementation with `os.Getenv()`
- Standard practice for configuration

### Controller Integration
```go
// In reconcile loop
token, err := auth.GetGitHubToken()
if err != nil {
    // Set error condition, requeue
}
// Use token with GitHub client
```

### Implementation
**Environment Variable**: `GITHUB_TOKEN`
**Function**: Simple `os.Getenv("GITHUB_TOKEN")` call
**Error Handling**: Return error if env var is empty

### Deployment Flexibility
The `GITHUB_TOKEN` environment variable can be set from:
- Kubernetes Secret (recommended)
- ConfigMap (not recommended for tokens)
- Direct value in deployment (not recommended)

Controller doesn't need to know the source.
