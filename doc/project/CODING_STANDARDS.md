# Coding Standards

> **Navigation**: [Design](../design/DESIGN.md) | [Testing Strategy](TESTING_STRATEGY.md) | [CI/CD Requirements](CICD_REQUIREMENTS.md)

## Core Principles

1. **Minimal Changes** - Smallest necessary modifications, reuse existing patterns
2. **Single Responsibility** - Each function has one clear purpose
3. **No Code Duplication** - Use helpers only when justified
4. **Functions < 50 lines** - Split complex functions
5. **Proper Encapsulation** - Hide implementation details, expose minimal API
6. **Logical Function Ordering** - Callers first, callees last

```go
// Good: Single responsibility
func (r *GithubIssueReconciler) fetchGithubIssues(ctx context.Context, repo string) ([]*github.Issue, error)
func (r *GithubIssueReconciler) findIssueByTitle(issues []*github.Issue, title string) *github.Issue

// Bad: Multiple responsibilities
func (r *GithubIssueReconciler) handleGithubOperations(ctx context.Context, issue *githubv1alpha1.GithubIssue) error
```

## Comments

- **No inline comments** - Code should be self-explanatory
- **GoDoc only when necessary** - For exported functions only
- **Generic documentation** - Avoid hardcoded implementation details
- **Clear variable names** - Prefer descriptive names over comments

```go
// Good
githubIssueTitle := githubIssue.Spec.Title
existingIssue := findIssueByTitle(allIssues, githubIssueTitle)

// Bad
title := gi.Spec.Title // title from spec
```

## Encapsulation Guidelines

### Information Hiding
- **Expose minimal public API** - Only functions needed by external packages
- **Use lowercase names** for internal/private functions
- **Group related functionality** in packages with clear boundaries

```go
// Good: Clean public API
func HandleAuthenticationError(...)     // Public - used by controller
func HandleGitHubAPIError(...)         // Public - used by controller

func handleRetryableError(...)         // Private - internal implementation
func handleNonRetryableError(...)      // Private - internal implementation

// Good: Local functions for single-use helpers
func ProcessComplexLogic(...) {
    localHelper := func(...) {
        // Implementation detail hidden inside function
    }
    
    localHelper(...)
}
```

### Package Boundaries
- **Clear responsibilities** - Each package has distinct purpose
- **Minimal dependencies** - Avoid circular imports
- **Pure functions** when possible - No side effects in utilities

```go
// Good: Pure utility functions
func FormatErrorMessage(err error) string          // No side effects
func ValidateInput(input string) error             // No side effects

// Bad: Mixed responsibilities
func HandleErrorAndRetry(err error) ctrl.Result    // Side effects + timing
```

## Function Ordering

### Callers First, Callees Last
- **Public API functions** at the top
- **Private helper functions** in the middle  
- **Core utility functions** at the bottom

```go
// 1. Public API (External callers)
func HandleAuthenticationError(...)
func HandleGitHubAPIError(...)

// 2. Private helpers (Internal callees)
func handleRetryableError(...)
func handleNonRetryableError(...)

// 3. Core utilities (Fundamental callees)
func SetCondition(...)
```

### Benefits
- **Top-down reading** - Start with interface, dive into implementation
- **Dependency flow** - Higher functions depend on lower ones
- **Easy navigation** - Find public API immediately

## File Organization

```
internal/
├── controller/
│   ├── githubissue_controller.go     # Main reconciliation logic
│   └── githubissue_controller_test.go
├── github/
│   ├── client.go                     # GitHub API operations and types
│   └── client_test.go
├── auth/
│   ├── token.go                      # Token retrieval logic
│   └── token_test.go
└── utils/
    ├── conditions.go                 # Status condition helpers
    └── validation.go                 # Input validation helpers
```

## Development Workflow

```bash
# Before committing
make lint test ci-checks
```

