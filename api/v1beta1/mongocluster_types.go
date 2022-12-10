/*
Copyright 2022.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Storage struct {
	// Size of the persistent volume claim
	Size             string `json:"size"`
	StorageClassName string `json:"storageClassName,omitempty"`
}

type ResourcesRequestLimit struct {
	// Request resource requests and limits
	Request string `json:"request"`
	// Memory resource requests and limits
	Limit string `json:"limit"`
}

type Resources struct {
	//Resource requests
	CPU ResourcesRequestLimit `json:"cpu"`
	//Resource limits
	Memory ResourcesRequestLimit `json:"memory"`
}

type MongoAuth struct {
	Password           string `json:"password,omitempty"`
	ExistingSecretName string `json:"existingSecret,omitempty"`
}

// MongoClusterSpec defines the desired state of MongoCluster
type MongoClusterSpec struct {
	Image        string    `json:"image,omitempty"`
	Replicas     int32     `json:"replicas,omitempty"`
	DatabaseName string    `json:"database,omitempty"`
	Storage      Storage   `json:"storage,omitempty"`
	Resources    Resources `json:"resources,omitempty"`
	Auth         MongoAuth `json:"auth,omitempty"`
}

// MongoClusterStatus defines the observed state of MongoCluster
type MongoClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MongoCluster is the Schema for the mongoclusters API
type MongoCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoClusterSpec   `json:"spec,omitempty"`
	Status MongoClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MongoClusterList contains a list of MongoCluster
type MongoClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MongoCluster{}, &MongoClusterList{})
}
