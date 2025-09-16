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
	ConditionTypeReady = "Ready"

	ReasonIssueCreated      = "IssueCreated"
	ReasonIssueSynchronized = "IssueSynchronized"

	ReasonIssueFound     = "IssueFound"
	ReasonUpdateRequired = "UpdateRequired"

	ReasonAuthenticationFailed = "AuthenticationFailed"
	ReasonGitHubAPIError       = "GitHubAPIError"
	ReasonExternallyOwnedIssue = "ExternallyOwnedIssue"
	ReasonConflictedOwnership  = "ConflictedOwnership"
)

func HandleError(ctx context.Context, k8sClient client.Client, githubIssue *githubv1alpha1.GithubIssue, err error) {
	if githubError, ok := err.(*github.GitHubError); ok {
		message := fmt.Sprintf("GitHub API error: %s", githubError.Message)
		SetCondition(ctx, k8sClient, githubIssue, metav1.ConditionFalse, ReasonGitHubAPIError, message)
		return
	}

	clearGithubStatus(githubIssue)
	message := fmt.Sprintf("GitHub authentication error: %s", err.Error())
	SetCondition(ctx, k8sClient, githubIssue, metav1.ConditionFalse, ReasonAuthenticationFailed, message)
}

func clearGithubStatus(githubIssue *githubv1alpha1.GithubIssue) {
	githubIssue.Status.IssueID = nil
	githubIssue.Status.URL = nil
	githubIssue.Status.State = nil
	githubIssue.Status.LastSyncTime = nil
	githubIssue.Status.HasPullRequest = nil
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
		log.Error(updateErr, "Failed to update status", "name", githubIssue.Name, "namespace", githubIssue.Namespace)
	}
}
