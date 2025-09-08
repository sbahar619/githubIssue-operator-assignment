# GitHub Client Design

> **Navigation**: [Design](../../DESIGN.md)

## Goal
Provide GitHub API integration for issue lifecycle management.

## Client Structure
- **Repository-scoped client** with embedded authentication
- **OAuth2 token authentication** via `go-github` library
- **Encapsulated operations** for single repository

## Core Operations
1. **ListIssues** - Fetch all open issues with pagination
2. **GetIssueByTitle** - Find issue by title (linear search)
3. **CreateIssue** - Create new issue with title/description
4. **UpdateIssue** - Update existing issue title/description  
5. **CloseIssue** - Close issue by setting state
6. **HasPullRequest** - Check if issue has associated PR

## Error Handling
- **Retryable**: 5xx server errors, 429 rate limit
- **Non-retryable**: 4xx client errors, authentication failures
- **Custom error type** with status code and retry classification

## URL Parsing
- **Input**: `https://github.com/owner/repo` (CRD validated)
- **Output**: Repository struct with owner/name fields
- **Method**: URL parsing + path splitting
