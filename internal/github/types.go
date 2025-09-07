package github

import (
	"fmt"

	"github.com/google/go-github/v57/github"
)

const (
	IssueStateOpen   = "open"
	IssueStateClosed = "closed"
)

type Repository struct {
	Owner string
	Name  string
}

type GitHubError struct {
	StatusCode  int
	Message     string
	IsRetryable bool
}

func (e *GitHubError) Error() string {
	return fmt.Sprintf("GitHub API error (status %d): %s", e.StatusCode, e.Message)
}

type Client struct {
	client *github.Client
	repo   *Repository
}
