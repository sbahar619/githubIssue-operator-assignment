package controller

import (
	"context"
	"fmt"
	"strings"

	gogithub "github.com/google/go-github/v57/github"
	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Labels for ownership management
	OperatorManagedLabel = "managed-by-github-operator"
	CROwnershipLabel     = "owner"
)

func (r *GithubIssueReconciler) HandleOwnership(ctx context.Context, githubClient *github.Client, issue *gogithub.Issue, cr *githubv1alpha1.GithubIssue) error {
	labels, err := githubClient.ListIssueLabels(ctx, issue.GetNumber())
	if err != nil {
		message := fmt.Sprintf("GitHub API error: %s", err.Error())
		UpdateCondition(ctx, r.Client, cr, metav1.ConditionFalse, ReasonGitHubAPIError, message)
		return err
	}

	if !hasOperatorManagedLabel(labels) {
		UpdateCondition(ctx, r.Client, cr,
			metav1.ConditionFalse,
			ReasonExternallyOwnedIssue,
			"Issue exists but is not managed by operator")
		return fmt.Errorf("issue is externally owned")
	}

	issueOwnerNS, issueOwnerCR := extractOwnerFromLabels(labels)

	if (issueOwnerNS == "" && issueOwnerCR == "") || (issueOwnerNS == cr.Namespace && issueOwnerCR == cr.Name) {
		return nil
	}

	UpdateCondition(ctx, r.Client, cr,
		metav1.ConditionFalse,
		ReasonConflictedOwnership,
		fmt.Sprintf("Issue owned by different CR: expected %s/%s, actual %s/%s", cr.Namespace, cr.Name, issueOwnerNS, issueOwnerCR))
	return fmt.Errorf("issue owned by different CR")
}

func hasOperatorManagedLabel(labels []string) bool {
	for _, label := range labels {
		if strings.Contains(label, OperatorManagedLabel) {
			return true
		}
	}
	return false
}

func GetOwnershipLabels(cr *githubv1alpha1.GithubIssue) (string, string) {
	nsLabel := fmt.Sprintf("ns-%s", cr.Namespace)
	crLabel := fmt.Sprintf("cr-%s", cr.Name)
	return nsLabel, crLabel
}

func extractOwnerFromLabels(labels []string) (namespace, crName string) {
	for _, label := range labels {
		if strings.HasPrefix(label, "ns-") {
			namespace = strings.TrimPrefix(label, "ns-")
		} else if strings.HasPrefix(label, "cr-") {
			crName = strings.TrimPrefix(label, "cr-")
		}
	}
	return namespace, crName
}
