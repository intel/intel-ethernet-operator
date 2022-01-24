// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeviceSelector struct {
	VendorID string `json:"vendorId,omitempty"`
	DeviceID string `json:"deviceId,omitempty"`
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{4}:[a-fA-F0-9]{2}:[01][a-fA-F0-9]\.[0-7]$`
	PCIAddress string `json:"pciAddress,omitempty"`
	Driver     string `json:"driver,omitempty"`
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
}

// EthernetClusterConfigSpec defines the desired state of EthernetClusterConfig
type EthernetClusterConfigSpec struct {
	// Select the nodes
	NodeSelector map[string]string `json:"nodeSelectors,omitempty"`
	// Select the devices on nodes
	DeviceSelector DeviceSelector `json:"deviceSelector,omitempty"`

	DeviceConfig DeviceConfig `json:"deviceConfig"`

	Priority  int  `json:"priority,omitempty"`
	DrainSkip bool `json:"drainSkip,omitempty"`
	// Force reboot after DDP update. Recommended for clusters, on which ControlPlane is running on E810 cards.
	ForceReboot bool `json:"forceReboot,omitempty"`
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
