/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	gogithub "github.com/google/go-github/v57/github"
	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/github"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/utils"
)

type GithubIssueReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=github.shahaf.com,resources=githubissues,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=github.shahaf.com,resources=githubissues/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=github.shahaf.com,resources=githubissues/finalizers,verbs=update

func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciliation started", "resource", req.NamespacedName)

	var githubIssue githubv1alpha1.GithubIssue
	if err := r.Get(ctx, req.NamespacedName, &githubIssue); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if githubIssue.Name == "" {
		log.Info("Resource not found", "resource", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	token, err := auth.GetGitHubToken()
	if err != nil {
		utils.HandleTokenRetrievalError(ctx, r.Client, &githubIssue, err)
		return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
	}
	log.Info("GitHub token retrieved")

	githubClient, err := github.NewClient(token, githubIssue.Spec.Repo)
	if err != nil {
		utils.HandleGitHubAPIError(ctx, r.Client, &githubIssue, err)
		return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
	}
	log.Info("GitHub client created", "repo", githubIssue.Spec.Repo)

	existingIssue, err := r.getIssueByTitle(ctx, githubClient, &githubIssue)
	if err != nil {
		utils.HandleGitHubAPIError(ctx, r.Client, &githubIssue, err)
		return ctrl.Result{RequeueAfter: time.Minute * 2}, nil
	}

	if existingIssue != nil {
		if err = r.handleUpdateIssue(ctx, githubClient, &githubIssue, existingIssue); err != nil {
			log.Info("Update required, requeueing", "error", err.Error())
			return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
		}
	} else {
		log.Info("Creating new issue", "title", githubIssue.Spec.Title)
		err = r.createNewIssue(ctx, githubClient, &githubIssue)
		if err != nil {
			log.Error(err, "Failed to create new issue")
			return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *GithubIssueReconciler) handleUpdateIssue(ctx context.Context, githubClient *github.Client, githubIssue *githubv1alpha1.GithubIssue, existingIssue *gogithub.Issue) error {
	log := logf.FromContext(ctx)
	log.Info("Synchronizing existing issue", "issueID", *githubIssue.Status.IssueID, "title", githubIssue.Spec.Title)

	if updateNeeded := r.isUpdateNeeded(githubIssue, existingIssue); updateNeeded {
		log.Info("Content differs, update required", "issueID", *githubIssue.Status.IssueID)

		utils.SetCondition(ctx, r.Client, githubIssue,
			metav1.ConditionFalse,
			utils.ReasonUpdateRequired,
			fmt.Sprintf("Issue #%d content differs from desired state, update required", *githubIssue.Status.IssueID))

		description := ""
		if githubIssue.Spec.Description != nil {
			description = *githubIssue.Spec.Description
		}
		updatedIssue, err := githubClient.UpdateIssue(ctx, *existingIssue.Number, githubIssue.Spec.Title, description)
		if err != nil {
			utils.SetCondition(ctx, r.Client, githubIssue,
				metav1.ConditionFalse,
				utils.ReasonGitHubAPIError,
				fmt.Sprintf("Failed to update issue #%d: %s", *githubIssue.Status.IssueID, err.Error()))
			return err
		}

		if err := r.updateStatusFromGitHub(githubIssue, updatedIssue); err != nil {
			utils.SetCondition(ctx, r.Client, githubIssue,
				metav1.ConditionFalse,
				utils.ReasonGitHubAPIError,
				fmt.Sprintf("Failed to update status after issue update: %s", err.Error()))
			return err
		}

		utils.SetCondition(ctx, r.Client, githubIssue,
			metav1.ConditionTrue,
			utils.ReasonIssueSynchronized,
			fmt.Sprintf("Issue #%d updated and synchronized successfully", *githubIssue.Status.IssueID))

		log.Info("Issue updated successfully", "issueID", *githubIssue.Status.IssueID)
		return nil
	}

	// No update needed, mark as synchronized
	log.Info("Issue synchronized", "issueID", *githubIssue.Status.IssueID, "status", "up-to-date")

	utils.SetCondition(ctx, r.Client, githubIssue,
		metav1.ConditionTrue,
		utils.ReasonIssueSynchronized,
		fmt.Sprintf("Issue #%d content matches desired state", *githubIssue.Status.IssueID))

	return nil
}

func (r *GithubIssueReconciler) getIssueByTitle(ctx context.Context, githubClient *github.Client, githubIssue *githubv1alpha1.GithubIssue) (*gogithub.Issue, error) {
	log := logf.FromContext(ctx)

	existingIssue, err := githubClient.GetIssueByTitle(ctx, githubIssue.Spec.Title)
	if err != nil {
		return nil, err
	}

	if existingIssue != nil {

		if err := r.updateStatusFromGitHub(githubIssue, existingIssue); err != nil {
			log.Error(err, "Failed to update status from GitHub issue")
			return nil, fmt.Errorf("status update failed: %w", err)
		}

		utils.SetCondition(ctx, r.Client, githubIssue,
			metav1.ConditionFalse,
			utils.ReasonIssueFound,
			fmt.Sprintf("Found existing issue #%d, checking if update is needed", *githubIssue.Status.IssueID))

		log.Info("Existing issue found",
			"issueID", *githubIssue.Status.IssueID,
			"url", *githubIssue.Status.URL,
			"state", *githubIssue.Status.State)
	} else {
		log.Info("No existing issue found", "title", githubIssue.Spec.Title)
	}

	return existingIssue, nil
}

//nolint:unparam // Function always returns nil during stub phase, will be implemented later
func (r *GithubIssueReconciler) createNewIssue(ctx context.Context, githubClient *github.Client, githubIssue *githubv1alpha1.GithubIssue) error {
	log := logf.FromContext(ctx)
	log.Info("Issue creation not implemented", "phase", "stub")

	_ = githubClient // TODO: Will be used when issue creation logic is implemented

	details := "GitHub issue created successfully (placeholder implementation)"
	utils.SetCondition(ctx, r.Client, githubIssue, metav1.ConditionTrue, utils.ReasonIssueCreated, details)
	return nil
}

func (r *GithubIssueReconciler) updateStatusFromGitHub(githubIssue *githubv1alpha1.GithubIssue, issue *gogithub.Issue) error {
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

	githubIssue.Status.IssueID = &issueNumber
	githubIssue.Status.URL = &url
	githubIssue.Status.State = &state

	now := metav1.Now()
	githubIssue.Status.LastSyncTime = &now

	hasPR := issue.PullRequestLinks != nil
	githubIssue.Status.HasPullRequest = &hasPR

	return nil
}

func (r *GithubIssueReconciler) isUpdateNeeded(githubIssue *githubv1alpha1.GithubIssue, issue *gogithub.Issue) bool {
	if githubIssue.Spec.Title != issue.GetTitle() {
		return true
	}

	specDesc := ""
	if githubIssue.Spec.Description != nil {
		specDesc = *githubIssue.Spec.Description
	}
	githubDesc := ""
	if issue.Body != nil {
		githubDesc = *issue.Body
	}
	return specDesc != githubDesc
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubIssueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&githubv1alpha1.GithubIssue{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("githubissue").
		Complete(r)
}
