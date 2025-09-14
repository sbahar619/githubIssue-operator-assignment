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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
)

const (
	FinalizerName = "github.shahaf.com/finalizer"
)

// GithubIssueReconciler reconciles a GithubIssue object
type GithubIssueReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=github.shahaf.com,resources=githubissues,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=github.shahaf.com,resources=githubissues/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=github.shahaf.com,resources=githubissues/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciliation started", "resource", req.NamespacedName)

	var githubIssue githubv1alpha1.GithubIssue
	if err := r.Get(ctx, req.NamespacedName, &githubIssue); err != nil {
		log.Info("Failed to get CR or CR not found", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Successfully retrieved CR", "name", githubIssue.Name, "namespace", githubIssue.Namespace)

	if !controllerutil.ContainsFinalizer(&githubIssue, FinalizerName) {
		log.Info("Adding finalizer to CR")
		controllerutil.AddFinalizer(&githubIssue, FinalizerName)
		if err := r.Update(ctx, &githubIssue); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		log.Info("Finalizer added successfully, continuing with reconciliation")
	} else {
		log.Info("Finalizer already present, proceeding with reconciliation")
	}

	if githubIssue.DeletionTimestamp != nil {
		log.Info("CR is being deleted, handling deletion")
		if err := r.handleDeletion(ctx, &githubIssue); err != nil {
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
		controllerutil.RemoveFinalizer(&githubIssue, FinalizerName)
		if err := r.Update(ctx, &githubIssue); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	log.Info("CR is not being deleted, calling handleCreateOrUpdate")

	if err := r.handleCreateOrUpdate(ctx, &githubIssue); err != nil {
		log.Error(err, "handleCreateOrUpdate failed, will requeue")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	log.Info("handleCreateOrUpdate completed successfully")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubIssueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&githubv1alpha1.GithubIssue{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("githubissue").
		Complete(r)
}
