// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

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
)

const (
	ethtoolPath = "ethtool"
)

var (
	ethtoolRegex = regexp.MustCompile(`^([a-z-]+?)(?:\s*:\s)(.+)$`)
)

var getPCIDevices = func() ([]*ghw.PCIDevice, error) {
	pci, err := ghw.PCI()
	if err != nil {
		return nil, fmt.Errorf("Failed to get PCI info: %v", err)
	}

	devices := pci.ListDevices()
	if len(devices) == 0 {
		return nil, fmt.Errorf("Got 0 devices")
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

func isDeviceSupported(d *pci.Device) bool {
	if d == nil {
		return false
	}

	for _, supported := range supportedDevices {
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

	devices := []ethernetv1.Device{}

	for _, pciDevice := range pciDevices {
		if isDeviceSupported(pciDevice) {
			d := ethernetv1.Device{
				PCIAddress: pciDevice.Address,
				Name:       pciDevice.Product.Name,
			}
			d = addNetInfo(log, d)
			devices = append(devices, d)
		}
	}

	return devices, nil
}

func addNetInfo(log logr.Logger, device ethernetv1.Device) ethernetv1.Device {
	net, err := getNetworkInfo()
	if err != nil {
		log.Error(err, "failed to get network interfaces")
		return device
	}

	nicName := ""
	for _, nic := range net.NICs {
		if nic.PCIAddress != nil && *nic.PCIAddress == device.PCIAddress {
			device.Firmware.MAC = nic.MacAddress
			nicName = nic.Name
			break
		}
	}

	if nicName != "" {
		out, err := execEthtool(nicName)
		if err != nil {
			log.Error(err, "failed when executing", "cmd", ethtoolPath)
			return device
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
	return device
}

func getDeviceMAC(pciAddr string, log logr.InfoLogger) (string, error) {
	inv, err := getInventory(log)
	if err != nil {
		log.Error(err, "Failed to retrieve inventory")
		return "", err
	}

	for _, i := range inv {
		if i.PCIAddress == pciAddr {
			return strings.Replace(strings.ToUpper(i.Firmware.MAC), ":", "", -1), nil
		}
	}
	return "", fmt.Errorf("Device %v not found", pciAddr)
}
