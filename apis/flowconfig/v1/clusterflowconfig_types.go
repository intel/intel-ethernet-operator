// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterFlowConfigSpec defines the desired state of ClusterFlowConfig
type ClusterFlowConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ClusterFlowConfig. Edit clusterflowconfig_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// ClusterFlowConfigStatus defines the observed state of ClusterFlowConfig
type ClusterFlowConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterFlowConfig is the Schema for the clusterflowconfigs API
type ClusterFlowConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterFlowConfigSpec   `json:"spec,omitempty"`
	Status ClusterFlowConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterFlowConfigList contains a list of ClusterFlowConfig
type ClusterFlowConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterFlowConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterFlowConfig{}, &ClusterFlowConfigList{})
}
