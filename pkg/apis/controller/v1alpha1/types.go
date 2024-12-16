/*
Copyright 2017 The Kubernetes Authors.

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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BlueGreenDeploymentSpec is the spec for a BlueGreenDeployment resource
type BlueGreenDeploymentSpec struct {
	// Replicas specifies the desired number of replicas, determining the number of pods
	// for both the Blue and Green ReplicaSets. If not specified, it defaults to 1.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// PodSpec specifies the desired behavior of pods in the Blue and Green ReplicaSets,
	// including the container image configuration and other settings.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	PodSpec v1.PodSpec `json:"podSpec,omitempty"`
}

// BlueGreenDeploymentStatus is the status for a BlueGreenDeployment resource
type BlueGreenDeploymentStatus struct {
	// ActiveReplicaSetColor specifies the color of the currently active ReplicaSet.
	// It is used to determine the label selectors for the service, effectively routing
	// traffic to the chosen ReplicaSet. Initially, it defaults to "blue".
	// +optional
	ActiveReplicaSetColor string `json:"activeReplicaSetColor,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BlueGreenDeployment is a specification for a BlueGreenDeployment resource
type BlueGreenDeployment struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the specification of the desired behavior of the BlueGreenDeployment object.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Spec BlueGreenDeploymentSpec `json:"spec,omitempty"`

	// Status is the most recently observed status of the BlueGreenDeployment object.
	// This data may be out of date by some window of time.
	// Populated by the system.
	// Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Status BlueGreenDeploymentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BlueGreenDeploymentList is a list of BlueGreenDeployment resources
type BlueGreenDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []BlueGreenDeployment `json:"items"`
}
