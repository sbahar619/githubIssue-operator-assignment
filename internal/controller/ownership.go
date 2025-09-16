package controller

import (
	"context"
	"fmt"
	"strings"

	gogithub "github.com/google/go-github/v57/github"
	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/github"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/utils"
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
		utils.HandleError(ctx, r.Client, cr, err)
		return err
	}

	if !hasOperatorManagedLabel(labels) {
		utils.SetCondition(ctx, r.Client, cr,
			metav1.ConditionFalse,
			utils.ReasonExternallyOwnedIssue,
			"Issue exists but is not managed by operator")
		return nil
	}

	expectedNS := cr.Namespace
	expectedCR := cr.Name
	actualNS, actualCR := extractOwnerFromLabels(labels)

	if (actualNS == "" && actualCR == "") || (actualNS == expectedNS && actualCR == expectedCR) {
		return nil
	}

	utils.SetCondition(ctx, r.Client, cr,
		metav1.ConditionFalse,
		utils.ReasonConflictedOwnership,
		fmt.Sprintf("Issue owned by different CR: expected %s/%s, actual %s/%s", expectedNS, expectedCR, actualNS, actualCR))
	return nil
}

func hasOperatorManagedLabel(labels []string) bool {
	for _, label := range labels {
		if strings.Contains(label, OperatorManagedLabel) {
			return true
		}
	}
	return false
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
