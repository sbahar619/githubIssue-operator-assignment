package github

import (
	"context"
	"testing"

	"github.com/google/go-github/v57/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Shared test constants
const (
	repoURL  = "https://github.com/owner/repo"
	owner    = "owner"
	repoName = "repo"
	token    = "test-token"
)

func TestGitHub(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GitHub Suite")
}

var _ = Describe("Repository URL Parsing", func() {
	Describe("parseRepositoryURL", func() {
		It("should parse valid GitHub URL", func() {
			repo, err := parseRepositoryURL(repoURL)

			Expect(err).NotTo(HaveOccurred())
			Expect(repo.Owner).To(Equal(owner))
			Expect(repo.Name).To(Equal(repoName))
		})

		It("should handle URL parsing errors", func() {
			_, err := parseRepositoryURL("://invalid-url")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse URL"))
		})
	})
})

var _ = Describe("GitHub Error", func() {
	It("should classify server errors as retryable", func() {
		err := newGitHubError(500, "Internal Server Error")

		Expect(err.IsRetryable).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("status code 500"))
	})

	It("should classify rate limit as retryable", func() {
		err := newGitHubError(429, "Rate limit exceeded")

		Expect(err.IsRetryable).To(BeTrue())
	})

	It("should classify client errors as non-retryable", func() {
		err := newGitHubError(404, "Not Found")

		Expect(err.IsRetryable).To(BeFalse())
	})
})

var _ = Describe("Client Constructor", func() {
	Describe("NewClient", func() {
		It("should create client with valid repository URL", func() {
			client, err := NewClient(token, repoURL)

			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
			Expect(client.repo.Owner).To(Equal(owner))
			Expect(client.repo.Name).To(Equal(repoName))
		})

		It("should return error with invalid repository URL", func() {
			_, err := NewClient(token, "://invalid-url")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse URL"))
		})
	})
})

var _ = Describe("GitHub Client Operations", func() {
	const (
		existingTitle    = "Test Issue"
		newTitle         = "New Issue"
		nonExistentTitle = "Non-existing Issue"
		updatedTitle     = "Updated Title"
	)

	var (
		client *Client
		ctx    context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

		mockedHTTPClient := mock.NewMockedHTTPClient(
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
				github.Issue{ID: github.Int64(1), Title: github.String(updatedTitle), State: github.String(IssueStateOpen)},
			),
		)

		githubClient := github.NewClient(mockedHTTPClient)

		client = &Client{
			githubClient: githubClient,
			repo:         &Repository{Owner: owner, Name: repoName},
		}
	})

	Describe("GetIssueByTitle", func() {
		Context("when issue exists", func() {
			var existingClient *Client

			BeforeEach(func() {
				mockedHTTPClientExisting := mock.NewMockedHTTPClient(
					mock.WithRequestMatch(
						mock.GetReposIssuesByOwnerByRepo,
						[]*github.Issue{
							{ID: github.Int64(1), Title: github.String(existingTitle), State: github.String(IssueStateOpen)},
						},
					),
				)

				githubClientExisting := github.NewClient(mockedHTTPClientExisting)
				existingClient = &Client{
					githubClient: githubClientExisting,
					repo:         &Repository{Owner: owner, Name: repoName},
				}
			})

			It("should get existing issue", func() {
				issue, err := existingClient.GetIssueByTitle(ctx, existingTitle)

				Expect(err).NotTo(HaveOccurred())
				Expect(issue).NotTo(BeNil())
				Expect(issue.GetTitle()).To(Equal(existingTitle))
			})
		})

		Context("when issue does not exist", func() {
			var nonExistentClient *Client

			BeforeEach(func() {
				mockedHTTPClientNonExistent := mock.NewMockedHTTPClient(
					mock.WithRequestMatch(
						mock.GetReposIssuesByOwnerByRepo,
						[]*github.Issue{},
					),
				)

				githubClientNonExistent := github.NewClient(mockedHTTPClientNonExistent)
				nonExistentClient = &Client{
					githubClient: githubClientNonExistent,
					repo:         &Repository{Owner: owner, Name: repoName},
				}
			})

			It("should return nil for non-existing issue", func() {
				issue, err := nonExistentClient.GetIssueByTitle(ctx, nonExistentTitle)

				Expect(err).NotTo(HaveOccurred())
				Expect(issue).To(BeNil())
			})
		})
	})

	Describe("CreateIssue", func() {
		It("should create new issue", func() {
			const description = "Description"

			issue, err := client.CreateIssue(ctx, newTitle, description)

			Expect(err).NotTo(HaveOccurred())
			Expect(issue).NotTo(BeNil())
			Expect(issue.GetTitle()).To(Equal(newTitle))
		})
	})

	Describe("UpdateIssue", func() {
		It("should update existing issue", func() {
			const updatedDescription = "Updated Description"

			issue, err := client.UpdateIssue(ctx, 1, updatedTitle, updatedDescription)

			Expect(err).NotTo(HaveOccurred())
			Expect(issue).NotTo(BeNil())
			Expect(issue.GetTitle()).To(Equal(updatedTitle))
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
			const (
				issueWithPRTitle = "Test Issue"
				prURL            = "https://api.github.com/repos/owner/repo/pulls/123"
			)

			mockedHTTPClientWithPR := mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.GetReposIssuesByOwnerByRepoByIssueNumber,
					github.Issue{ID: github.Int64(2), Title: github.String(issueWithPRTitle), State: github.String(IssueStateOpen), PullRequestLinks: &github.PullRequestLinks{URL: github.String(prURL)}},
				),
			)

			githubClientWithPR := github.NewClient(mockedHTTPClientWithPR)
			clientWithPR := &Client{
				githubClient: githubClientWithPR,
				repo:         &Repository{Owner: owner, Name: repoName},
			}

			hasPR, err := clientWithPR.HasPullRequest(ctx, 2)

			Expect(err).NotTo(HaveOccurred())
			Expect(hasPR).To(BeTrue())
		})
	})
})
