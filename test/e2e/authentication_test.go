/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/utils"
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
	clientset *kubernetes.Clientset
	ctx       context.Context
)

var _ = BeforeEach(func() {
	if k8sClient == nil {
		ctx = context.Background()

		config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		).ClientConfig()
		Expect(err).NotTo(HaveOccurred(), "Should build kubeconfig")

		clientset, err = kubernetes.NewForConfig(config)
		Expect(err).NotTo(HaveOccurred(), "Should create clientset")

		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(appsv1.AddToScheme(scheme)).To(Succeed())
		Expect(githubv1alpha1.AddToScheme(scheme)).To(Succeed())

		k8sClient, err = client.New(config, client.Options{Scheme: scheme})
		Expect(err).NotTo(HaveOccurred(), "Should create controller-runtime client")
	}
})

var _ = Describe("GitHub Token Authentication", func() {
	var (
		crName    string
		namespace string
		timestamp int64
	)

	BeforeEach(func() {
		timestamp = time.Now().UnixNano()
		crName = fmt.Sprintf("auth-%d", timestamp)
		namespace = fmt.Sprintf("test-ns-%d", timestamp)

		By("Creating test namespace")
		createNamespace(namespace)
	})

	AfterEach(func() {
		By("Cleaning up test resources")
		deleteNamespace(namespace)

		By("Ensuring controller is restored to valid configuration")
		updateControllerSecret(secretName)
		waitForControllerReady()
	})

	Context("Token Retrieval Error", func() {
		It("should handle empty token gracefully", func() {
			emptySecretName := fmt.Sprintf("github-issue-operator-secret-empty-%d", timestamp)

			By("Applying empty token secret")
			createTokenSecret(emptySecretName, "")

			By("Updating controller secret")
			updateControllerSecret(emptySecretName)

			By("Waiting for controller to restart")
			waitForControllerReady()

			By("Creating GithubIssue CR")
			cr := createGithubIssue(crName, namespace)

			By("Verifying token retrieval failure is handled")
			Eventually(func() bool {
				return hasConditionWithReason(cr, utils.ReasonAuthenticationFailed)
			}, timeout, pollInterval).Should(BeTrue())

			By("Cleaning up empty token secret")
			deleteTokenSecret(emptySecretName)
		})
	})

	Context("Invalid Token Authentication", func() {
		var (
			invalidSecretName string
		)

		It("should handle invalid token gracefully", func() {
			invalidSecretName = fmt.Sprintf("github-issue-operator-secret-invalid-%d", timestamp)

			By("Applying invalid token secret")
			createTokenSecret(invalidSecretName, invalidTokenValue)

			By("Updating controller secret")
			updateControllerSecret(invalidSecretName)

			By("Waiting for controller to restart")
			waitForControllerReady()

			By("Creating GithubIssue CR")
			cr := createGithubIssue(crName, namespace)

			By("Verifying authentication failure is handled")
			Eventually(func() bool {
				return hasConditionWithReason(cr, utils.ReasonAuthenticationFailed, utils.ReasonGitHubAPIError)
			}, timeout, pollInterval).Should(BeTrue())

			By("Restoring original controller configuration")
			updateControllerSecret(secretName)
			deleteTokenSecret(invalidSecretName)
			waitForControllerReady()
		})
	})
})

func createNamespace(name string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
}

func deleteNamespace(name string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	_ = k8sClient.Delete(ctx, ns)
}

func createGithubIssue(name, namespace string) *githubv1alpha1.GithubIssue {
	cr := &githubv1alpha1.GithubIssue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: githubv1alpha1.GithubIssueSpec{
			Repo:  githubRepoURL,
			Title: fmt.Sprintf("auth-%d", time.Now().Unix()),
			Description: func() *string {
				desc := "E2E authentication test"
				return &desc
			}(),
		},
	}
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

func getOperatorDeployment() (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{Name: deploymentName, Namespace: operatorNamespace}
	err := k8sClient.Get(ctx, key, deployment)
	return deployment, err
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

func hasConditionWithReason(cr *githubv1alpha1.GithubIssue, reasons ...string) bool {
	key := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
	err := k8sClient.Get(ctx, key, cr)
	if err != nil {
		return false
	}

	condition := meta.FindStatusCondition(cr.Status.Conditions, utils.ConditionTypeReady)
	if condition == nil || condition.Status != metav1.ConditionFalse {
		return false
	}

	return slices.Contains(reasons, condition.Reason)
}

func waitForControllerReady() {
	Eventually(func() bool {
		return isControllerReady()
	}, time.Minute*3, time.Second*10).Should(BeTrue())
}

func isControllerReady() bool {
	deployment, err := getOperatorDeployment()
	if err != nil {
		return false
	}
	return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas
}
