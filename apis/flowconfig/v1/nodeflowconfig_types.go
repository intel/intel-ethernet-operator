// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeFlowConfigSpec defines the desired state of NodeFlowConfig
type NodeFlowConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// NodeFlowConfigStatus defines the observed state of NodeFlowConfig
type NodeFlowConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NodeFlowConfig is the Schema for the nodeflowconfigs API
type NodeFlowConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeFlowConfigSpec   `json:"spec,omitempty"`
	Status NodeFlowConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NodeFlowConfigList contains a list of NodeFlowConfig
type NodeFlowConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeFlowConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeFlowConfig{}, &NodeFlowConfigList{})
}
