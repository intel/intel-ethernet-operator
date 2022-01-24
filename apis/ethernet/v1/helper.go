// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

func (ds DeviceSelector) Matches(d Device) bool {
	if ds.VendorID != "" && ds.VendorID != d.VendorID {
		return false
	}
	if ds.PCIAddress != "" && ds.PCIAddress != d.PCIAddress {
		return false
	}
	if ds.DeviceID != "" && ds.DeviceID != d.DeviceID {
		return false
	}
	return true
}
