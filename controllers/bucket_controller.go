/*


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

package controllers

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "github.com/yriveiro/gcs-bucket-operator/api/v1alpha1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
)

// StopReconciler indicates if the inner operation signaled to stop the reconcile
// or not.
type StopReconciler bool

// BucketReconciler reconciles a Bucket object
type BucketReconciler struct {
	client.Client
	StorageClient *storage.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
}

// +kubebuilder:rbac:groups=storage.k8s.riveiro.io,resources=buckets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.k8s.riveiro.io,resources=buckets/status,verbs=get;update;patch

// Reconcile reconciliates the resource state to the desire state
func (r *BucketReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithValues("bucket-operator", req.NamespacedName)

	l.Info(fmt.Sprintf("starting reconcile loop for namspace: %v", req.NamespacedName))
	defer l.Info(fmt.Sprintf("finish reconcile loop for namespace: %v", req.NamespacedName))

	b := &storagev1.Bucket{}

	if err := r.Get(ctx, req.NamespacedName, b); err != nil {
		if k8serr.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	if b.IsBeingDeleted() {
		l.Info(fmt.Sprintf("HandleFinalizer for namespace: %v", req.NamespacedName))
		if err := r.handleFinalizer(ctx, b); err != nil {
			r.Recorder.Event(b, corev1.EventTypeWarning, "Deleting finalizer", fmt.Sprintf("Failed to delete finalizer: %s", err))

			return ctrl.Result{}, fmt.Errorf("error when handling finalizer: %v", err)
		}

		r.Recorder.Event(b, corev1.EventTypeNormal, "Deleted", "Object finalizer is deleted")

		return ctrl.Result{}, nil
	}

	if !b.HasFinalizer(storagev1.BucketFinalizerName) {
		l.Info(fmt.Sprintf("addFinalizer for %v", req.NamespacedName))
		if err := r.addFinalizer(ctx, b); err != nil {
			r.Recorder.Event(b, corev1.EventTypeWarning, "Adding finalizer", fmt.Sprintf("Failed to add finalizer: %s", err))

			return ctrl.Result{}, fmt.Errorf("error when adding finalizer: %v", err)
		}

		r.Recorder.Event(b, corev1.EventTypeNormal, "Added", "Object finalizer is added")

		return ctrl.Result{}, nil
	}

	if !b.IsGCSBucketRefValid() {
		l.Info(fmt.Sprintf("operation forbidden, the resource %s is already binded to %s gcs bucket", b.GetName(), b.Status.GCSBucketRef))

		return ctrl.Result{}, nil
	}

	if err := r.create(ctx, b); err != nil {
		r.Recorder.Event(b, corev1.EventTypeWarning, "Creating bucket", fmt.Sprintf("failed to create bucket: %s", err))
		return ctrl.Result{}, fmt.Errorf("error when creating GCS Bucket: %v", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager setup the controller with a manager
func (r *BucketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1.Bucket{}).
		Complete(r)
}
