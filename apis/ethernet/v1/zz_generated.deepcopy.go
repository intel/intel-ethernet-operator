// +build !ignore_autogenerated

// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DDPInfo) DeepCopyInto(out *DDPInfo) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DDPInfo.
func (in *DDPInfo) DeepCopy() *DDPInfo {
	if in == nil {
		return nil
	}
	out := new(DDPInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Device) DeepCopyInto(out *Device) {
	*out = *in
	out.Firmware = in.Firmware
	out.DDP = in.DDP
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Device.
func (in *Device) DeepCopy() *Device {
	if in == nil {
		return nil
	}
	out := new(Device)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeviceConfig) DeepCopyInto(out *DeviceConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeviceConfig.
func (in *DeviceConfig) DeepCopy() *DeviceConfig {
	if in == nil {
		return nil
	}
	out := new(DeviceConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeviceNodeConfig) DeepCopyInto(out *DeviceNodeConfig) {
	*out = *in
	out.DeviceConfig = in.DeviceConfig
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeviceNodeConfig.
func (in *DeviceNodeConfig) DeepCopy() *DeviceNodeConfig {
	if in == nil {
		return nil
	}
	out := new(DeviceNodeConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeviceSelector) DeepCopyInto(out *DeviceSelector) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeviceSelector.
func (in *DeviceSelector) DeepCopy() *DeviceSelector {
	if in == nil {
		return nil
	}
	out := new(DeviceSelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EthernetClusterConfig) DeepCopyInto(out *EthernetClusterConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EthernetClusterConfig.
func (in *EthernetClusterConfig) DeepCopy() *EthernetClusterConfig {
	if in == nil {
		return nil
	}
	out := new(EthernetClusterConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EthernetClusterConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EthernetClusterConfigList) DeepCopyInto(out *EthernetClusterConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EthernetClusterConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EthernetClusterConfigList.
func (in *EthernetClusterConfigList) DeepCopy() *EthernetClusterConfigList {
	if in == nil {
		return nil
	}
	out := new(EthernetClusterConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EthernetClusterConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EthernetClusterConfigSpec) DeepCopyInto(out *EthernetClusterConfigSpec) {
	*out = *in
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	out.DeviceSelector = in.DeviceSelector
	out.DeviceConfig = in.DeviceConfig
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EthernetClusterConfigSpec.
func (in *EthernetClusterConfigSpec) DeepCopy() *EthernetClusterConfigSpec {
	if in == nil {
		return nil
	}
	out := new(EthernetClusterConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EthernetClusterConfigStatus) DeepCopyInto(out *EthernetClusterConfigStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EthernetClusterConfigStatus.
func (in *EthernetClusterConfigStatus) DeepCopy() *EthernetClusterConfigStatus {
	if in == nil {
		return nil
	}
	out := new(EthernetClusterConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EthernetNodeConfig) DeepCopyInto(out *EthernetNodeConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EthernetNodeConfig.
func (in *EthernetNodeConfig) DeepCopy() *EthernetNodeConfig {
	if in == nil {
		return nil
	}
	out := new(EthernetNodeConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EthernetNodeConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EthernetNodeConfigList) DeepCopyInto(out *EthernetNodeConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EthernetNodeConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EthernetNodeConfigList.
func (in *EthernetNodeConfigList) DeepCopy() *EthernetNodeConfigList {
	if in == nil {
		return nil
	}
	out := new(EthernetNodeConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EthernetNodeConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EthernetNodeConfigSpec) DeepCopyInto(out *EthernetNodeConfigSpec) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make([]DeviceNodeConfig, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EthernetNodeConfigSpec.
func (in *EthernetNodeConfigSpec) DeepCopy() *EthernetNodeConfigSpec {
	if in == nil {
		return nil
	}
	out := new(EthernetNodeConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EthernetNodeConfigStatus) DeepCopyInto(out *EthernetNodeConfigStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Devices != nil {
		in, out := &in.Devices, &out.Devices
		*out = make([]Device, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EthernetNodeConfigStatus.
func (in *EthernetNodeConfigStatus) DeepCopy() *EthernetNodeConfigStatus {
	if in == nil {
		return nil
	}
	out := new(EthernetNodeConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FirmwareInfo) DeepCopyInto(out *FirmwareInfo) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FirmwareInfo.
func (in *FirmwareInfo) DeepCopy() *FirmwareInfo {
	if in == nil {
		return nil
	}
	out := new(FirmwareInfo)
	in.DeepCopyInto(out)
	return out
}
