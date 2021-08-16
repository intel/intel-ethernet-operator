// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package main

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/otcshare/intel-ethernet-operator/pkg/utils"

	"github.com/jaypipes/ghw"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	deviceConfig = "./devices.json"
)

var getInclusterConfigFunc = rest.InClusterConfig

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

func isDeviceSupported(dev *ghw.PCIDevice, supportedList *utils.SupportedDevices) bool {
	for name, supported := range *supportedList {
		if dev.Vendor.ID != supported.VendorID {
			continue
		} else if dev.Class.ID != supported.Class {
			continue
		} else if dev.Subclass.ID != supported.SubClass {
			continue
		} else if dev.Product.ID != supported.DeviceID {
			continue
		}

		fmt.Printf("FOUND %v at %v: Vendor=%v Class=%v:%v Device=%v\n", name,
			dev.Address, dev.Vendor.ID, dev.Class.ID,
			dev.Subclass.ID, dev.Product.ID)
		return true
	}

	return false
}

func findSupportedDevice(supportedList *utils.SupportedDevices) (bool, error) {
	if supportedList == nil {
		return false, fmt.Errorf("config not provided")
	}

	present, err := getPCIDevices()
	if err != nil {
		return false, fmt.Errorf("Failed to get PCI devices: %v", err)
	}

	for _, dev := range present {
		if isDeviceSupported(dev, supportedList) {
			return true, nil
		}
	}

	return false, nil
}

func setNodeLabel(nodeName, label string, isDevicePresent bool) error {
	if label == "" {
		return fmt.Errorf("label is empty (check the NODELABEL env var)")
	}
	if nodeName == "" {
		return fmt.Errorf("nodeName is empty (check the NODENAME env var)")
	}

	cfg, err := getInclusterConfigFunc()
	if err != nil {
		return fmt.Errorf("Failed to get cluster config: %v\n", err.Error())
	}
	cli, err := clientset.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("Failed to initialize clientset: %v\n", err.Error())
	}

	node, err := cli.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Failed to get the node object: %v\n", err)
	}
	nodeLabels := node.GetLabels()
	if isDevicePresent {
		nodeLabels[label] = ""
	} else {
		delete(nodeLabels, label)
	}
	node.SetLabels(nodeLabels)
	_, err = cli.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("Failed to update the node object: %v\n", err)
	}

	return nil
}

func deviceDiscovery(deviceConfig string) error {
	supportedList, err := utils.LoadSupportedDevices(deviceConfig)
	if err != nil {
		return fmt.Errorf("Failed to load devices: %v", err)
	}
	if len(supportedList) == 0 {
		return fmt.Errorf("No devices configured")
	}

	devFound, err := findSupportedDevice(&supportedList)
	if err != nil {
		return fmt.Errorf("Failed to find device: %v", err)
	}

	return setNodeLabel(os.Getenv("NODENAME"), os.Getenv("NODELABEL"), devFound)
}

func main() {
	err := deviceDiscovery(deviceConfig)
	if err != nil {
		fmt.Printf("Device discovery failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Device discovery finished successfully\n")

	os.Exit(0)
}
