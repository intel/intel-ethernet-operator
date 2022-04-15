// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeviceNodeConfig struct {
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{4}:[a-fA-F0-9]{2}:[01][a-fA-F0-9]\.[0-7]$`
	// PciAddress of device
	PCIAddress string `json:"PCIAddress"`
	// Configuration which will be applied to this device
	DeviceConfig DeviceConfig `json:"deviceConfig"`
}

// EthernetNodeConfigSpec defines the desired state of EthernetNodeConfig
type EthernetNodeConfigSpec struct {
	// Contains mapping of PciAddress to Configuration which will be applied to device on particular PciAddress
	Config []DeviceNodeConfig `json:"config,omitempty"`
	// Skips drain process when true; default false. Should be true if operator is running on SNO
	DrainSkip bool `json:"drainSkip,omitempty"`
}

type FirmwareInfo struct {
	MAC     string `json:"MAC"`
	Version string `json:"version"`
}

type DDPInfo struct {
	PackageName string `json:"packageName"`
	Version     string `json:"version"`
	TrackID     string `json:"trackId"`
}

type Device struct {
	// VendorId of card
	VendorID string `json:"vendorID"`
	// DeviceId of card
	DeviceID string `json:"deviceID"`
	// PciAddress of card
	PCIAddress string `json:"PCIAddress"`
	// Contains human-readable name of card
	Name string `json:"name"`
	// Contains name of driver which is managing card
	Driver string `json:"driver"`
	// Version of driver
	DriverVersion string `json:"driverVersion"`
	// FirmwareInfo contains information about MAC address of card and loaded version of Firmware
	Firmware FirmwareInfo `json:"firmware"`
	// DDPInfo contains information about loaded DDP profile
	DDP DDPInfo `json:"DDP"`
}

// EthernetNodeConfigStatus defines the observed state of EthernetNodeConfig
type EthernetNodeConfigStatus struct {
	// Provides information about device update status
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Contains list of supported CLV cards and details about them
	Devices []Device `json:"devices,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=enc
//+kubebuilder:printcolumn:name="Update",type=string,JSONPath=`.status.conditions[?(@.type=="Updated")].reason`
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Updated")].message`

// EthernetNodeConfig is the Schema for the ethernetnodeconfigs API
type EthernetNodeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EthernetNodeConfigSpec   `json:"spec,omitempty"`
	Status EthernetNodeConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EthernetNodeConfigList contains a list of EthernetNodeConfig
type EthernetNodeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EthernetNodeConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EthernetNodeConfig{}, &EthernetNodeConfigList{})
}
