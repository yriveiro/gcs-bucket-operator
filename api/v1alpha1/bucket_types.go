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

package v1alpha1

import (
	"cloud.google.com/go/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BucketSpec defines the desired state of Bucket
type BucketSpec struct {
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`

	// Defines the project where the bucket will be created.
	// +kubebuilder:validation:Required
	Project string `json:"project,omitempty"`

	// Defines the location where the bucket will be created.
	// https://cloud.google.com/storage/docs/locations
	// +kubebuilder:validation:Required
	Location string `json:"location,omitempty"`

	// Defines the kind of the storage to use.
	// https://cloud.google.com/storage/docs/storage-classes
	// +kubebuilder:validation:Required
	StorageClass string `json:"storageClass,omitempty"` //

	// Defines if we gcs bucket should be delete with the CR.
	// +kubebuilder:validation:Required
	RemoveOnDelete bool `json:"removeOnDelete,omitempty"` //
}

// BucketStatus defines the observed state of Bucket
type BucketStatus struct {
	GCSBucketRef string `json:"gcsBucketRef,omitempty"`
}

// BucketFinalizerName is the name of the bucket finalizer
const BucketFinalizerName = "bucket.storage.k8s.riveiro.io/finalizer"

// BucketOwnerLabel is a label to ensure we're labeling buckets in GCS
// with the proper owner and allow tracing.
const BucketOwnerLabel = "bucket-storage-k8s-riveiro-io-owner"

// BucketAnnotation is a annotation to keep tracking of the original
// bucket created by the resource.
const BucketAnnotation = "storage.k8s.riveiro.io/bucket"

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=bk
// +kubebuilder:subresource:status

// Bucket is theSchema for the buckets API
type Bucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BucketSpec   `json:"spec,omitempty"`
	Status BucketStatus `json:"status,omitempty"`
}

// IsBeingDeleted returns true if a deletion timestamp is set
func (b *Bucket) IsBeingDeleted() bool {
	return !b.ObjectMeta.DeletionTimestamp.IsZero()
}

// HasFinalizer returns true if the item has the specified finalizer
func (b *Bucket) HasFinalizer(finalizerName string) bool {
	return containsString(b.ObjectMeta.Finalizers, finalizerName)
}

// AddFinalizer adds the specified finalizer
func (b *Bucket) AddFinalizer(finalizerName string) {
	b.ObjectMeta.Finalizers = append(b.ObjectMeta.Finalizers, finalizerName)
}

// RemoveFinalizer removes the specified finalizer
func (b *Bucket) RemoveFinalizer(finalizerName string) {
	b.ObjectMeta.Finalizers = removeString(b.ObjectMeta.Finalizers, finalizerName)
}

// Owned checks if the resource is owner of the bucket
func (b *Bucket) Owned(a *storage.BucketAttrs) bool {
	if _, ok := a.Labels[BucketOwnerLabel]; !ok {
		return false
	}

	return a.Labels[BucketOwnerLabel] == b.GetObjectMeta().GetName()
}

// IsGCSBucketRefValid check if the resource already has a ref with a
// GCS bucket
func (b *Bucket) IsGCSBucketRefValid() bool {
	if b.Status.GCSBucketRef == "" {
		// not binded with any GCS bucket yet.
		return true
	}

	return b.Status.GCSBucketRef == b.Spec.Name
}

// +kubebuilder:object:root=true

// BucketList contains a list of Bucket
type BucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bucket `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Bucket{}, &BucketList{})
}
