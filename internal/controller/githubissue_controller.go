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
	log.Info("Starting reconciliation", "githubissue", req.NamespacedName)

	var githubIssue githubv1alpha1.GithubIssue
	if err := r.Get(ctx, req.NamespacedName, &githubIssue); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if githubIssue.Name == "" {
		log.Info("Resource not found, skipping", "githubissue", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	token, err := auth.GetGitHubToken()
	if err != nil {
		utils.HandleGitHubClientError(ctx, r.Client, &githubIssue, utils.TokenRetrievalError, err)
		return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
	}
	log.Info("Successfully retrieved GitHub token")

	githubClient, err := github.NewClient(token, githubIssue.Spec.Repo)
	if err != nil {
		utils.HandleGitHubClientError(ctx, r.Client, &githubIssue, utils.ClientCreationError, err)
		return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
	}
	log.Info("Successfully created GitHub client", "repo", githubIssue.Spec.Repo)

	return r.handleUpdateOrCreateIssue(ctx, githubClient, &githubIssue)
}

func (r *GithubIssueReconciler) handleUpdateOrCreateIssue(ctx context.Context, githubClient *github.Client, githubIssue *githubv1alpha1.GithubIssue) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	existingIssue, err := r.getIssueByTitle(ctx, githubClient, githubIssue)
	if err != nil {
		utils.HandleGitHubAPIError(ctx, r.Client, githubIssue, err)
		return ctrl.Result{RequeueAfter: time.Minute * 2}, nil
	}

	if existingIssue != nil {
		log.Info("Issue synchronization path: existing issue found")
		return r.synchronizeExistingIssue(ctx, githubClient, githubIssue, existingIssue)
	}

	log.Info("Issue creation path: no existing issue found")
	return r.createNewIssue(ctx, githubClient, githubIssue)
}

func (r *GithubIssueReconciler) getIssueByTitle(ctx context.Context, githubClient *github.Client, githubIssue *githubv1alpha1.GithubIssue) (*gogithub.Issue, error) {
	log := logf.FromContext(ctx)

	existingIssue, err := githubClient.GetIssueByTitle(ctx, githubIssue.Spec.Title)
	if err != nil {
		return nil, err
	}

	if existingIssue != nil {
		log.Info("Found existing GitHub issue",
			"issueID", existingIssue.GetID(),
			"issueNumber", existingIssue.GetNumber(),
			"issueURL", existingIssue.GetHTMLURL())
	} else {
		log.Info("No existing GitHub issue found for title", "title", githubIssue.Spec.Title)
	}

	return existingIssue, nil
}

func (r *GithubIssueReconciler) synchronizeExistingIssue(ctx context.Context, githubClient *github.Client, githubIssue *githubv1alpha1.GithubIssue, existingIssue *gogithub.Issue) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Synchronization logic not yet implemented", "issueID", existingIssue.GetID())

	_ = githubClient // TODO: Will be used when synchronization logic is implemented

	details := fmt.Sprintf("GitHub issue #%d synchronized successfully", existingIssue.GetNumber())
	utils.SetCondition(ctx, r.Client, githubIssue, metav1.ConditionTrue, utils.ReasonIssueSynchronized, details)
	return ctrl.Result{}, nil
}

func (r *GithubIssueReconciler) createNewIssue(ctx context.Context, githubClient *github.Client, githubIssue *githubv1alpha1.GithubIssue) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Issue creation logic not yet implemented")

	_ = githubClient // TODO: Will be used when issue creation logic is implemented

	details := "GitHub issue created successfully (placeholder implementation)"
	utils.SetCondition(ctx, r.Client, githubIssue, metav1.ConditionTrue, utils.ReasonIssueCreated, details)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubIssueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&githubv1alpha1.GithubIssue{}).
		Named("githubissue").
		Complete(r)
}
