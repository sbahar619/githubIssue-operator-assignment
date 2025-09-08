# Authentication Design

## Goal
Retrieve GitHub token from environment variable for API authentication.

## Design Pattern
**Simple Environment Variable Lookup**: `os.Getenv("GITHUB_TOKEN")`

## Implementation
- **Function**: `auth.GetGitHubToken()` 
- **Environment Variable**: `GITHUB_TOKEN`
- **Error Handling**: Returns error if token is empty/unset
- **Controller Integration**: Called during reconciliation, sets error condition on failure

## Deployment Flexibility
Environment variable can be sourced from:
- **Kubernetes Secret** (recommended)
- **ConfigMap** (not recommended for tokens)  
- **Direct value** (not recommended)

Controller abstracts the source - only cares about the final environment variable value.
