// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2023 Intel Corporation

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FlowConfigNodeAgentDeploymentSpec defines the desired state of FlowConfigNodeAgentDeployment
type FlowConfigNodeAgentDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// NADAnnotation is the name reference to Network Attachement Definition required by UFT container
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	NADAnnotation string `json:"NADAnnotation,omitempty"`
	// DCFVfPoolName is the name reference to CVL admin VF pool
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	DCFVfPoolName string `json:"DCFVfPoolName,omitempty"`
}

// FlowConfigNodeAgentDeploymentStatus defines the observed state of FlowConfigNodeAgentDeployment
type FlowConfigNodeAgentDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FlowConfigNodeAgentDeployment is the Schema for the flowconfignodeagentdeployments API
//+operator-sdk:csv:customresourcedefinitions:resources={{Pod,v1,flowconfig-daemon}}
type FlowConfigNodeAgentDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlowConfigNodeAgentDeploymentSpec   `json:"spec,omitempty"`
	Status FlowConfigNodeAgentDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FlowConfigNodeAgentDeploymentList contains a list of FlowConfigNodeAgentDeployment
type FlowConfigNodeAgentDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FlowConfigNodeAgentDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FlowConfigNodeAgentDeployment{}, &FlowConfigNodeAgentDeploymentList{})
}
