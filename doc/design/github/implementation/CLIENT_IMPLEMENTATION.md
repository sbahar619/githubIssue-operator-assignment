# GitHub Client Implementation

> **Context**: [GitHub Design](../design/GITHUB_DESIGN.md) | [Design](../../DESIGN.md)

## Overview

Implement GitHub API client for issue lifecycle management with repository-scoped operations and error handling.

## File Structure

```
internal/github/
├── client.go           # GitHub client and operations
├── client_test.go      # Unit tests
└── types.go           # GitHub-specific types and errors
```

## Step 1: GitHub Types

**File**: `internal/github/types.go`

```go
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
```

## Step 2: GitHub Client

**File**: `internal/github/client.go`

```go
package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

func ParseRepositoryURL(repoURL string) (*Repository, error) {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	pathSegments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")

	return &Repository{
		Owner: pathSegments[0],
		Name:  pathSegments[1],
	}, nil
}

func NewGitHubError(statusCode int, message string) *GitHubError {
	isRetryable := statusCode >= 500 || statusCode == 429
	return &GitHubError{
		StatusCode:  statusCode,
		Message:     message,
		IsRetryable: isRetryable,
	}
}

func NewClient(token, repoURL string) (*Client, error) {
	repo, err := ParseRepositoryURL(repoURL)
	if err != nil {
		return nil, err
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), tokenSource)
	githubClient := github.NewClient(httpClient)

	return &Client{
		client: githubClient,
		repo:   repo,
	}, nil
}

func (c *Client) handleError(err error) error {
	if githubError, ok := err.(*github.ErrorResponse); ok {
		errorMessage := "unknown error"
		if githubError.Message != "" {
			errorMessage = githubError.Message
		}
		return NewGitHubError(githubError.Response.StatusCode, errorMessage)
	}

	return &GitHubError{
		StatusCode:  0,
		Message:     err.Error(),
		IsRetryable: false,
	}
}

func (c *Client) ListIssues(ctx context.Context) ([]*github.Issue, error) {
	opts := &github.IssueListByRepoOptions{
		State: IssueStateOpen,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allIssues []*github.Issue
	for {
		issues, resp, err := c.client.Issues.ListByRepo(ctx, c.repo.Owner, c.repo.Name, opts)
		if err != nil {
			return nil, c.handleError(err)
		}

		allIssues = append(allIssues, issues...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

func (c *Client) GetIssueByTitle(ctx context.Context, title string) (*github.Issue, error) {
	allIssues, err := c.ListIssues(ctx)
	if err != nil {
		return nil, err
	}

	for _, issue := range allIssues {
		if issue.GetTitle() == title {
			return issue, nil
		}
	}

	return nil, nil
}

func (c *Client) CreateIssue(ctx context.Context, title, description string) (*github.Issue, error) {
	issueRequest := &github.IssueRequest{
		Title: &title,
	}

	if description != "" {
		issueRequest.Body = &description
	}

	issue, _, err := c.client.Issues.Create(ctx, c.repo.Owner, c.repo.Name, issueRequest)
	if err != nil {
		return nil, c.handleError(err)
	}

	return issue, nil
}

func (c *Client) UpdateIssue(ctx context.Context, issueNumber int, title, description string) (*github.Issue, error) {
	issueRequest := &github.IssueRequest{
		Title: &title,
	}

	if description != "" {
		issueRequest.Body = &description
	}

	issue, _, err := c.client.Issues.Edit(ctx, c.repo.Owner, c.repo.Name, issueNumber, issueRequest)
	if err != nil {
		return nil, c.handleError(err)
	}

	return issue, nil
}

func (c *Client) CloseIssue(ctx context.Context, issueNumber int) (*github.Issue, error) {
	state := IssueStateClosed
	issueRequest := &github.IssueRequest{
		State: &state,
	}

	issue, _, err := c.client.Issues.Edit(ctx, c.repo.Owner, c.repo.Name, issueNumber, issueRequest)
	if err != nil {
		return nil, c.handleError(err)
	}

	return issue, nil
}

func (c *Client) HasPullRequest(ctx context.Context, issueNumber int) (bool, error) {
	issue, _, err := c.client.Issues.Get(ctx, c.repo.Owner, c.repo.Name, issueNumber)
	if err != nil {
		return false, c.handleError(err)
	}

	return issue.PullRequestLinks != nil, nil
}
```

## Step 3: Unit Tests

**File**: `internal/github/client_test.go`

