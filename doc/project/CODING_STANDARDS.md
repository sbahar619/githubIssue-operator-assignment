# Coding Standards

> **Navigation**: [Design](../design/DESIGN.md) | [Testing Strategy](TESTING_STRATEGY.md) | [CI/CD Requirements](CICD_REQUIREMENTS.md)

## Core Principles

1. **Minimal Changes** - Smallest necessary modifications, reuse existing patterns
2. **Single Responsibility** - Each function has one clear purpose
3. **No Code Duplication** - Use helpers only when justified
4. **Functions < 50 lines** - Split complex functions

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

## File Organization

```
internal/
├── controller/
│   ├── githubissue_controller.go     # Main reconciliation logic
│   └── githubissue_controller_test.go
├── github/
│   ├── client.go                     # GitHub API operations
│   ├── client_test.go
│   └── types.go                      # GitHub-specific types
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

