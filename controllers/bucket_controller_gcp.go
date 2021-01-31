package controllers

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	storagev1 "github.com/yriveiro/gcs-bucket-operator/api/v1alpha1"
)

func (r *BucketReconciler) delete(ctx context.Context, b *storagev1.Bucket) error {
	r.Log.Info(fmt.Sprintf("deleting gcs bucket: %s from namespace: %s", b.GetName(), b.GetNamespace()))

	bkt := r.StorageClient.Bucket(b.Spec.Name)
	a, err := bkt.Attrs(ctx)

	if err == storage.ErrBucketNotExist {
		r.Log.Info(fmt.Sprintf("bucket %s not exist, skipping deletion", b.Spec.Name))

		return nil
	}

	if b.Spec.RemoveOnDelete {
		if !b.Owned(a) {
			err := fmt.Errorf(fmt.Sprintf("resource: %s not owner of the gcs bucket", b.Spec.Name))
			r.Log.Error(err, "deletion aborted")

			return err
		}

		if err := bkt.Delete(ctx); err != nil {
			r.Log.Error(err, fmt.Sprintf("error deleting bucket: %s from gcp", b.Spec.Name))
			return err
		}

	}

	r.Log.Info(fmt.Sprintf("bucket %s deleted", b.Spec.Name))

	return nil
}

func (r *BucketReconciler) create(ctx context.Context, b *storagev1.Bucket) error {
	bkt := r.StorageClient.Bucket(b.Spec.Name)
	a, err := bkt.Attrs(ctx)

	if err == nil {
		if !b.Owned(a) {
			r.Log.Info(fmt.Sprintf("gcs bucket %s exists but %s is not owner", b.Spec.Name, b.GetName()))

			return nil
		}

		r.Log.Info(fmt.Sprintf("gcs bucket %s exists and %s is the owner", b.Spec.Name, b.GetName()))
		if b.Status.GCSBucketRef == "" {
			b.Status.GCSBucketRef = b.Spec.Name

			return r.Update(ctx, b)
		}

		return nil
	}

	if err != storage.ErrBucketNotExist {
		r.Log.Error(err, fmt.Sprintf("unable to fetch gcs bucket %s status", b.Spec.Name))

		return err
	}

	r.Log.Info(fmt.Sprintf("gcs bucket %s not found, creating", b.Spec.Name))

	labels := map[string]string{storagev1.BucketOwnerLabel: b.ObjectMeta.GetName()}

	bktAttr := &storage.BucketAttrs{
		StorageClass: b.Spec.StorageClass,
		Location:     b.Spec.Location,
		Labels:       labels,
	}

	if err := bkt.Create(ctx, b.Spec.Project, bktAttr); err != nil {
		r.Log.Error(err, fmt.Sprintf("unable to create gcs bucket %s", b.Spec.Name))

		return err
	}

	b.Status.GCSBucketRef = b.Spec.Name

	return r.Update(ctx, b)
}
