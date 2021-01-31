/*

Copyright 2021 Yago Riveiro <yago.riveiro@gmail.com>

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

	storagev1 "github.com/yriveiro/gcs-bucket-operator/api/v1alpha1"
)

func (r *BucketReconciler) addFinalizer(ctx context.Context, b *storagev1.Bucket) error {
	b.AddFinalizer(storagev1.BucketFinalizerName)
	return r.Update(ctx, b)
}

func (r *BucketReconciler) handleFinalizer(ctx context.Context, b *storagev1.Bucket) error {
	if !b.HasFinalizer(storagev1.BucketFinalizerName) {
		return nil
	}

	if err := r.delete(ctx, b); err != nil {
		return err
	}

	b.RemoveFinalizer(storagev1.BucketFinalizerName)
	return r.Update(ctx, b)
}
