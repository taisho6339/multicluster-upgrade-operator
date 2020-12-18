/*
Copyright 2020 taisho6339.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterVersionSpec defines the desired state of ClusterVersion
type ClusterVersionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MinItems=2
	Clusters    []Cluster   `json:"clusters,omitempty"`
	OpsEndpoint OpsEndpoint `json:"opsEndpoint"`

	// +kubebuilder:validation:Minimum=1
	RequiredAvailableCount int `json:"requiredAvailableCount"`
}

// Cluster defines the cluster spec
// ID is specific provider's cluster id.
// For instance, GKE represents "projects/%s/locations/%s/clusters/%s"
type Cluster struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// OpsEndpoint defines the endpoint spec for the gRPC server which performs specific operations.
type OpsEndpoint struct {
	Endpoint string `json:"endpoint"`
	Insecure bool   `json:"insecure"`
}

// ClusterVersionStatus defines the observed state of ClusterVersion
type ClusterVersionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ClusterID     string `json:"ClusterID"`
	OperationID   string `json:"OperationID"`
	OperationType string `json:"OperationType"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ClusterVersion is the Schema for the clusterversions API
type ClusterVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterVersionSpec   `json:"spec,omitempty"`
	Status ClusterVersionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterVersionList contains a list of ClusterVersion
type ClusterVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterVersion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterVersion{}, &ClusterVersionList{})
}

func (in *ClusterVersionStatus) ResetStatus() {
	in.OperationID = ""
	in.OperationType = ""
	in.ClusterID = ""
}
