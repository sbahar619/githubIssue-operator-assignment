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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/controller"
)

var _ = Describe("GitHub Token Authentication", func() {
	var (
		crName    string
		namespace string
		timestamp int64
	)

	BeforeEach(func() {
		timestamp = time.Now().Unix()
		crName = fmt.Sprintf("auth-%d", timestamp)
		namespace = fmt.Sprintf("test-ns-%d", timestamp)

		By("Creating test namespace")
		createNamespace(namespace)
	})

	AfterEach(func() {
		By("Cleaning up test resources")
		deleteNamespace(namespace)

		By("Ensuring controller is restored to valid configuration")
		updateAuthConfiguration(secretName)
	})

	Context("when GitHub token is not configured properly", func() {
		It("should report authentication failure when token is missing", func() {
			emptySecretName := fmt.Sprintf("github-issue-operator-secret-empty-%d", timestamp)

			By("Configuring operator without GitHub token")
			createTokenSecret(emptySecretName, "")
			DeferCleanup(deleteTokenSecret, emptySecretName)

			By("Applying the empty token configuration")
			updateAuthConfiguration(emptySecretName)

			By("Creating a GitHub issue request")
			cr := createGithubIssue(crName, namespace)

			By("Expecting authentication failure to be reported")
			Eventually(func() bool {
				return hasConditionWithReason(cr, controller.ReasonAuthenticationFailed)
			}, timeout, pollInterval).Should(BeTrue())
		})

		It("should report GitHub API error when token is invalid", func() {
			invalidToken := "invalid_token_12345"
			invalidSecretName := fmt.Sprintf("github-issue-operator-secret-invalid-%d", timestamp)

			By("Configuring operator with invalid GitHub token")
			createTokenSecret(invalidSecretName, invalidToken)
			DeferCleanup(deleteTokenSecret, invalidSecretName)

			By("Applying the invalid token configuration")
			updateAuthConfiguration(invalidSecretName)

			By("Creating a GitHub issue request")
			cr := createGithubIssue(crName, namespace)

			By("Expecting GitHub API error to be reported")
			Eventually(func() bool {
				return hasConditionWithReason(cr, controller.ReasonGitHubAPIError)
			}, timeout, pollInterval).Should(BeTrue())
		})
	})
})

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
			tokenSecretKey: tokenValue,
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

func updateDeploymentSecret(secretName string) {
	Eventually(func() error {
		deployment, err := getOperatorDeployment()
		if err != nil {
			return err
		}

		deployment.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Name = secretName
		return k8sClient.Update(ctx, deployment)
	}, time.Second*10, time.Second*1).Should(Succeed())
}

func waitForSecretUpdate(secretName string) {
	Eventually(func() bool {
		current, _ := getOperatorDeployment()
		return current.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Name == secretName
	}, time.Second*30, time.Second*2).Should(BeTrue())
}

func updateAuthConfiguration(secretName string) {
	updateDeploymentSecret(secretName)
	time.Sleep(time.Second * 15)
	waitForSecretUpdate(secretName)
	waitForControllerReady()
}

func waitForControllerReady() {
	Eventually(func() bool {
		return isControllerReady()
	}, time.Minute*2, time.Second*10).Should(BeTrue())
}
