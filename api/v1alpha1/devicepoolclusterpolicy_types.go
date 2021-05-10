// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeviceConfig struct {
	// Network remote DDP package to be applied
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\.\-\/]+
	DDPURL string `json:”ddpURL,omitempty”`
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{32}$`
	DDPChecksum string `json:”ddpChecksum,omitempty”`

	// Network remote Firmware to be applied
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\.\-\/]+
	FWURL string `json:”fwURL,omitempty”`
	// +kubebuilder:validation:Pattern=`^[a-fA-F0-9]{32}$`
	FWChecksum string `json:”fwChecksum,omitempty”`

	// Force DDP and/or FW application given incompatibility
	Force bool `json:”force,omitempty”`
}

type DeviceSelectors struct {
	Vendors  []string `json:"vendors,omitempty"`
	Devices  []string `json:"devices,omitempty"`
	PciAddrs []string `json:"pciAddrs,omitempty"`
	Drivers  []string `json:"drivers,omitempty"`
}

// DevicePoolClusterPolicySpec defines the desired state of DevicePoolClusterPolicy
type DevicePoolClusterPolicySpec struct {
	// Select the nodes
	NodeSelectors []string `json:"nodeSelector,omitempty"`
	// Select devices on nodes
	DeviceSelector DeviceSelectors `json:"deviceSelector,omitempty"`
	DeviceConfig   DeviceConfig    `json:"deviceConfig"`
}

// DevicePoolClusterPolicyStatus defines the observed state of DevicePoolClusterPolicy
type DevicePoolClusterPolicyStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DevicePoolClusterPolicy is the Schema for the devicepoolclusterpolicies API
type DevicePoolClusterPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevicePoolClusterPolicySpec   `json:"spec,omitempty"`
	Status DevicePoolClusterPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DevicePoolClusterPolicyList contains a list of DevicePoolClusterPolicy
type DevicePoolClusterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DevicePoolClusterPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DevicePoolClusterPolicy{}, &DevicePoolClusterPolicyList{})
}
