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

package v1

import (
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
}

// +kubebuilder:object:root=true

// Bucket is theSchema for the buckets API
type Bucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BucketSpec   `json:"spec,omitempty"`
	Status BucketStatus `json:"status,omitempty"`
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
