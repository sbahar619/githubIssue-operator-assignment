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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	appsv1 "k8s.io/api/apps/v1"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/utils"
)

const (
	// Operator configuration
	operatorNamespace          = "github-issue-operator-system"
	deploymentName             = "github-issue-operator-controller-manager"
	secretName                = "github-issue-operator-token-secret"
	controllerContainerName   = "manager"
	
	// Test configuration
	githubRepoURL             = "https://github.com/sbahar619/githubIssue-operator-assignment"
	invalidTokenValue         = "invalid-token-value"
	timeout                   = time.Minute * 2
	pollInterval              = time.Second * 5
)

var (
	k8sClient   client.Client
	clientset   *kubernetes.Clientset
	ctx         context.Context
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
	)

	BeforeEach(func() {
		timestamp := time.Now().Unix()
		crName = fmt.Sprintf("auth-%d", timestamp)
		namespace = fmt.Sprintf("test-ns-%d", timestamp)

		By("Creating test namespace")
		createNamespace(namespace)
	})

	AfterEach(func() {
		By("Cleaning up test resources")
		cleanupNamespace(namespace)
	})

	Context("Invalid Token Authentication", func() {
		var (
			originalDeployment *appsv1.Deployment
			invalidSecretName  string
		)
		
		It("should handle invalid token gracefully", func() {
			timestamp := time.Now().Unix()
			invalidSecretName = fmt.Sprintf("github-issue-operator-secret-invalid-%d", timestamp)

			By("Backing up original controller deployment")
			originalDeployment = backupControllerDeployment()

			By("Creating invalid token secret")
			createInvalidTokenSecret(invalidSecretName)

			By("Patching controller to use invalid token secret")
			patchControllerToUseInvalidSecret(invalidSecretName)

			By("Waiting for controller to restart")
			waitForControllerReady()

			By("Creating GithubIssue CR")
			cr := createGithubIssue(crName, namespace)

			By("Verifying authentication failure is handled")
			Eventually(func() bool {
				return hasAuthenticationFailure(cr)
			}, timeout, pollInterval).Should(BeTrue())

			By("Restoring original controller configuration")
			restoreControllerDeployment(originalDeployment)
			cleanupInvalidTokenSecret(invalidSecretName)
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

func cleanupNamespace(name string) {
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

func backupControllerDeployment() *appsv1.Deployment {
	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{Name: deploymentName, Namespace: operatorNamespace}
	Expect(k8sClient.Get(ctx, key, deployment)).To(Succeed())
	return deployment.DeepCopy()
}

func createInvalidTokenSecret(secretName string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: operatorNamespace,
		},
		Type: corev1.SecretTypeOpaque,
	StringData: map[string]string{
		"token": invalidTokenValue,
	},
	}
	
	Expect(k8sClient.Create(ctx, secret)).To(Succeed())
}

func patchControllerToUseInvalidSecret(secretName string) {
	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{Name: deploymentName, Namespace: operatorNamespace}
	Expect(k8sClient.Get(ctx, key, deployment)).To(Succeed())

	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		if container.Name == controllerContainerName {
			for j := range container.Env {
				if container.Env[j].Name == auth.GitHubTokenEnvVar {
					container.Env[j].ValueFrom.SecretKeyRef.Name = secretName
					break
				}
			}
			break
		}
	}

	Expect(k8sClient.Update(ctx, deployment)).To(Succeed())
}

func restoreControllerDeployment(original *appsv1.Deployment) {
	if original == nil {
		return
	}
	
	current := &appsv1.Deployment{}
	key := types.NamespacedName{Name: deploymentName, Namespace: operatorNamespace}
	Expect(k8sClient.Get(ctx, key, current)).To(Succeed())
	
	current.Spec = original.Spec
	Expect(k8sClient.Update(ctx, current)).To(Succeed())
}

func cleanupInvalidTokenSecret(secretName string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: operatorNamespace,
		},
	}
	_ = k8sClient.Delete(ctx, secret)
	// Ignore errors - secret might already be deleted or not exist
}

func waitForControllerReady() {
	Eventually(func() bool {
		deployment := &appsv1.Deployment{}
		key := types.NamespacedName{Name: deploymentName, Namespace: operatorNamespace}
		err := k8sClient.Get(ctx, key, deployment)
		if err != nil {
			return false
		}
		return deployment.Status.ReadyReplicas == deployment.Status.Replicas && 
			   deployment.Status.Replicas > 0 &&
			   deployment.Status.UpdatedReplicas == deployment.Status.Replicas
	}, time.Minute*3, time.Second*10).Should(BeTrue())
	
	// Give controller time to settle after restart
	time.Sleep(time.Second * 10)
}

func hasAuthenticationFailure(cr *githubv1alpha1.GithubIssue) bool {
	key := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
	err := k8sClient.Get(ctx, key, cr)
	if err != nil {
		return false
	}
	
	for _, condition := range cr.Status.Conditions {
		if condition.Type == utils.ConditionTypeReady &&
			condition.Status == metav1.ConditionFalse &&
			(condition.Reason == utils.ReasonAuthenticationFailed || 
			 condition.Reason == utils.ReasonGitHubAPIError) {
			return true
		}
	}
	return false
}
