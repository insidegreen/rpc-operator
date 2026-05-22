/*
Copyright 2026.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineClusterSpec defines the desired state of a PipelineCluster: a named
// group of N Redpanda Connect instances running in streams mode. Phase 1 only
// stands up the instances; assigning pipelines (clusterRef) is Phase 2.
type PipelineClusterSpec struct {
	// Replicas is the number of Redpanda Connect streams-mode instances.
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Image is the Redpanda Connect container image used for each instance.
	// +kubebuilder:default="docker.redpanda.com/redpandadata/connect:4"
	// +optional
	Image string `json:"image,omitempty"`

	// JSONLogging forces structured JSON logs on every instance. Phase 3 relies
	// on this to filter logs per stream; default on.
	// +kubebuilder:default=true
	// +optional
	JSONLogging bool `json:"jsonLogging,omitempty"`

	// Resources sets CPU/memory requests and limits per instance container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// PipelineClusterPhase reports the high-level lifecycle stage of a cluster.
// +kubebuilder:validation:Enum=Pending;Ready;Degraded
type PipelineClusterPhase string

const (
	ClusterPhasePending  PipelineClusterPhase = "Pending"
	ClusterPhaseReady    PipelineClusterPhase = "Ready"
	ClusterPhaseDegraded PipelineClusterPhase = "Degraded"
)

// PipelineClusterStatus defines the observed state of a PipelineCluster.
type PipelineClusterStatus struct {
	// Phase is the high-level lifecycle stage of the cluster.
	// +optional
	Phase PipelineClusterPhase `json:"phase,omitempty"`

	// ReadyReplicas mirrors the StatefulSet's ready replica count.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// ObservedGeneration is the .metadata.generation this status reflects.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PipelineCluster is the Schema for the pipelineclusters API.
type PipelineCluster struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec PipelineClusterSpec `json:"spec"`

	// +optional
	Status PipelineClusterStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PipelineClusterList contains a list of PipelineCluster.
type PipelineClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PipelineCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PipelineCluster{}, &PipelineClusterList{})
}
