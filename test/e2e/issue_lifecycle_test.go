package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/github"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/utils"
)

var _ = Describe("GitHub Issue Lifecycle (Create, Update, Delete)", Ordered, func() {
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
		}, time.Minute*2, time.Second*5).Should(BeTrue(), "GitHub issue should be closed")

		By("Waiting for CR deletion and finalizer cleanup")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(githubIssue), githubIssue)
			return err != nil
		}, time.Minute*2, time.Second*5).Should(BeTrue(), "CR should be deleted")
	})

})

func newGithubIssue(name, namespace, title string, description *string) *githubv1alpha1.GithubIssue {
	return &githubv1alpha1.GithubIssue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: githubv1alpha1.GithubIssueSpec{
			Repo:        githubRepoURL,
			Title:       title,
			Description: description,
		},
	}
}

func getOperatorToken() (string, error) {
	deployment, err := getOperatorDeployment()
	if err != nil {
		return "", err
	}

	secretName := deployment.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Name

	secret := &corev1.Secret{}
	key := types.NamespacedName{Name: secretName, Namespace: operatorNamespace}
	if err := k8sClient.Get(ctx, key, secret); err != nil {
		return "", err
	}

	tokenBytes, exists := secret.Data[tokenSecretKey]
	if !exists {
		return "", fmt.Errorf("token not found in secret")
	}

	return string(tokenBytes), nil
}

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
