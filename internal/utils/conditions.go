package utils

import (
	"context"
	"fmt"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/github"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// Condition Types
	ConditionTypeReady = "Ready"

	// Error Reasons
	ReasonAuthenticationFailed = "AuthenticationFailed"
	ReasonGitHubAPIError       = "GitHubAPIError"
	ReasonUnexpectedError      = "UnexpectedError"

	// Success Reasons
	ReasonIssueCreated      = "IssueCreated"
	ReasonIssueSynchronized = "IssueSynchronized"
)

func HandleTokenRetrievalError(ctx context.Context, k8sClient client.Client, githubIssue *githubv1alpha1.GithubIssue, err error) {
	message := fmt.Sprintf("GitHub token not available: %s", err.Error())
	SetCondition(ctx, k8sClient, githubIssue, metav1.ConditionFalse, ReasonAuthenticationFailed, message)
}

func HandleGitHubAPIError(ctx context.Context, k8sClient client.Client, githubIssue *githubv1alpha1.GithubIssue, err error) bool {
	handleUnexpectedError := func(ctx context.Context, k8sClient client.Client, githubIssue *githubv1alpha1.GithubIssue, err error) {
		message := fmt.Sprintf("Unexpected error during GitHub lookup: %s", err.Error())
		SetCondition(ctx, k8sClient, githubIssue, metav1.ConditionFalse, ReasonUnexpectedError, message)
	}

	if githubError, ok := err.(*github.GitHubError); ok {
		if githubError.IsRetryable {
			handleRetryableGitHubError(ctx, k8sClient, githubIssue, githubError)
		} else {
			handleNonRetryableGitHubError(ctx, k8sClient, githubIssue, githubError)
		}
		return true
	}

	handleUnexpectedError(ctx, k8sClient, githubIssue, err)
	return false
}

func handleRetryableGitHubError(ctx context.Context, k8sClient client.Client, githubIssue *githubv1alpha1.GithubIssue, githubError *github.GitHubError) {
	message := fmt.Sprintf("GitHub API temporarily unavailable: %s", githubError.Message)
	SetCondition(ctx, k8sClient, githubIssue, metav1.ConditionFalse, ReasonGitHubAPIError, message)
}

func handleNonRetryableGitHubError(ctx context.Context, k8sClient client.Client, githubIssue *githubv1alpha1.GithubIssue, githubError *github.GitHubError) {
	message := fmt.Sprintf("GitHub API error: %s", githubError.Message)
	SetCondition(ctx, k8sClient, githubIssue, metav1.ConditionFalse, ReasonGitHubAPIError, message)
}

func SetCondition(ctx context.Context, k8sClient client.Client, githubIssue *githubv1alpha1.GithubIssue, status metav1.ConditionStatus, reason, message string) {
	log := logf.FromContext(ctx)

	meta.SetStatusCondition(&githubIssue.Status.Conditions, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  status,
		Reason:  reason,
		Message: message,
	})

	if updateErr := k8sClient.Status().Update(ctx, githubIssue); updateErr != nil {
		log.Error(updateErr, "Failed to update status", "githubissue", githubIssue.Name)
	}
}
