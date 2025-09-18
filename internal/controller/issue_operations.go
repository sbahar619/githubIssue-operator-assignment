package controller

import (
	"context"
	"fmt"

	gogithub "github.com/google/go-github/v57/github"
	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *GithubIssueReconciler) newGitHubClient(cr *githubv1alpha1.GithubIssue) (*github.Client, error) {
	token, err := auth.GetGitHubToken()
	if err != nil {
		return nil, err
	}

	githubClient, err := github.NewClient(token, cr.Spec.Repo)
	if err != nil {
		// CRD validates repo URL format, so this error indicates internal bug
		return nil, err
	}

	return githubClient, nil
}

func (r *GithubIssueReconciler) getExistingIssue(ctx context.Context, githubClient *github.Client, cr *githubv1alpha1.GithubIssue) (*gogithub.Issue, error) {
	if cr.Status.IssueID != nil {
		existingIssue, err := githubClient.GetIssueByID(ctx, *cr.Status.IssueID)
		if err == nil {
			return existingIssue, nil
		}
	}

	return r.getIssueByTitle(ctx, githubClient, cr)
}

func (r *GithubIssueReconciler) getIssueByTitle(ctx context.Context, githubClient *github.Client, cr *githubv1alpha1.GithubIssue) (*gogithub.Issue, error) {
	log := logf.FromContext(ctx)

	existingIssue, err := githubClient.GetIssueByTitle(ctx, cr.Spec.Title)
	if err != nil {
		return nil, err
	}

	if existingIssue != nil {
		if err := r.updateStatusFromGitHub(cr, existingIssue); err != nil {
			return nil, err
		}

		log.Info("Found existing GitHub issue", "issueID", *cr.Status.IssueID, "name", cr.Name, "namespace", cr.Namespace)
	}

	return existingIssue, nil
}

func (r *GithubIssueReconciler) createNewIssue(ctx context.Context, githubClient *github.Client, cr *githubv1alpha1.GithubIssue) error {
	log := logf.FromContext(ctx)

	log.Info("Creating GitHub issue", "title", cr.Spec.Title, "name", cr.Name, "namespace", cr.Namespace)

	description := ""
	if cr.Spec.Description != nil {
		description = *cr.Spec.Description
	}

	createdIssue, err := githubClient.CreateIssue(ctx, cr.Spec.Title, description)
	if err != nil {
		UpdateCondition(ctx, r.Client, cr,
			metav1.ConditionFalse,
			ReasonGitHubAPIError,
			fmt.Sprintf("Failed to create GitHub issue: %s", err.Error()))
		return err
	}

	labels := []string{
		OperatorManagedLabel,
		fmt.Sprintf("ns-%s", cr.Namespace),
		fmt.Sprintf("cr-%s", cr.Name),
	}

	if err := githubClient.AddLabelsToIssue(ctx, createdIssue.GetNumber(), labels); err != nil {
		message := fmt.Sprintf("GitHub API error: %s", err.Error())
		UpdateCondition(ctx, r.Client, cr, metav1.ConditionFalse, ReasonGitHubAPIError, message)
		return err
	}

	if err := r.updateStatusFromGitHub(cr, createdIssue); err != nil {
		UpdateCondition(ctx, r.Client, cr,
			metav1.ConditionFalse,
			ReasonGitHubAPIError,
			fmt.Sprintf("Failed to update status after issue creation: %s", err.Error()))
		return err
	}

	UpdateCondition(ctx, r.Client, cr,
		metav1.ConditionTrue,
		ReasonIssueCreated,
		fmt.Sprintf("GitHub issue #%d created successfully", *cr.Status.IssueID))

	log.Info("GitHub issue created", "issueID", *cr.Status.IssueID, "name", cr.Name, "namespace", cr.Namespace)
	return nil
}

