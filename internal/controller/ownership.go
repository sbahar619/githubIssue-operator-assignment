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
		return fmt.Errorf("failed to retrieve issue labels: %w", err)
	}

	if !hasOperatorManagedLabel(labels) {
		utils.SetCondition(ctx, r.Client, cr,
			metav1.ConditionFalse,
			utils.ReasonExternallyOwnedIssue,
			"Issue exists but is not managed by operator")
		return nil
	}

	expectedOwner := fmt.Sprintf("%s/%s", cr.Namespace, cr.Name)
	actualOwner := extractCROwnerFromLabels(labels)

	if actualOwner == "" || actualOwner == expectedOwner {
		return nil
	}

	utils.SetCondition(ctx, r.Client, cr,
		metav1.ConditionFalse,
		utils.ReasonConflictedOwnership,
		fmt.Sprintf("Issue owned by different CR: expected %s, actual %s", expectedOwner, actualOwner))
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

func extractCROwnerFromLabels(labels []string) string {
	prefix := "owner-"
	for _, label := range labels {
		if strings.HasPrefix(label, prefix) {
			ownerPart := strings.TrimPrefix(label, prefix)

			dotIndex := strings.Index(ownerPart, ".")
			if dotIndex == -1 {
				return ownerPart
			}

			namespace := ownerPart[:dotIndex]
			name := ownerPart[dotIndex+1:]
			return namespace + "/" + name
		}
	}
	return ""
}
