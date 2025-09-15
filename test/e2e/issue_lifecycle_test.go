package e2e

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/utils"
)

var _ = Describe("GitHub Issue Creation and Update", Ordered, func() {
	var crName, namespace, initialTitle, updatedTitle string
	var githubIssue *githubv1alpha1.GithubIssue

	BeforeAll(func() {
		crName = "update-test"
		namespace = "test-ns"
		initialTitle = "E2E-Test-Initial"
		updatedTitle = "E2E-Test-Updated"

		By("Creating test namespace")
		createNamespace(namespace)

		By("Creating GitHub issue CR")
		githubIssue = newGithubIssue(crName, namespace, initialTitle, nil)
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
		}, time.Minute*2, time.Second*5).Should(BeTrue(), "Should create new issue")
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
		}, time.Minute*2, time.Second*5).Should(BeTrue(), "Should complete update with new sync time")
	})

})
