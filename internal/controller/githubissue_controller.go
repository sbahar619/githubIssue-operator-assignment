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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/auth"
	"github.com/sbahar619/githubIssue-operator-assignment/internal/github"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionTypeReady         = "Ready"
	ReasonAuthenticationFailed = "AuthenticationFailed"
	ReasonClientError          = "ClientError"
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

	githubIssue, err := r.fetchGithubIssue(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}

	if githubIssue == nil {
		log.Info("GithubIssue resource not found, skipping reconciliation", "githubissue", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	log.Info("Starting reconciliation", "githubissue", req.NamespacedName)

	githubClient, result := r.initializeGitHubClient(ctx, githubIssue)
	if !result.IsZero() {
		return result, nil
	}

	_ = githubClient

	return r.updateSuccessStatus(ctx, githubIssue)
}

func (r *GithubIssueReconciler) fetchGithubIssue(ctx context.Context, namespacedName types.NamespacedName) (*githubv1alpha1.GithubIssue, error) {
	var githubIssue githubv1alpha1.GithubIssue
	if err := r.Get(ctx, namespacedName, &githubIssue); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return &githubIssue, nil
}

func (r *GithubIssueReconciler) initializeGitHubClient(ctx context.Context, githubIssue *githubv1alpha1.GithubIssue) (*github.Client, ctrl.Result) {
	log := logf.FromContext(ctx)

	token, err := auth.GetGitHubToken()
	if err != nil {
		return nil, r.handleAuthenticationError(ctx, githubIssue, err)
	}

	log.Info("Successfully retrieved GitHub token")

	githubClient, err := github.NewClient(token, githubIssue.Spec.Repo)
	if err != nil {
		return nil, r.handleClientError(ctx, githubIssue, err)
	}

	log.Info("Successfully created GitHub client", "repo", githubIssue.Spec.Repo)
	return githubClient, ctrl.Result{}
}

func (r *GithubIssueReconciler) handleAuthenticationError(ctx context.Context, githubIssue *githubv1alpha1.GithubIssue, err error) ctrl.Result {
	message := "GitHub token not available"
	if err != nil {
		message += ": " + err.Error()
	}
	return r.setErrorConditionAndRequeue(ctx, githubIssue,
		ReasonAuthenticationFailed,
		message,
		time.Minute*1)
}

func (r *GithubIssueReconciler) handleClientError(ctx context.Context, githubIssue *githubv1alpha1.GithubIssue, err error) ctrl.Result {
	return r.setErrorConditionAndRequeue(ctx, githubIssue,
		ReasonClientError,
		"Failed to create GitHub client: "+err.Error(),
		time.Minute*1)
}

func (r *GithubIssueReconciler) setErrorConditionAndRequeue(ctx context.Context, githubIssue *githubv1alpha1.GithubIssue, reason, message string, requeueAfter time.Duration) ctrl.Result {
	log := logf.FromContext(ctx)

	meta.SetStatusCondition(&githubIssue.Status.Conditions, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})

	if updateErr := r.Status().Update(ctx, githubIssue); updateErr != nil {
		log.Error(updateErr, "Failed to update status", "githubissue", githubIssue.Name)
	}

	return ctrl.Result{RequeueAfter: requeueAfter}
}

func (r *GithubIssueReconciler) updateSuccessStatus(ctx context.Context, githubIssue *githubv1alpha1.GithubIssue) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	meta.SetStatusCondition(&githubIssue.Status.Conditions, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  metav1.ConditionTrue,
		Reason:  "ClientInitialized",
		Message: "GitHub client successfully initialized",
	})

	if err := r.Status().Update(ctx, githubIssue); err != nil {
		log.Error(err, "Failed to update status", "githubissue", githubIssue.Name)
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubIssueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&githubv1alpha1.GithubIssue{}).
		Named("githubissue").
		Complete(r)
}
