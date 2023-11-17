// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2023 Intel Corporation

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterFlowConfigSpec defines the desired state of ClusterFlowConfig
type ClusterFlowConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// This is a label selector which selects Pods. This field follows standard label
	// selector semantics; if present but empty, it selects all pods.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`

	// Rules is a list of FlowCreate rules
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Rules []*ClusterFlowRule `json:"rules,omitempty"`
}

// ClusterFlowConfigStatus defines the observed state of ClusterFlowConfig
type ClusterFlowConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterFlowConfig is the Schema for the clusterflowconfigs API
//+operator-sdk:csv:customresourcedefinitions:resources={{Pod,v1,flowconfig-daemon}}
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

// ClusterFlowRules struct for flow rules creation and validation
type ClusterFlowRule struct {
	Attr    *FlowAttr            `json:"attr,omitempty"`
	Pattern []*FlowItem          `json:"pattern,omitempty"`
	Action  []*ClusterFlowAction `json:"action,omitempty"`
}
