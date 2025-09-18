package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sbahar619/githubIssue-operator-assignment/internal/controller"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/github"
)

var _ = Describe("Issue Ownership Management", func() {
	var namespace string
	var timestamp int64

	BeforeEach(func() {
		timestamp = time.Now().Unix()
		namespace = fmt.Sprintf("test-ns-%d", timestamp)

		By("Setup test environment")
		createNamespace(namespace)
	})

	AfterEach(func() {
		By("Clean up test resources")
		deleteNamespace(namespace)
	})

	It("Should respect existing GitHub issues created outside the operator", func() {
		title := fmt.Sprintf("Manual-Issue-%d", timestamp)

		By("Create external GitHub issue")
		token, err := getOperatorToken()
		Expect(err).NotTo(HaveOccurred())
		githubClient, err := github.NewClient(token, githubRepoURL)
		Expect(err).NotTo(HaveOccurred())

		issue, err := githubClient.CreateIssue(ctx, title, "This issue was created manually by a user")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(githubClient.CloseIssue, ctx, issue.GetNumber())

		By("Verify external issue exists")
		Eventually(func() bool {
			foundIssue, err := githubClient.GetIssueByTitle(ctx, title)
			return err == nil && foundIssue != nil
		}, timeout, pollInterval).Should(BeTrue())

		By("Create GithubIssue resource with same title")
		crName := fmt.Sprintf("manual-issue-%d", timestamp)
		cr := newGithubIssue(crName, namespace, title)
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())

		By("Verify external ownership is detected")
		Eventually(func() bool {
			return hasConditionWithReason(cr, controller.ReasonExternallyOwnedIssue)
		}, timeout, pollInterval).Should(BeTrue())
	})

	It("Should prevent multiple resources from claiming the same GitHub issue", func() {
		title := fmt.Sprintf("Shared-Issue-Title-%d", timestamp)

		By("Create first GithubIssue resource")
		firstResource := newGithubIssue("team-a-issue", namespace, title)
		Expect(k8sClient.Create(ctx, firstResource)).To(Succeed())

		By("Verify first issue is created successfully")
		Eventually(func() bool {
			return hasConditionWithReason(firstResource, controller.ReasonIssueCreated)
		}, timeout, pollInterval).Should(BeTrue())

		By("Create second GithubIssue resource with same title")
		secondResource := newGithubIssue("team-b-issue", namespace, title)
		Expect(k8sClient.Create(ctx, secondResource)).To(Succeed())

		By("Verify ownership conflict is detected")
		Eventually(func() bool {
			return hasConditionWithReason(secondResource, controller.ReasonConflictedOwnership)
		}, timeout, pollInterval).Should(BeTrue())
	})

	It("Should allow recreation after deletion without ownership conflicts", func() {
		title := fmt.Sprintf("Recreate-Test-%d", timestamp)

		By("Create first GithubIssue resource")
		firstCR := newGithubIssue("recreate-test", namespace, title)
		Expect(k8sClient.Create(ctx, firstCR)).To(Succeed())

		By("Verify first issue is created successfully")
		Eventually(func() bool {
			return hasConditionWithReason(firstCR, controller.ReasonIssueCreated)
		}, timeout, pollInterval).Should(BeTrue())

		By("Delete the GithubIssue resource")
		Expect(k8sClient.Delete(ctx, firstCR)).To(Succeed())

		By("Verify resource is fully deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(firstCR), firstCR)
			return err != nil
		}, timeout, pollInterval).Should(BeTrue())

		By("Create new GithubIssue resource with same title")
		secondCR := newGithubIssue("recreate-test-2", namespace, title)
		Expect(k8sClient.Create(ctx, secondCR)).To(Succeed())

		By("Verify successful creation without ownership conflicts")
		Eventually(func() bool {
			return hasConditionWithReason(secondCR, controller.ReasonIssueSynchronized)
		}, timeout, pollInterval).Should(BeTrue())
	})

	It("Should allow recreation after full lifecycle without ownership conflicts", func() {
		title := fmt.Sprintf("Lifecycle-Test-%d", timestamp)
		updatedTitle := fmt.Sprintf("Lifecycle-Updated-%d", timestamp)

		By("Create first GithubIssue resource")
		firstCR := newGithubIssue("lifecycle-test", namespace, title)
		Expect(k8sClient.Create(ctx, firstCR)).To(Succeed())

		By("Verify first issue is created successfully")
		Eventually(func() bool {
			return hasConditionWithReason(firstCR, controller.ReasonIssueCreated)
		}, timeout, pollInterval).Should(BeTrue())

		By("Update the GithubIssue resource")
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(firstCR), firstCR)).To(Succeed())
		firstCR.Spec.Title = updatedTitle
		Expect(k8sClient.Update(ctx, firstCR)).To(Succeed())

		By("Verify issue is updated successfully")
		Eventually(func() bool {
			return hasConditionWithReason(firstCR, controller.ReasonIssueSynchronized)
		}, timeout, pollInterval).Should(BeTrue())

		By("Delete the GithubIssue resource")
		Expect(k8sClient.Delete(ctx, firstCR)).To(Succeed())

		By("Verify resource is fully deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(firstCR), firstCR)
			return err != nil
		}, timeout, pollInterval).Should(BeTrue())

		By("Create new GithubIssue resource with updated title")
		secondCR := newGithubIssue("lifecycle-test-2", namespace, updatedTitle)
		Expect(k8sClient.Create(ctx, secondCR)).To(Succeed())

		By("Verify successful creation without ownership conflicts")
		Eventually(func() bool {
			return hasConditionWithReason(secondCR, controller.ReasonIssueSynchronized)
		}, timeout, pollInterval).Should(BeTrue())
	})
})
