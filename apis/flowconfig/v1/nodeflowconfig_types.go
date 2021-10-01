// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeFlowConfigSpec defines the desired state of NodeFlowConfig
type NodeFlowConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Rules is a list of FlowCreate rules
	Rules []*FlowRules `json:"rules,omitempty"`
}

// NodeFlowConfigStatus defines the observed state of NodeFlowConfig
type NodeFlowConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	PortInfo []PortsInformation `json:"portInfo"`
	// Last applied rules
	Rules      []*FlowRules   `json:"rules,omitempty"`
	SyncStatus SyncStatusType `json:"syncStatus,omitempty"`
	SyncMsg    string         `json:"syncMsg,omitempty"`
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

type SyncStatusType string

const (
	SyncError    SyncStatusType = "Error"
	SyncSuccess  SyncStatusType = "Success"
	SyncProgress SyncStatusType = "Progress"
)

// PortsInformation defines port information
type PortsInformation struct {
	PortId   uint32 `json:"portId"`
	PortPci  string `json:"portPci,omitempty"`
	PortMode string `json:"portMode,omitempty"`
}

// FlowRules struct for flow rules creation and validation
type FlowRules struct {
	PortId  uint32        `json:"portId,omitempty"`
	Attr    *FlowAttr     `json:"attr,omitempty"`
	Pattern []*FlowItem   `json:"pattern,omitempty"`
	Action  []*FlowAction `json:"action,omitempty"`
}

// FlowAttr defines Flow rule attributes
type FlowAttr struct {
	Group    uint32 `json:"group,omitempty"`
	Priority uint32 `json:"priority,omitempty"`
	Ingress  uint32 `json:"ingress,omitempty"`
	Egress   uint32 `json:"egress,omitempty"`
	Transfer uint32 `json:"transfer,omitempty"`
	Reserved uint32 `json:"reserved,omitempty"`
}

// FlowItem defines flow pattern definition
type FlowItem struct {
	Type string `json:"type,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Spec *runtime.RawExtension `json:"spec,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Last *runtime.RawExtension `json:"last,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Mask *runtime.RawExtension `json:"mask,omitempty"`
}

// FlowAction defines flow actions
type FlowAction struct {
	Type string `json:"type,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Conf *runtime.RawExtension `json:"conf,omitempty"`
}
