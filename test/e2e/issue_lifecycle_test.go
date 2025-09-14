package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/utils"
)

var _ = Describe("GitHub Issue Creation and Update", func() {
	var (
		crName    string
		namespace string
		timestamp int64
	)

	Context("When testing issue creation and update", Ordered, func() {
		var (
			githubIssue  *githubv1alpha1.GithubIssue
			initialTitle string
			updatedTitle string
		)

		BeforeAll(func() {
			timestamp = time.Now().UnixNano()
			crName = fmt.Sprintf("update-test-%d", timestamp)
			namespace = fmt.Sprintf("test-ns-%d", timestamp)

			By("Creating test namespace")
			createNamespace(namespace)

			initialTitle = fmt.Sprintf("E2E-Test-Initial-%d", timestamp)
			updatedTitle = fmt.Sprintf("E2E-Test-Updated-%d", timestamp)
			githubIssue = newGithubIssue(crName, namespace, initialTitle, nil)

			By("Creating CR")
			Expect(k8sClient.Create(ctx, githubIssue)).To(Succeed())
		})

		AfterAll(func() {
			By("Cleaning up test namespace")
			deleteNamespace(namespace)
		})

		It("Should create GitHub issue", func() {
			By("Waiting for issue creation")
			Eventually(func() bool {
				return hasConditionWithStatus(githubIssue, metav1.ConditionTrue, utils.ReasonIssueCreated)
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
				if !hasConditionWithStatus(githubIssue, metav1.ConditionTrue, utils.ReasonIssueSynchronized) {
					return false
				}
				return githubIssue.Status.LastSyncTime != nil &&
					githubIssue.Status.LastSyncTime.After(initialSyncTime.Time)
			}, time.Minute*2, time.Second*5).Should(BeTrue(), "Should complete update with new sync time")
		})
	})

})