func (r *GithubIssueReconciler) handleUpdateIssue(ctx context.Context, githubClient *github.Client, cr *githubv1alpha1.GithubIssue, existingIssue *gogithub.Issue) error {
	log := logf.FromContext(ctx)

	if existingIssue.GetState() == github.IssueStateClosed {
		log.Info("Reopening closed GitHub issue", "issueID", existingIssue.GetNumber(), "name", cr.Name, "namespace", cr.Namespace)
		if _, err := githubClient.OpenIssue(ctx, existingIssue.GetNumber()); err != nil {
			message := fmt.Sprintf("GitHub API error: %s", err.Error())
			UpdateCondition(ctx, r.Client, cr, metav1.ConditionFalse, ReasonGitHubAPIError, message)
			return err
		}
	}

	labels := []string{
		OperatorManagedLabel,
		fmt.Sprintf("ns-%s", cr.Namespace),
		fmt.Sprintf("cr-%s", cr.Name),
	}

	if err := githubClient.AddLabelsToIssue(ctx, existingIssue.GetNumber(), labels); err != nil {
		message := fmt.Sprintf("GitHub API error: %s", err.Error())
		UpdateCondition(ctx, r.Client, cr, metav1.ConditionFalse, ReasonGitHubAPIError, message)
		return err
	}

	if updateNeeded := r.isUpdateNeeded(cr, existingIssue); updateNeeded {
		log.Info("Updating GitHub issue content", "issueID", *cr.Status.IssueID, "name", cr.Name, "namespace", cr.Namespace)

		description := ""
		if cr.Spec.Description != nil {
			description = *cr.Spec.Description
		}
		updatedIssue, err := githubClient.UpdateIssue(ctx, *existingIssue.Number, cr.Spec.Title, description)
		if err != nil {
			UpdateCondition(ctx, r.Client, cr,
				metav1.ConditionFalse,
				ReasonGitHubAPIError,
				fmt.Sprintf("Failed to update issue #%d: %s", *cr.Status.IssueID, err.Error()))
			return err
		}

		if err := r.updateStatusFromGitHub(cr, updatedIssue); err != nil {
			UpdateCondition(ctx, r.Client, cr,
				metav1.ConditionFalse,
				ReasonGitHubAPIError,
				fmt.Sprintf("Failed to update status after issue update: %s", err.Error()))
			return err
		}

		UpdateCondition(ctx, r.Client, cr,
			metav1.ConditionTrue,
			ReasonIssueSynchronized,
			fmt.Sprintf("Issue #%d updated and synchronized successfully", *cr.Status.IssueID))

		log.Info("GitHub issue updated", "issueID", *cr.Status.IssueID, "name", cr.Name, "namespace", cr.Namespace)
		return nil
	}

	UpdateCondition(ctx, r.Client, cr,
		metav1.ConditionTrue,
		ReasonIssueSynchronized,
		fmt.Sprintf("Issue #%d content matches desired state", *cr.Status.IssueID))

	return nil
}

func (r *GithubIssueReconciler) isUpdateNeeded(cr *githubv1alpha1.GithubIssue, issue *gogithub.Issue) bool {
	if cr.Spec.Title != issue.GetTitle() {
		return true
	}

	specDesc := ""
	if cr.Spec.Description != nil {
		specDesc = *cr.Spec.Description
	}
	githubDesc := ""
	if issue.Body != nil {
		githubDesc = *issue.Body
	}
	return specDesc != githubDesc
}

func (r *GithubIssueReconciler) deleteOwnershipLabels(ctx context.Context, githubClient *github.Client, cr *githubv1alpha1.GithubIssue) {
	log := logf.FromContext(ctx)

	nsLabel := fmt.Sprintf("ns-%s", cr.Namespace)
	crLabel := fmt.Sprintf("cr-%s", cr.Name)

	if err := githubClient.RemoveLabelFromIssue(ctx, *cr.Status.IssueID, nsLabel); err != nil {
		log.Error(err, "Failed to remove namespace label from GitHub issue", "issueID", *cr.Status.IssueID, "label", nsLabel, "name", cr.Name, "namespace", cr.Namespace)
	}

	if err := githubClient.RemoveLabelFromIssue(ctx, *cr.Status.IssueID, crLabel); err != nil {
		log.Error(err, "Failed to remove CR label from GitHub issue", "issueID", *cr.Status.IssueID, "label", crLabel, "name", cr.Name, "namespace", cr.Namespace)
	}
}

func (r *GithubIssueReconciler) closeGitHubIssue(ctx context.Context, githubClient *github.Client, cr *githubv1alpha1.GithubIssue) error {
	log := logf.FromContext(ctx)

	if _, err := githubClient.CloseIssue(ctx, *cr.Status.IssueID); err != nil {
		if githubError, ok := err.(*github.GitHubError); ok && githubError.StatusCode == github.HTTPStatusGone {
			return nil
		}
		return err
	}

	log.Info("GitHub issue closed", "issueID", *cr.Status.IssueID, "name", cr.Name, "namespace", cr.Namespace)
	return nil
}
