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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	mailv1alpha1 "github.com/circa10a/postk8s/api/v1alpha1"
)

// MailReconciler reconciles a Mail object
type MailReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=mail.circa10a.github.io,resources=mails,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mail.circa10a.github.io,resources=mails/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mail.circa10a.github.io,resources=mails/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MailReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1) Load the Mail object
	mail := &mailv1alpha1.Mail{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, mail); err != nil {
		if apierrors.IsNotFound(err) {
			// object was deleted after reconcile request — nothing to do
			return ctrl.Result{}, nil
		}
		// transient error reading the object
		return ctrl.Result{}, err
	}

	// 2) Inspect spec fields
	log.Info("Got Mail", "service", mail.Spec.Service, "customerRef", mail.Spec.CustomerReference)

	// Example: set status and update it
	mail.Status.State = "Processed"
	mail.Status.Sent = false

	// Update status; handle conflicts by requeuing (or use retry.RetryOnConflict for more control)
	if err := r.Status().Update(ctx, mail); err != nil {
		// if conflict or temporary error, requeue to try again
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MailReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mailv1alpha1.Mail{}).
		Named("mail").
		Complete(r)
}