```go
package github

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-github/v57/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	repoURL          = "https://github.com/owner/repo"
	repoOwner        = "owner"
	repoName         = "repo"
	existingTitle    = "Test Issue"
	newTitle         = "New Issue"
	nonExistentTitle = "Non-existing Issue"
	invalidURL       = "://invalid-url"
	token            = "test-token"
)

func TestGitHub(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GitHub Suite")
}

var _ = Describe("Repository URL Parsing", func() {
	Describe("ParseRepositoryURL", func() {
		It("should parse valid GitHub URL", func() {
			repo, err := ParseRepositoryURL(repoURL)

			Expect(err).NotTo(HaveOccurred())
			Expect(repo.Owner).To(Equal(repoOwner))
			Expect(repo.Name).To(Equal(repoName))
		})

		It("should handle URL parsing errors", func() {
			_, err := ParseRepositoryURL(invalidURL)
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse URL"))
		})
	})
})

var _ = Describe("GitHub Error", func() {
	It("should classify server errors as retryable", func() {
		err := NewGitHubError(500, "Internal Server Error")
		
		Expect(err.IsRetryable).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("status 500"))
	})

	It("should classify rate limit as retryable", func() {
		err := NewGitHubError(429, "Rate limit exceeded")
		
		Expect(err.IsRetryable).To(BeTrue())
	})

	It("should classify client errors as non-retryable", func() {
		err := NewGitHubError(404, "Not Found")
		
		Expect(err.IsRetryable).To(BeFalse())
	})
})

var _ = Describe("Client Constructor", func() {
	Describe("NewClient", func() {
		It("should create client with valid repository URL", func() {
			client, err := NewClient(token, repoURL)

			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
			Expect(client.repo.Owner).To(Equal(repoOwner))
			Expect(client.repo.Name).To(Equal(repoName))
		})

		It("should return error with invalid repository URL", func() {
			_, err := NewClient(token, invalidURL)
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse URL"))
		})
	})
})

var _ = Describe("GitHub Client Operations", func() {
	var (
		client *Client
		ctx    context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		
		mockedHTTPClient := mock.NewMockedHTTPClient(
			mock.WithRequestMatch(
				mock.GetReposIssuesByOwnerByRepo,
				[]github.Issue{
					{ID: github.Int64(1), Title: github.String(existingTitle), State: github.String(IssueStateOpen)},
				},
			),
			mock.WithRequestMatch(
				mock.PostReposIssuesByOwnerByRepo,
				github.Issue{ID: github.Int64(2), Title: github.String(newTitle), State: github.String(IssueStateOpen)},
			),
			mock.WithRequestMatch(
				mock.GetReposIssuesByOwnerByRepoByIssueNumber,
				github.Issue{ID: github.Int64(1), Title: github.String(existingTitle), State: github.String(IssueStateOpen), PullRequestLinks: nil},
			),
			mock.WithRequestMatch(
				mock.PatchReposIssuesByOwnerByRepoByIssueNumber,
				github.Issue{ID: github.Int64(1), Title: github.String("Updated Title"), State: github.String(IssueStateOpen)},
			),
		)

		githubClient := github.NewClient(mockedHTTPClient)
		
		client = &Client{
			client: githubClient,
			repo:   &Repository{Owner: repoOwner, Name: repoName},
		}
	})

	Describe("ListIssues", func() {
		It("should return list of issues", func() {
			issues, err := client.ListIssues(ctx)
			
			Expect(err).NotTo(HaveOccurred())
			Expect(issues).To(HaveLen(1))
			Expect(issues[0].GetTitle()).To(Equal(existingTitle))
		})
	})

	Describe("GetIssueByTitle", func() {
		It("should get existing issue", func() {
			issue, err := client.GetIssueByTitle(ctx, existingTitle)
			
			Expect(err).NotTo(HaveOccurred())
			Expect(issue).NotTo(BeNil())
			Expect(issue.GetTitle()).To(Equal(existingTitle))
		})

		It("should return nil for non-existing issue", func() {
			issue, err := client.GetIssueByTitle(ctx, nonExistentTitle)
			
			Expect(err).NotTo(HaveOccurred())
			Expect(issue).To(BeNil())
		})
	})

	Describe("CreateIssue", func() {
		It("should create new issue", func() {
			issue, err := client.CreateIssue(ctx, newTitle, "Description")
			
			Expect(err).NotTo(HaveOccurred())
			Expect(issue).NotTo(BeNil())
			Expect(issue.GetTitle()).To(Equal(newTitle))
		})
	})

	Describe("UpdateIssue", func() {
		It("should update existing issue", func() {
			issue, err := client.UpdateIssue(ctx, 1, "Updated Title", "Updated Description")
			
			Expect(err).NotTo(HaveOccurred())
			Expect(issue).NotTo(BeNil())
			Expect(issue.GetTitle()).To(Equal("Updated Title"))
		})
	})

	Describe("CloseIssue", func() {
		It("should close existing issue", func() {
			issue, err := client.CloseIssue(ctx, 1)
			
			Expect(err).NotTo(HaveOccurred())
			Expect(issue).NotTo(BeNil())
		})
	})

	Describe("HasPullRequest", func() {
		It("should return false when issue has no pull request", func() {
			hasPR, err := client.HasPullRequest(ctx, 1)
			
			Expect(err).NotTo(HaveOccurred())
			Expect(hasPR).To(BeFalse())
		})

		It("should return true when issue has pull request", func() {
			mockedHTTPClientWithPR := mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.GetReposIssuesByOwnerByRepoByIssueNumber,
					github.Issue{ID: github.Int64(2), Title: github.String(existingTitle), State: github.String(IssueStateOpen), PullRequestLinks: &github.PullRequestLinks{URL: github.String("https://api.github.com/repos/owner/repo/pulls/123")}},
				),
			)

			githubClientWithPR := github.NewClient(mockedHTTPClientWithPR)
			clientWithPR := &Client{
				client: githubClientWithPR,
				repo:   &Repository{Owner: repoOwner, Name: repoName},
			}

			hasPR, err := clientWithPR.HasPullRequest(ctx, 2)
			
			Expect(err).NotTo(HaveOccurred())
			Expect(hasPR).To(BeTrue())
		})
	})
})
```

## Step 4: Dependencies

**Add to `go.mod`**:

```bash
go get github.com/google/go-github/v57@latest
go get golang.org/x/oauth2@latest
go get github.com/migueleliasweb/go-github-mock@latest
```

## Verification

```bash
# Test
go test ./internal/github/... -v

# Lint
make lint

# Build check
go build ./internal/github/...
```
