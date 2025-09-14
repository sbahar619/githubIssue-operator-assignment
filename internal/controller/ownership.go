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
	OperatorManagedLabel = "managed-by-github-operator" // Static label (never removed)
	CROwnershipLabel     = "owner"                      // Dynamic: will be formatted as "owner-namespace-name" (removed on deletion)
)

// HandleOwnership validates and manages issue ownership
func (r *GithubIssueReconciler) HandleOwnership(ctx context.Context, githubClient *github.Client, issue *gogithub.Issue, cr *githubv1alpha1.GithubIssue) error {
	labels, err := githubClient.ListIssueLabels(ctx, issue.GetNumber())
	if err != nil {
		return fmt.Errorf("failed to retrieve issue labels: %w", err)
	}

	// Check if issue is managed by operator
	if !hasOperatorManagedLabel(labels) {
		utils.SetCondition(ctx, r.Client, cr,
			metav1.ConditionFalse,
			utils.ReasonExternallyOwnedIssue,
			"Issue exists but is not managed by operator")
		return nil // Handled internally, don't requeue
	}

	// Check specific CR ownership
	expectedOwner := fmt.Sprintf("%s/%s", cr.Namespace, cr.Name)
	actualOwner := extractCROwnerFromLabels(labels)

	// If orphaned (no owner) or owned by this CR, we can proceed
	if actualOwner == "" || actualOwner == expectedOwner {
		return nil // Ownership verified, can proceed
	}

	// Issue is owned by different CR
	utils.SetCondition(ctx, r.Client, cr,
		metav1.ConditionFalse,
		utils.ReasonConflictedOwnership,
		fmt.Sprintf("Issue owned by different CR: expected %s, actual %s", expectedOwner, actualOwner))
	return nil // Handled internally, don't requeue
}

// hasOperatorManagedLabel checks if the issue has the operator managed label
func hasOperatorManagedLabel(labels []string) bool {
	for _, label := range labels {
		if strings.Contains(label, OperatorManagedLabel) {
			return true
		}
	}
	return false
}

// extractCROwnerFromLabels extracts the CR owner from issue labels
func extractCROwnerFromLabels(labels []string) string {
	prefix := "owner-"
	for _, label := range labels {
		if strings.HasPrefix(label, prefix) {
			// Extract namespace/name from "owner-namespace-name" format
			ownerPart := strings.TrimPrefix(label, prefix)
			// Convert "namespace-name" back to "namespace/name"
			parts := strings.Split(ownerPart, "-")
			if len(parts) >= 2 {
				// Rejoin with "/" - assumes namespace and name don't contain hyphens
				// For more robust parsing, we'd need a different separator
				return strings.Join(parts[:len(parts)-1], "-") + "/" + parts[len(parts)-1]
			}
			return ownerPart
		}
	}
	return ""
}
