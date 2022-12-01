// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

package daemon

import (
	"fmt"
	"os/exec"
	"strings"

	"regexp"

	"github.com/go-logr/logr"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/net"
	"github.com/jaypipes/ghw/pkg/pci"
	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"
	"github.com/otcshare/intel-ethernet-operator/pkg/utils"
)

const (
	ethtoolPath = "ethtool"
)

var (
	ethtoolRegex = regexp.MustCompile(`^([a-z-]+?)(?:\s*:\s)(.+)$`)
	devlinkRegex = regexp.MustCompile(`^\s+([\w\.]+) (.+)$`)
)

var getPCIDevices = func() ([]*ghw.PCIDevice, error) {
	pci, err := ghw.PCI()
	if err != nil {
		return nil, fmt.Errorf("failed to get PCI info: %v", err)
	}

	devices := pci.ListDevices()
	if len(devices) == 0 {
		return nil, fmt.Errorf("got 0 devices")
	}
	return devices, nil
}

var getNetworkInfo = func() (*net.Info, error) {
	net, err := ghw.Network()
	if err != nil {
		return nil, fmt.Errorf("failed to get network info: %v", err)
	}
	return net, nil
}

var execEthtool = func(nicName string) ([]byte, error) {
	return exec.Command(ethtoolPath, "-i", nicName).Output()
}

var execDevlink = func(pciAddr string) ([]byte, error) {
	devName := fmt.Sprintf("pci/%s", pciAddr)

	return exec.Command("devlink", "dev", "info", devName).CombinedOutput()
}

func isDeviceSupported(d *pci.Device) bool {
	if d == nil {
		return false
	}

	for _, supported := range *compatibilityMap {
		if supported.VendorID == d.Vendor.ID &&
			supported.Class == d.Class.ID &&
			supported.SubClass == d.Subclass.ID &&
			supported.DeviceID == d.Product.ID {
			return true
		}
	}
	return false
}

func GetInventory(log logr.Logger) ([]ethernetv1.Device, error) {
	pciDevices, err := getPCIDevices()
	if err != nil {
		return nil, err
	}

	var devices []ethernetv1.Device

	for _, pciDevice := range pciDevices {
		if isDeviceSupported(pciDevice) {
			d := ethernetv1.Device{
				PCIAddress: pciDevice.Address,
				Name:       pciDevice.Product.Name,
				VendorID:   pciDevice.Vendor.ID,
				DeviceID:   pciDevice.Product.ID,
			}
			addNetInfo(log, &d)
			addDDPInfo(log, &d)
			devices = append(devices, d)
		}
	}

	return devices, nil
}

func addNetInfo(log logr.Logger, device *ethernetv1.Device) {
	log.Info("adding netInfo for supported device", "device", device)

	net, err := getNetworkInfo()
	if err != nil {
		log.Error(err, "failed to get network interfaces")
		return
	}

	nicName := ""
	for _, nic := range net.NICs {
		if nic.PCIAddress != nil && *nic.PCIAddress == device.PCIAddress {
			device.Firmware.MAC = nic.MacAddress
			nicName = nic.Name
			break
		}
	}
	if nicName == "" {
		log.Info("failed to find nicName for device", "pciAddress", device.PCIAddress)
		return // NIC not found
	}

	out, err := execEthtool(nicName)
	if err != nil {
		log.Error(err, "failed when executing", "cmd", ethtoolPath)
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		m := ethtoolRegex.FindStringSubmatch(line)
		if len(m) == 3 {
			switch m[1] {
			case "driver":
				device.Driver = m[2]
			case "version":
				device.DriverVersion = m[2]
			case "firmware-version":
				device.Firmware.Version = m[2]
			}
		}
	}

}

func addDDPInfo(log logr.Logger, device *ethernetv1.Device) {
	out, err := execDevlink(device.PCIAddress)
	if err != nil {
		log.Error(err, "failed when executing devlink", "out", string(out))
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		tokens := devlinkRegex.FindStringSubmatch(line)
		if len(tokens) != 3 {
			continue
		}

		switch tokens[1] {
		case "fw.app.name":
			device.DDP.PackageName = tokens[2]
		case "fw.app":
			device.DDP.Version = tokens[2]
		case "fw.app.bundle_id":
			device.DDP.TrackID = tokens[2]
		}
	}
}

func splitPCIAddr(pciAddr string, log logr.Logger) (string, string, string, string, error) {
	pciAddrList := strings.Split(pciAddr, ":")
	if len(pciAddrList) != 3 {
		return "", "", "", "", fmt.Errorf("PCI Address %v format issue, cannot split on colon :", pciAddr)
	}
	busDevice := strings.Split(pciAddrList[2], ".")
	if len(busDevice) != 2 {
		return "", "", "", "", fmt.Errorf("Bus and device %v format issue, cannot split on dot .", pciAddrList[2])
	}

	return pciAddrList[0], pciAddrList[1], busDevice[0], busDevice[1], nil
}

type DeviceIDs utils.SupportedDevice
