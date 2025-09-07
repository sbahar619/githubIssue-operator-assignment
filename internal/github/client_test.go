package github

import (
	"context"
	"testing"

	"github.com/google/go-github/v57/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	githubRepoURL = "https://github.com/owner/repo"
	repoOwner     = "owner"
	repoName      = "repo"
	existingIssue = "Test Issue"
	newIssue      = "New Issue"
	updatedIssue  = "Updated Issue"
	missingIssue  = "Non-existing Issue"
	malformedURL  = "://invalid-url"
	githubToken   = "test-token"

	// Issue state constants (duplicated from main package for tests)
	issueStateOpen   = "open"
	issueStateClosed = "closed"
)

func TestGitHub(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GitHub Suite")
}

var _ = Describe("Repository URL Parsing", func() {
	Describe("ParseRepositoryURL", func() {
		It("should parse valid GitHub URL", func() {
			repo, err := ParseRepositoryURL(githubRepoURL)

			Expect(err).NotTo(HaveOccurred())
			Expect(repo.Owner).To(Equal(repoOwner))
			Expect(repo.Name).To(Equal(repoName))
		})

		It("should handle URL parsing errors", func() {
			_, err := ParseRepositoryURL(malformedURL)

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
			client, err := NewClient(githubToken, githubRepoURL)

			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
			Expect(client.repo.Owner).To(Equal(repoOwner))
			Expect(client.repo.Name).To(Equal(repoName))
		})

		It("should return error with invalid repository URL", func() {
			_, err := NewClient(githubToken, malformedURL)

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
					{ID: github.Int64(1), Title: github.String(existingIssue), State: github.String(issueStateOpen)},
				},
			),
			mock.WithRequestMatch(
				mock.PostReposIssuesByOwnerByRepo,
				github.Issue{ID: github.Int64(2), Title: github.String(newIssue), State: github.String(issueStateOpen)},
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
			Expect(issues[0].GetTitle()).To(Equal(existingIssue))
		})
	})

	Describe("GetIssueByTitle", func() {
		It("should get existing issue", func() {
			issue, err := client.GetIssueByTitle(ctx, existingIssue)

			Expect(err).NotTo(HaveOccurred())
			Expect(issue).NotTo(BeNil())
			Expect(issue.GetTitle()).To(Equal(existingIssue))
		})

		It("should return nil for non-existing issue", func() {
			issue, err := client.GetIssueByTitle(ctx, missingIssue)

			Expect(err).NotTo(HaveOccurred())
			Expect(issue).To(BeNil())
		})
	})

	Describe("CreateIssue", func() {
		It("should create new issue", func() {
			issue, err := client.CreateIssue(ctx, newIssue, "Description")

			Expect(err).NotTo(HaveOccurred())
			Expect(issue).NotTo(BeNil())
			Expect(issue.GetTitle()).To(Equal(newIssue))
		})
	})
})
