package utils

import (
	"context"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
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

func ClearGithubStatus(githubIssue *githubv1alpha1.GithubIssue) {
	githubIssue.Status.IssueID = nil
	githubIssue.Status.URL = nil
	githubIssue.Status.State = nil
	githubIssue.Status.LastSyncTime = nil
	githubIssue.Status.HasPullRequest = nil
}

func UpdateCondition(ctx context.Context, k8sClient client.Client, githubIssue *githubv1alpha1.GithubIssue, status metav1.ConditionStatus, reason, message string) {
	log := logf.FromContext(ctx)

	meta.SetStatusCondition(&githubIssue.Status.Conditions, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  status,
		Reason:  reason,
		Message: message,
	})

	if err := k8sClient.Status().Update(ctx, githubIssue); err != nil {
		log.Error(err, "Failed to update status", "name", githubIssue.Name, "namespace", githubIssue.Namespace)
	}
}
