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
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
)

var _ = BeforeSuite(func() {
	ctx = context.Background()

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	Expect(err).NotTo(HaveOccurred(), "Should build kubeconfig")

	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(appsv1.AddToScheme(scheme)).To(Succeed())
	Expect(githubv1alpha1.AddToScheme(scheme)).To(Succeed())

	k8sClient, err = client.New(config, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred(), "Should create controller-runtime client")
})

var _ = AfterSuite(func() {
	By("Waiting for any GithubIssue resources with finalizers to complete cleanup")
	Eventually(func() bool {
		var githubIssues githubv1alpha1.GithubIssueList
		if err := k8sClient.List(ctx, &githubIssues); err != nil {
			return false
		}

		// Check if any issues are still being deleted (have finalizers)
		for _, issue := range githubIssues.Items {
			if issue.DeletionTimestamp != nil && len(issue.Finalizers) > 0 {
				_, _ = fmt.Fprintf(GinkgoWriter, "Still waiting for finalizer cleanup: %s/%s\n",
					issue.Namespace, issue.Name)
				return false
			}
		}
		return true
	}, time.Minute*2, time.Second*5).Should(BeTrue(), "All finalizers should be cleaned up")
})

// TestE2E runs the end-to-end (e2e) test suite for GitHub Issue Operator authentication.
// These tests assume the operator is already deployed and running in the cluster.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting GitHub Issue Operator authentication test suite\n")
	RunSpecs(t, "GitHub Issue Operator E2E Authentication Tests")
}
