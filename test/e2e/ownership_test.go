package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sbahar619/githubIssue-operator-assignment/internal/github"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/utils"
)

var _ = Describe("GitHub Issue Ownership", func() {
	var namespace string
	var timestamp int64

	BeforeEach(func() {
		timestamp = time.Now().Unix()
		namespace = fmt.Sprintf("test-ns-%d", timestamp)

		By("Creating test namespace")
		createNamespace(namespace)
	})

	AfterEach(func() {
		By("Cleaning up test namespace")
		deleteNamespace(namespace)
	})

	It("Should detect externally owned issue", func() {
		title := fmt.Sprintf("External-Issue-Test-%d", timestamp)

		By("Creating GitHub issue directly without operator labels")
		token, err := getOperatorToken()
		Expect(err).NotTo(HaveOccurred())
		githubClient, err := github.NewClient(token, githubRepoURL)
		Expect(err).NotTo(HaveOccurred())

		issue, err := githubClient.CreateIssue(ctx, title, "External issue")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(githubClient.CloseIssue, ctx, issue.GetNumber())

		By("Waiting for GitHub search API to index the new issue")
		time.Sleep(time.Second * 10)

		By("Creating CR with same title")
		crName := fmt.Sprintf("external-test-%d", timestamp)
		cr := newGithubIssue(crName, namespace, title)
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())

		By("Expecting external ownership detection")
		Eventually(func() bool {
			return hasConditionWithReason(cr, utils.ReasonExternallyOwnedIssue)
		}, timeout, pollInterval).Should(BeTrue())
	})

	It("Should detect conflicted ownership when issue owned by different CR", func() {
		title := fmt.Sprintf("Conflicted-Ownership-Test-%d", timestamp)

		By("Creating first CR to establish ownership")
		ownerCR := newGithubIssue("owner-cr", namespace, title)
		Expect(k8sClient.Create(ctx, ownerCR)).To(Succeed())

		By("Waiting for owner CR to create the issue successfully")
		Eventually(func() bool {
			return hasConditionWithReason(ownerCR, utils.ReasonIssueCreated)
		}, timeout, pollInterval).Should(BeTrue())

		By("Creating conflicting CR with same title but different name")
		conflictCR := newGithubIssue("conflict-cr", namespace, title)
		Expect(k8sClient.Create(ctx, conflictCR)).To(Succeed())

		By("Expecting conflicted ownership detection")
		Eventually(func() bool {
			return hasConditionWithReason(conflictCR, utils.ReasonConflictedOwnership)
		}, timeout, pollInterval).Should(BeTrue())
	})
})
