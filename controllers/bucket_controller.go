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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "gitlab.com/riveiro/bucket-operator/api/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
)

// StopReconciler indicates if we should continue the operation of reconcile
// or stop it.
type StopReconciler bool

// BucketOperatorOpNotAllowed custom error for operations not allowed by the
// defined state.
type BucketOperatorOpNotAllowed struct{}

func (e *BucketOperatorOpNotAllowed) Error() string {
	return "operation not allowed"
}

// Finalizer represents the finalizer to delete the bucket when the resource
// is deleted.
const Finalizer = "bucket.storage.k8s.riveiro.io/finalizer"

// CurrentBucketAnnotation is a annotation to keep tracking of the original
// bucket created by the resource.
const CurrentBucketAnnotation = "bucket.storage.k8s.riveiro.io/currentBucket"

// BucketOwner annotation to ensures no other CR can delete the bucket.
const BucketOwner = "bucket-storage-k8s-riveiro-io-owner"

// BucketReconciler reconciles a Bucket object
type BucketReconciler struct {
	client.Client
	StorageClient *storage.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
}

// +kubebuilder:rbac:groups=storage.k8s.riveiro.io,resources=buckets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.k8s.riveiro.io,resources=buckets/status,verbs=get;update;patch

// Reconcile reconciliates the resource state to the desire state
func (r *BucketReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithValues("bucket-operator", req.NamespacedName)

	var b storagev1.Bucket

	if err := r.Get(ctx, req.NamespacedName, &b); err != nil {
		if k8serr.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		l.Error(err, "unable to get state of resource")

		return ctrl.Result{}, err
	}

	if err := r.addFinalizer(&b); err != nil {
		return ctrl.Result{}, err
	}

	if stop, err := r.handleAnnotations(ctx, &b, l); err != nil {
		// Annotations raise an error, retry.
		return ctrl.Result{}, err
	} else if stop {
		// Stop the reconciliation as notified.
		return ctrl.Result{}, nil
	}

	if stop, err := r.handleDeleteNotification(ctx, &b, l); err != nil {
		// Deletion raise an error, retry
		return ctrl.Result{}, err
	} else if stop {
		// Stop reconciliation, the item is being deleted
		return ctrl.Result{}, nil
	}

	if _, err := r.handleCreateOrUpdateNotification(ctx, &b, l); err != nil {
		// CreateOrUpdate raise an error, retry
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager setup the controller with a manager
func (r *BucketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storagev1.Bucket{}).
		Complete(r)
}

func (r *BucketReconciler) addFinalizer(b *storagev1.Bucket) error {
	if b.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(b.ObjectMeta.Finalizers, Finalizer) {
			b.ObjectMeta.Finalizers = append(b.ObjectMeta.Finalizers, Finalizer)

			if err := r.Update(context.Background(), b); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *BucketReconciler) handleAnnotations(ctx context.Context, b *storagev1.Bucket, l logr.Logger) (StopReconciler, error) {
	a := b.GetAnnotations()

	if a == nil {
		a = make(map[string]string, 1)
	}

	if _, ok := a[CurrentBucketAnnotation]; !ok {
		l.Info("add annotation ", CurrentBucketAnnotation, b.Spec.Name)

		a[CurrentBucketAnnotation] = b.Spec.Name
		b.SetAnnotations(a)

		if err := r.Update(context.Background(), b); err != nil {
			// Bubble up the error to ensure retry
			return false, err
		}
	} else {
		if a[CurrentBucketAnnotation] != b.Spec.Name {
			// To ensure integrity it's not possible to update the Spec.Name
			// value once the GCP bucket is created.

			err := &BucketOperatorOpNotAllowed{}
			l.Error(err, "update Bucket.Spec.Name is not allowed, create a new resource instead")

			// Stop the reconciliation once the desired state is not allowed.
			return true, nil
		}
	}

	// OK, return control to Reconciler
	return false, nil
}

func (r *BucketReconciler) handleDeleteNotification(ctx context.Context, b *storagev1.Bucket, l logr.Logger) (StopReconciler, error) {
	if b.ObjectMeta.DeletionTimestamp.IsZero() {
		// No delete needed, continue reconciliation
		return false, nil

	}

	l.Info("deleted notification", "Bucket.Spec.Name", b.Spec.Name)

	if containsString(b.ObjectMeta.Finalizers, Finalizer) {
		if b.Spec.RemoveOnDelete {
			if err := r.deleteBucket(ctx, b, l); err != nil {
				// Bubble up to retry
				return false, err
			}
		}

		// remove our finalizer from the list and update it.
		b.ObjectMeta.Finalizers = removeString(b.ObjectMeta.Finalizers, Finalizer)
		if err := r.Update(context.Background(), b); err != nil {
			l.Error(err, "unable to remove finalizer", "finalizer", Finalizer)

			// Something wrong happened, bubble up error to retry
			return false, err
		}

		// Continue with the reconciliation process
		return true, nil
	}

	// Stop Reconciler as the item is being deleted
	return false, nil
}

func (r *BucketReconciler) deleteBucket(ctx context.Context, b *storagev1.Bucket, l logr.Logger) error {
	bkt := r.StorageClient.Bucket(b.Spec.Name)
	a, err := bkt.Attrs(ctx)

	if err == storage.ErrBucketNotExist {
		l.Info("bucket not exist, skipping deletion", "bucket", b.Spec.Name)

		return nil
	}

	if _, ok := a.Labels[BucketOwner]; !ok {
		err := &BucketOperatorOpNotAllowed{}

		l.Error(err, "missing 'owner' label, can't delete on safety")

		return err
	} else if a.Labels[BucketOwner] != b.GetObjectMeta().GetName() {
		err := &BucketOperatorOpNotAllowed{}

		l.Error(err, "mismatch in bucket ownership, can't delete on safety")

	}

	if err := bkt.Delete(ctx); err != nil {
		// If external API fails bubble up the error so it can be retried
		l.Error(err, "error deleting bucket from gcp", "bucket", b.Spec.Name)
		return err
	}

	l.Info("bucket deleted", "bucket", b.Spec.Name)

	return nil
}

func (r *BucketReconciler) handleCreateOrUpdateNotification(ctx context.Context, b *storagev1.Bucket, l logr.Logger) (StopReconciler, error) {
	bkt := r.StorageClient.Bucket(b.Spec.Name)

	if _, err := bkt.Attrs(ctx); err != nil {
		if err != storage.ErrBucketNotExist {
			l.Error(err, "unable to fetch bucket", "bucket.Spec.Name", b.Spec.Name)
			return false, err
		}

		l.Info("bucket not found, creating bucket", "bucket.Spec.Name", b.Spec.Name)

		labels := map[string]string{BucketOwner: b.ObjectMeta.GetName()}

		l.Info(fmt.Sprintf("%s", labels))

		if err = bkt.Create(ctx, b.Spec.Project, &storage.BucketAttrs{
			StorageClass: b.Spec.StorageClass,
			Location:     b.Spec.Location,
			Labels:       labels,
		}); err != nil {
			return false, err
		}

		var a map[string]string
		if a = b.GetAnnotations(); a == nil {
			a = make(map[string]string, 1)
		}

		l.Info("add annotation", "currentBucket", a[CurrentBucketAnnotation])

		a[CurrentBucketAnnotation] = b.Spec.Name
		b.SetAnnotations(a)

		if err := r.Update(context.Background(), b); err != nil {
			return false, err
		}

		return true, nil
	}

	l.Info("bucket already exists, skip reconcile", "bucket.Spec.Name", b.Spec.Name)

	// We are done and we can Stop the Reconciler
	return true, nil
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
