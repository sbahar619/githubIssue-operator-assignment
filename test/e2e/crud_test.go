package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/github"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/utils"
)

var _ = Describe("GitHub Issue Lifecycle (Create, Update, Delete)", Ordered, func() {
	var crName, namespace, initialTitle, updatedTitle string
	var githubIssue *githubv1alpha1.GithubIssue

	BeforeAll(func() {
		timestamp := time.Now().Unix()
		crName = fmt.Sprintf("update-test-%d", timestamp)
		namespace = fmt.Sprintf("test-ns-%d", timestamp)
		initialTitle = fmt.Sprintf("E2E-Test-Initial-%d", timestamp)
		updatedTitle = fmt.Sprintf("E2E-Test-Updated-%d", timestamp)

		By("Creating test namespace")
		createNamespace(namespace)

		By("Creating GitHub issue CR")
		githubIssue = newGithubIssue(crName, namespace, initialTitle)
		Expect(k8sClient.Create(ctx, githubIssue)).To(Succeed())
	})

	AfterAll(func() {
		By("Cleaning up test namespace")
		deleteNamespace(namespace)
	})

	It("Should create GitHub issue", func() {
		By("Waiting for issue creation")
		Eventually(func() bool {
			return hasConditionWithReason(githubIssue, utils.ReasonIssueCreated)
		}, timeout, pollInterval).Should(BeTrue(), "Should create new issue")
	})

	It("Should update GitHub issue when CR changes", func() {
		By("Capturing initial sync time before update")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(githubIssue), githubIssue)).To(Succeed())
		initialSyncTime := githubIssue.Status.LastSyncTime
		Expect(initialSyncTime).NotTo(BeNil(), "Should have initial sync time")

		By("Updating CR spec to trigger update")
		githubIssue.Spec.Title = updatedTitle
		Expect(k8sClient.Update(ctx, githubIssue)).To(Succeed())

		By("Waiting for update completion and verifying sync time changed")
		Eventually(func() bool {
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(githubIssue), githubIssue); err != nil {
				return false
			}
			if !hasConditionWithReason(githubIssue, utils.ReasonIssueSynchronized) {
				return false
			}
			return githubIssue.Status.LastSyncTime != nil &&
				githubIssue.Status.LastSyncTime.After(initialSyncTime.Time)
		}, timeout, pollInterval).Should(BeTrue(), "Should complete update with new sync time")
	})

	It("Should close GitHub issue when CR is deleted", func() {
		By("Capturing GitHub issue ID before deletion")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(githubIssue), githubIssue)).To(Succeed())
		Expect(githubIssue.Status.IssueID).NotTo(BeNil(), "Should have GitHub issue ID")
		issueID := *githubIssue.Status.IssueID

		By("Deleting the CR")
		Expect(k8sClient.Delete(ctx, githubIssue)).To(Succeed())

		By("Waiting for GitHub issue to be closed")
		Eventually(func() bool {
			return isGitHubIssueClosed(issueID)
		}, timeout, pollInterval).Should(BeTrue(), "GitHub issue should be closed")

		By("Waiting for CR deletion and finalizer cleanup")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(githubIssue), githubIssue)
			return err != nil
		}, timeout, pollInterval).Should(BeTrue(), "CR should be deleted")
	})

})

func isGitHubIssueClosed(issueID int) bool {
	token, err := getOperatorToken()
	if err != nil {
		return false
	}

	githubClient, err := github.NewClient(token, githubRepoURL)
	if err != nil {
		return false
	}

	issue, err := githubClient.GetIssueByID(ctx, issueID)
	if err != nil {
		return false
	}

	return issue.GetState() == github.IssueStateClosed
}
