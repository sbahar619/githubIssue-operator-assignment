package e2e

import (
	"context"
	"fmt"
	"slices"
	"time"

	. "github.com/onsi/gomega" //nolint:revive,staticcheck
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
)

const (
	// Operator configuration
	operatorNamespace       = "github-issue-operator-system"
	deploymentName          = "github-issue-operator-controller-manager"
	secretName              = "github-issue-operator-token-secret"
	controllerContainerName = "manager"

	// Test configuration
	githubRepoURL     = "https://github.com/sbahar619/githubIssue-operator-assignment"
	invalidTokenValue = "invalid-token-value"
	timeout           = time.Minute * 2
	pollInterval      = time.Second * 5
)

var (
	k8sClient client.Client
	ctx       context.Context
)

func createNamespace(name string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	_ = k8sClient.Create(ctx, ns)
}

func deleteNamespace(name string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	_ = k8sClient.Delete(ctx, ns)
}

func getOperatorDeployment() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{Name: deploymentName, Namespace: operatorNamespace}
	err := k8sClient.Get(ctx, key, deployment)
	return deployment, err
}

func isControllerReady() bool {
	deployment, err := getOperatorDeployment()
	if err != nil {
		return false
	}
	return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas
}

func hasConditionWithStatus(
	cr *githubv1alpha1.GithubIssue,
	expectedStatus metav1.ConditionStatus,
	reasons ...string,
) bool {
	condition := getReadyCondition(cr)
	return condition != nil &&
		condition.Status == expectedStatus &&
		slices.Contains(reasons, condition.Reason)
}

func getReadyCondition(cr *githubv1alpha1.GithubIssue) *metav1.Condition {
	key := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
	err := k8sClient.Get(ctx, key, cr)
	if err != nil {
		return nil
	}

	return meta.FindStatusCondition(cr.Status.Conditions, "Ready")
}

func newGithubIssue(name, namespace, title string, description *string) *githubv1alpha1.GithubIssue {
	return &githubv1alpha1.GithubIssue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: githubv1alpha1.GithubIssueSpec{
			Title:       title,
			Description: description,
			Repo:        githubRepoURL,
		},
	}
}

func createGithubIssue(name, namespace string) *githubv1alpha1.GithubIssue {
	title := fmt.Sprintf("auth-%d", time.Now().Unix())
	description := "E2E authentication test"
	cr := newGithubIssue(name, namespace, title, &description)
	Expect(k8sClient.Create(ctx, cr)).To(Succeed())
	return cr
}

func createTokenSecret(secretName, tokenValue string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: operatorNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"token": tokenValue,
		},
	}
	Expect(k8sClient.Create(ctx, secret)).To(Succeed())
}

func deleteTokenSecret(secretName string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: operatorNamespace,
		},
	}
	_ = k8sClient.Delete(ctx, secret)
}

func updateControllerSecret(secretName string) {
	Eventually(func() error {
		deployment, err := getOperatorDeployment()
		if err != nil {
			return err
		}

		deployment.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Name = secretName

		return k8sClient.Update(ctx, deployment)
	}, time.Second*30, time.Second*2).Should(Succeed())
}

func waitForControllerReady() {
	Eventually(func() bool {
		return isControllerReady()
	}, time.Minute*3, time.Second*10).Should(BeTrue())
}
