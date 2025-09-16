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
	return fmt.Sprintf("github api error: status code %d: %s", e.StatusCode, e.Message)
}

type Client struct {
	githubClient *github.Client
	repo         *Repository
}

func NewClient(token, repoURL string) (*Client, error) {
	repo, err := parseRepositoryURL(repoURL)
	if err != nil {
		return nil, err
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), tokenSource)
	githubClient := github.NewClient(httpClient)

	return &Client{
		githubClient: githubClient,
		repo:         repo,
	}, nil
}

func (c *Client) ValidateAuthentication(ctx context.Context) error {
	_, _, err := c.githubClient.Users.Get(ctx, "")
	if err != nil {
		return c.handleError(err)
	}
	return nil
}

func parseRepositoryURL(repoURL string) (*Repository, error) {
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

func newGitHubError(statusCode int, message string) *GitHubError {
	isRetryable := statusCode >= http.StatusInternalServerError || statusCode == http.StatusTooManyRequests
	return &GitHubError{
		StatusCode:  statusCode,
		Message:     message,
		IsRetryable: isRetryable,
	}
}

func (c *Client) handleError(err error) error {
	if githubError, ok := err.(*github.ErrorResponse); ok {
		errorMessage := "unknown error"
		if githubError.Message != "" {
			errorMessage = githubError.Message
		}
		return newGitHubError(githubError.Response.StatusCode, errorMessage)
	}

	return &GitHubError{
		StatusCode:  0,
		Message:     err.Error(),
		IsRetryable: false,
	}
}

func (c *Client) GetIssueByID(ctx context.Context, issueNumber int) (*github.Issue, error) {
	issue, _, err := c.githubClient.Issues.Get(ctx, c.repo.Owner, c.repo.Name, issueNumber)
	if err != nil {
		return nil, c.handleError(err)
	}

	return issue, nil
}

func (c *Client) GetIssueByTitle(ctx context.Context, title string) (*github.Issue, error) {
	query := fmt.Sprintf(`"%s" in:title repo:%s/%s is:issue`,
		title, c.repo.Owner, c.repo.Name)

	result, _, err := c.githubClient.Search.Issues(ctx, query, &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 1},
	})
	if err != nil {
		return nil, c.handleError(err)
	}

	if len(result.Issues) == 0 {
		return nil, nil
	}

	return result.Issues[0], nil
}

func newIssueRequest(title, description string) *github.IssueRequest {
	issueRequest := &github.IssueRequest{
		Title: &title,
	}

	if description != "" {
		issueRequest.Body = &description
	}

	return issueRequest
}

func (c *Client) CreateIssue(ctx context.Context, title, description string) (*github.Issue, error) {
	issueRequest := newIssueRequest(title, description)

	issue, _, err := c.githubClient.Issues.Create(ctx, c.repo.Owner, c.repo.Name, issueRequest)
	if err != nil {
		return nil, c.handleError(err)
	}

	return issue, nil
}

func (c *Client) UpdateIssue(ctx context.Context, issueNumber int, title, description string) (*github.Issue, error) {
	issueRequest := newIssueRequest(title, description)

	issue, _, err := c.githubClient.Issues.Edit(ctx, c.repo.Owner, c.repo.Name, issueNumber, issueRequest)
	if err != nil {
		return nil, c.handleError(err)
	}

	return issue, nil
}

func (c *Client) OpenIssue(ctx context.Context, issueNumber int) (*github.Issue, error) {
	state := IssueStateOpen
	issueRequest := &github.IssueRequest{State: &state}

	issue, _, err := c.githubClient.Issues.Edit(ctx, c.repo.Owner, c.repo.Name, issueNumber, issueRequest)
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

	issue, _, err := c.githubClient.Issues.Edit(ctx, c.repo.Owner, c.repo.Name, issueNumber, issueRequest)
	if err != nil {
		return nil, c.handleError(err)
	}

	return issue, nil
}

func (c *Client) HasPullRequest(ctx context.Context, issueNumber int) (bool, error) {
	issue, _, err := c.githubClient.Issues.Get(ctx, c.repo.Owner, c.repo.Name, issueNumber)
	if err != nil {
		return false, c.handleError(err)
	}

	return issue.PullRequestLinks != nil, nil
}

func (c *Client) AddLabelsToIssue(ctx context.Context, issueNumber int, labels []string) error {
	_, _, err := c.githubClient.Issues.AddLabelsToIssue(ctx, c.repo.Owner, c.repo.Name, issueNumber, labels)
	if err != nil {
		return c.handleError(err)
	}
	return nil
}

func (c *Client) ListIssueLabels(ctx context.Context, issueNumber int) ([]string, error) {
	issue, _, err := c.githubClient.Issues.Get(ctx, c.repo.Owner, c.repo.Name, issueNumber)
	if err != nil {
		return nil, c.handleError(err)
	}

	return extractLabelNames(issue.Labels), nil
}

func (c *Client) RemoveLabelFromIssue(ctx context.Context, issueNumber int, labelName string) error {
	_, err := c.githubClient.Issues.RemoveLabelForIssue(ctx, c.repo.Owner, c.repo.Name, issueNumber, labelName)
	if err != nil {
		return c.handleError(err)
	}
	return nil
}

func extractLabelNames(labels []*github.Label) []string {
	labelNames := make([]string, len(labels))
	for i, label := range labels {
		labelNames[i] = label.GetName()
	}
	return labelNames
}
