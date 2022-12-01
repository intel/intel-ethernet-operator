// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeviceSelector struct {
	// VendorId of devices to be selected. If value is not set, then CLV cards with any VendorId are selected
	VendorID string `json:"vendorId,omitempty"`
	// DeviceId of devices to be selected. If value is not set, then CLV cards with any DeviceId are selected
	DeviceID string `json:"deviceId,omitempty"`
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{4}:[a-fA-F0-9]{2}:[01][a-fA-F0-9]\.[0-7]$`
	// PciAdress of devices to be selected. If value is not set, then CLV cards with any PciAddress are selected
	PCIAddress string `json:"pciAddress,omitempty"`
}

type DeviceConfig struct {
	// Path to .zip DDP package to be applied
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\.\-\/]+
	DDPURL string `json:"ddpURL,omitempty"`
	// MD5 checksum of .zip DDP package
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{32}$`
	DDPChecksum string `json:"ddpChecksum,omitempty"`

	// Path to .tar.gz Firmware (NVMUpdate package) to be applied
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\.\-\/]+
	FWURL string `json:"fwURL,omitempty"`
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{32}$`
	// MD5 checksum of .tar.gz Firmware
	FWChecksum string `json:"fwChecksum,omitempty"`
}

// EthernetClusterConfigSpec defines the desired state of EthernetClusterConfig
type EthernetClusterConfigSpec struct {
	// Selector for nodes. If value is not set, then configuration is applied to all nodes with CLV cards in cluster
	NodeSelector map[string]string `json:"nodeSelectors,omitempty"`
	// Selector for devices on nodes. If value is not set, then configuration is applied to all CLV cards on selected nodes
	DeviceSelector DeviceSelector `json:"deviceSelector,omitempty"`
	// Contains configuration which will be applied to selected devices
	DeviceConfig DeviceConfig `json:"deviceConfig"`

	// Higher priority policies can override lower ones.
	//If several ClusterConfigs have same Priority, then operator will apply ClusterConfig with highest CreationTimestamp (newest one)
	Priority int `json:"priority,omitempty"`
}

// EthernetClusterConfigStatus defines the observed state of EthernetClusterConfig
type EthernetClusterConfigStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=ecc

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
