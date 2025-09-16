package controller

import (
	"context"
	"fmt"

	gogithub "github.com/google/go-github/v57/github"
	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *GithubIssueReconciler) handleCreateOrUpdate(ctx context.Context, cr *githubv1alpha1.GithubIssue) error {
	githubClient, err := r.newGitHubClient(ctx, cr)
	if err != nil {
		utils.SetCondition(ctx, r.Client, cr,
			metav1.ConditionFalse,
			utils.ReasonAuthenticationFailed,
			err.Error())
		return nil
	}

	if err := githubClient.ValidateAuthentication(ctx); err != nil {
		utils.SetCondition(ctx, r.Client, cr,
			metav1.ConditionFalse,
			utils.ReasonAuthenticationFailed,
			err.Error())
		return nil
	}

	existingIssue, err := r.getExistingIssue(ctx, githubClient, cr)
	if err != nil {
		utils.HandleError(ctx, r.Client, cr, err)
		return err
	}

	if existingIssue != nil {
		if err := r.HandleOwnership(ctx, githubClient, existingIssue, cr); err != nil {
			utils.HandleError(ctx, r.Client, cr, err)
			return err
		}

		if err := r.handleUpdateIssue(ctx, githubClient, cr, existingIssue); err != nil {
			return err
		}
		return nil
	}

	if err := r.createNewIssue(ctx, githubClient, cr); err != nil {
		return err
	}

	return nil
}

func (r *GithubIssueReconciler) handleDeletion(ctx context.Context, cr *githubv1alpha1.GithubIssue) error {
	if cr.Status.IssueID != nil {
		githubClient, err := r.newGitHubClient(ctx, cr)
		if err != nil {
			return err
		}

		if err := r.closeGitHubIssue(ctx, githubClient, cr); err != nil {
			return err
		}
	}

	controllerutil.RemoveFinalizer(cr, FinalizerName)
	return r.Update(ctx, cr)
}

func (r *GithubIssueReconciler) updateStatusFromGitHub(cr *githubv1alpha1.GithubIssue, issue *gogithub.Issue) error {
	issueNumber := issue.GetNumber()
	if issueNumber == 0 {
		return fmt.Errorf("GitHub issue has invalid number: %d", issueNumber)
	}

	url := issue.GetHTMLURL()
	if url == "" {
		return fmt.Errorf("GitHub issue has empty URL")
	}

	state := issue.GetState()
	if state == "" {
		return fmt.Errorf("GitHub issue has empty state")
	}

	cr.Status.IssueID = &issueNumber
	cr.Status.URL = &url
	cr.Status.State = &state

	now := metav1.Now()
	cr.Status.LastSyncTime = &now

	hasPR := issue.PullRequestLinks != nil
	cr.Status.HasPullRequest = &hasPR

	return nil
}
