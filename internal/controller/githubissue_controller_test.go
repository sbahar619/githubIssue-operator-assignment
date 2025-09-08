package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
)

const (
	testIssueName       = "test-issue"
	testNamespace       = "default"
	testRepoURL         = "https://github.com/test/repo"
	testTitle           = "Test Issue"
	defaultRequeueDelay = time.Minute * 1
)

func newTestGithubIssue(name, namespace, repo, title string) *githubv1alpha1.GithubIssue {
	return &githubv1alpha1.GithubIssue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: githubv1alpha1.GithubIssueSpec{
			Repo:  repo,
			Title: title,
		},
	}
}

func getStatusCondition(issueName, namespace string) *metav1.Condition {
	var updatedIssue githubv1alpha1.GithubIssue
	ExpectWithOffset(1, fakeClient.Get(ctx, types.NamespacedName{Name: issueName, Namespace: namespace}, &updatedIssue)).To(Succeed())
	return meta.FindStatusCondition(updatedIssue.Status.Conditions, ConditionTypeReady)
}

var (
	reconciler *GithubIssueReconciler
	fakeClient client.Client
	testScheme *runtime.Scheme
	ctx        context.Context
	testIssue  *githubv1alpha1.GithubIssue
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

func init() {
	ctx = context.Background()
	testScheme = runtime.NewScheme()
	if err := githubv1alpha1.AddToScheme(testScheme); err != nil {
		panic(err)
	}
}

var _ = Describe("GithubIssue Controller", func() {
	BeforeEach(func() {
		fakeClient = fake.NewClientBuilder().
			WithScheme(testScheme).
			WithStatusSubresource(&githubv1alpha1.GithubIssue{}).
			Build()
		reconciler = &GithubIssueReconciler{
			Client: fakeClient,
			Scheme: testScheme,
		}
		
		testIssue = newTestGithubIssue(testIssueName, testNamespace, testRepoURL, testTitle)
		Expect(fakeClient.Create(ctx, testIssue)).To(Succeed())
	})

	Describe("handleAuthenticationError", func() {
		It("should format error message correctly", func() {
			errorMessage := "token validation failed"
			expectedMessage := "GitHub token not available: " + errorMessage
			testError := errors.New(errorMessage)
			
			result := reconciler.handleAuthenticationError(ctx, testIssue, testError)
			
			Expect(result.RequeueAfter).To(Equal(defaultRequeueDelay))
			
			condition := getStatusCondition(testIssueName, testNamespace)
			Expect(condition.Message).To(Equal(expectedMessage))
		})
	})

	Describe("setErrorConditionAndRequeue", func() {
		It("should set error condition and return requeue result", func() {
			customReason := "CustomError"
			customMessage := "Custom error message"
			requeueDuration := time.Minute * 5

			result := reconciler.setErrorConditionAndRequeue(ctx, testIssue, customReason, customMessage, requeueDuration)

			Expect(result.RequeueAfter).To(Equal(requeueDuration))

			condition := getStatusCondition(testIssueName, testNamespace)
			Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			Expect(condition.Reason).To(Equal(customReason))
			Expect(condition.Message).To(Equal(customMessage))
		})
	})

	Describe("updateSuccessStatus", func() {
		It("should set success condition and handle update errors", func() {
			result, err := reconciler.updateSuccessStatus(ctx, testIssue)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			condition := getStatusCondition(testIssueName, testNamespace)
			Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			Expect(condition.Reason).To(Equal("ClientInitialized"))
		})
	})

	Describe("Reconcile", func() {
		It("should handle nil GithubIssue resource correctly", func() {
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent",
					Namespace: testNamespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())
		})
	})

})
