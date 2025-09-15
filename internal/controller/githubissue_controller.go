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
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	githubv1alpha1 "github.com/sbahar619/githubIssue-operator-assignment/api/v1alpha1"
)

const (
	FinalizerName = "github.shahaf.com/finalizer"
)

// GenerationChangedOrDeletionPredicate returns a predicate that allows events when:
// 1. Generation changes (spec updates) - prevents unnecessary reconciliation on status updates
// 2. DeletionTimestamp is set (deletion events) - ensures finalizers are processed
// This maintains the performance benefits of GenerationChangedPredicate while fixing deletion handling.
func GenerationChangedOrDeletionPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Allow if generation changed (spec update)
			if e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() {
				return true
			}
			// Allow if deletion timestamp was set (deletion event)
			oldDeletion := e.ObjectOld.GetDeletionTimestamp()
			newDeletion := e.ObjectNew.GetDeletionTimestamp()
			if oldDeletion == nil && newDeletion != nil {
				// Log deletion event for debugging
				logf.Log.Info("Deletion event detected - allowing reconciliation",
					"name", e.ObjectNew.GetName(),
					"namespace", e.ObjectNew.GetNamespace())
				return true
			}
			// Block status-only updates
			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			// Always allow create events
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Always allow delete events (though these are rare with finalizers)
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			// Allow generic events (like periodic resync)
			return true
		},
	}
}

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

	var githubIssue githubv1alpha1.GithubIssue
	if err := r.Get(ctx, req.NamespacedName, &githubIssue); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !controllerutil.ContainsFinalizer(&githubIssue, FinalizerName) {
		controllerutil.AddFinalizer(&githubIssue, FinalizerName)
		if err := r.Update(ctx, &githubIssue); err != nil {
			log.Error(err, "Failed to add finalizer", "name", githubIssue.Name, "namespace", githubIssue.Namespace)
			return ctrl.Result{}, err
		}
	}

	if githubIssue.DeletionTimestamp != nil {
		log.Info("Deleting GitHub issue", "name", githubIssue.Name, "namespace", githubIssue.Namespace)
		if err := r.handleDeletion(ctx, &githubIssue); err != nil {
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
		controllerutil.RemoveFinalizer(&githubIssue, FinalizerName)
		if err := r.Update(ctx, &githubIssue); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := r.handleCreateOrUpdate(ctx, &githubIssue); err != nil {
		log.Error(err, "Failed to reconcile GitHub issue", "name", githubIssue.Name, "namespace", githubIssue.Namespace)
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubIssueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&githubv1alpha1.GithubIssue{}).
		WithEventFilter(GenerationChangedOrDeletionPredicate()).
		Named("githubissue").
		Complete(r)
}
