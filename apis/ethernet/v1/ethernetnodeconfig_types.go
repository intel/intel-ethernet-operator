// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeviceNodeConfig struct {
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{4}:[a-fA-F0-9]{2}:[01][a-fA-F0-9]\.[0-7]$`
	PCIAddress   string       `json:"PCIAddress"`
	DeviceConfig DeviceConfig `json:"deviceConfig"`
}

// EthernetNodeConfigSpec defines the desired state of EthernetNodeConfig
type EthernetNodeConfigSpec struct {
	Config      []DeviceNodeConfig `json:"config,omitempty"`
	DrainSkip   bool               `json:"drainSkip,omitempty"`
	ForceReboot bool               `json:"forceReboot,omitempty"`
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
	VendorID      string       `json:"vendorID"`
	DeviceID      string       `json:"deviceID"`
	PCIAddress    string       `json:"PCIAddress"`
	Name          string       `json:"name"`
	Driver        string       `json:"driver"`
	DriverVersion string       `json:"driverVersion"`
	Firmware      FirmwareInfo `json:"firmware"`
	DDP           DDPInfo      `json:"DDP"`
}

// EthernetNodeConfigStatus defines the observed state of EthernetNodeConfig
type EthernetNodeConfigStatus struct {
	// Provides information about device update status
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	Devices    []Device           `json:"devices,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=enc
// +kubebuilder:printcolumn:name="Update",type=string,JSONPath=`.status.conditions[?(@.type=="Updated")].reason`

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
