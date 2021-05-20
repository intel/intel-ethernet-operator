// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeviceSelectors struct {
	VendorID   []string `json:"vendorId,omitempty"`
	DeviceID   []string `json:"deviceId,omitempty"`
	PCIAddress []string `json:"pciAddress,omitempty"`
	Drivers    []string `json:"drivers,omitempty"`
}

type DeviceConfig struct {
	// DDP package to be applied
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\.\-\/]+
	DDPURL string `json:"ddpURL,omitempty"`
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{32}$`
	DDPChecksum string `json:"ddpChecksum,omitempty"`

	// Firmware (NVMUpdate package) to be applied
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\.\-\/]+
	FWURL string `json:"fwURL,omitempty"`
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{32}$`
	FWChecksum string `json:"fwChecksum,omitempty"`

	// Force DDP and/or FW application given incompatibility
	Force bool `json:"force,omitempty"`
}

type SyncStatus string

var (
	// InProgressSync indicates that the synchronization of the CR is in progress
	InProgressSync SyncStatus = "InProgress"
	// SucceededSync indicates that the synchronization of the CR succeeded
	SucceededSync SyncStatus = "Succeeded"
	// FailedSync indicates that the synchronization of the CR failed
	FailedSync SyncStatus = "Failed"
	// IgnoredSync indicates that the CR is ignored
	IgnoredSync SyncStatus = "Ignored"
)

// EthernetClusterConfigSpec defines the desired state of EthernetClusterConfig
type EthernetClusterConfigSpec struct {
	// Select the nodes
	NodeSelectors map[string]string `json:"nodeSelectors,omitempty"`
	// Select the devices on nodes
	DeviceSelector DeviceSelectors `json:"deviceSelector,omitempty"`

	DeviceConfig DeviceConfig `json:"deviceConfig"`

	Priority  int  `json:"priority,omitempty"`
	DrainSkip bool `json:"drainSkip,omitempty"`
}

// EthernetClusterConfigStatus defines the observed state of EthernetClusterConfig
type EthernetClusterConfigStatus struct {
	// Indicates the synchronization status of the CR
	// +operator-sdk:csv:customresourcedefinitions:type=status
	SyncStatus    SyncStatus `json:"syncStatus,omitempty"`
	LastSyncError string     `json:"lastSyncError,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EthernetClusterConfig is the Schema for the ethernetclusterconfigs API
type EthernetClusterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EthernetClusterConfigSpec   `json:"spec,omitempty"`
	Status EthernetClusterConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EthernetClusterConfigList contains a list of EthernetClusterConfig
type EthernetClusterConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EthernetClusterConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EthernetClusterConfig{}, &EthernetClusterConfigList{})
}
