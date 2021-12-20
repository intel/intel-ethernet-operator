// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package daemon

import (
	"fmt"
	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"
	"github.com/otcshare/intel-ethernet-operator/pkg/utils"
)

const (
	compatibilityWildard = "*"
)

var (
	compatMapPath    = "./devices.json"
	compatibilityMap *CompatibilityMap
)

func (r *NodeConfigReconciler) verifyCompatibility(fwPath, ddpPath string, dev ethernetv1.Device, force bool) error {
	log := r.log.WithName("verifyCompatibility")

	if force {
		log.V(2).Info("Force flag provided - skipping compatibility check for", "device", dev.PCIAddress)
		return nil
	}

	fwVer, err := r.fwUpdater.getFWVersion(fwPath, dev)
	if err != nil {
		log.Error(err, "Failed to retrieve firmware version")
		return err
	}

	ddpVer, err := r.ddpUpdater.getDDPVersion(ddpPath, dev)
	if err != nil {
		log.Error(err, "Failed to retrieve DDP version")
		return err
	}

	driverVer := utils.GetDriverVersion(dev)
	deviceIDs, err := getIDs(dev.PCIAddress, log)
	if err != nil {
		log.Error(err, "Failed to retrieve device IDs")
		return err
	}

	for _, c := range *compatibilityMap {
		if deviceMatcher(deviceIDs,
			fwVer,
			ddpVer,
			driverVer,
			c) {
			log.V(2).Info("Matching compatibility entry found", "entry", c)
			return nil
		}
	}

	return fmt.Errorf("no matching compatibility entry for %v (Ven:%v Cl:%v SubCl:%v DevID:%v Drv:%v FW:%v DDP:%v)",
		dev.PCIAddress, deviceIDs.VendorID, deviceIDs.Class, deviceIDs.SubClass, deviceIDs.DeviceID, driverVer,
		fwVer, ddpVer)
}

type CompatibilityMap map[string]Compatibility
type Compatibility struct {
	utils.SupportedDevice
	Driver   string
	Firmware string
	DDP      []string
}

var deviceMatcher = func(ids DeviceIDs, fwVer, ddpVer, driverVer string, entry Compatibility) bool {
	if ids.VendorID == entry.VendorID &&
		ids.Class == entry.Class &&
		ids.SubClass == entry.SubClass &&
		ids.DeviceID == entry.DeviceID &&
		(entry.Firmware == compatibilityWildard || fwVer == entry.Firmware) &&
		(entry.Driver == compatibilityWildard || driverVer == entry.Driver) &&
		ddpVersionMatcher(ddpVer, entry.DDP) {
		return true
	}
	return false
}

var ddpVersionMatcher = func(ddpVer string, ddp []string) bool {
	if len(ddp) == 1 && ddp[0] == compatibilityWildard {
		return true
	}

	for _, d := range ddp {
		if ddpVer == d {
			return true
		}
	}
	return false
}
