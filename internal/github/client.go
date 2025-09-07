package github

import (
	"context"
	"fmt"
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
