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
	tokenSecretKey          = "token"

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
	var deployment *appsv1.Deployment
	var err error

	Eventually(func() error {
		deployment = &appsv1.Deployment{}
		key := types.NamespacedName{Name: deploymentName, Namespace: operatorNamespace}
		err = k8sClient.Get(ctx, key, deployment)
		return err
	}, time.Second*30, time.Second*2).Should(Succeed())

	return deployment, err
}

func isControllerReady() bool {
	deployment, err := getOperatorDeployment()
	if err != nil {
		return false
	}
	return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas
}

func hasConditionWithReason(cr *githubv1alpha1.GithubIssue, reasons ...string) bool {
	key := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
	err := k8sClient.Get(ctx, key, cr)
	if err != nil {
		return false
	}

	condition := meta.FindStatusCondition(cr.Status.Conditions, "Ready")
	if condition == nil {
		return false
	}

	return slices.Contains(reasons, condition.Reason)
}

func newGithubIssue(name, namespace, title string) *githubv1alpha1.GithubIssue {
	return &githubv1alpha1.GithubIssue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: githubv1alpha1.GithubIssueSpec{
			Repo:  githubRepoURL,
			Title: title,
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
