// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package daemon

import (
	"fmt"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/net"
	"github.com/jaypipes/pcidb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/otcshare/intel-ethernet-operator/pkg/utils"
	ctrl "sigs.k8s.io/controller-runtime"
)

var pcis = func() ([]*ghw.PCIDevice, error) {
	return []*ghw.PCIDevice{
		{
			Address: "00:00:00.0",
			Vendor: &pcidb.Vendor{
				ID: "0000",
			},
			Class: &pcidb.Class{
				ID: "00",
			},
			Subclass: &pcidb.Subclass{
				ID: "00",
			},
			Product: &pcidb.Product{
				ID:   "test",
				Name: "testname",
			},
		},
	}, nil
}

var _ = Describe("InventoryTest", func() {
	log := ctrl.Log.WithName("FirmwareDaemon-test")
	var _ = Context("GetInventory", func() {
		var _ = It("will return error when is not able to get PCI devices", func() {
			getPCIDevices = func() ([]*ghw.PCIDevice, error) {
				return nil, fmt.Errorf("Failed to get PCI")
			}
			d, err := GetInventory(log)
			Expect(err).To(HaveOccurred())
			Expect(d).To(BeEmpty())
		})

		var _ = It("will return empty []ethernetv1.Device if supported device was not found", func() {
			getPCIDevices = pcis
			supportedDevices = utils.SupportedDevices{
				"dev1": utils.SupportedDevice{
					VendorID: "0001",
					Class:    "00",
					SubClass: "00",
					DeviceID: "test",
				},
			}
			d, err := GetInventory(log)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(BeEmpty())

			supportedDevices = utils.SupportedDevices{
				"dev1": utils.SupportedDevice{
					VendorID: "0000",
					Class:    "01",
					SubClass: "00",
					DeviceID: "test",
				},
			}
			d, err = GetInventory(log)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(BeEmpty())

			supportedDevices = utils.SupportedDevices{
				"dev1": utils.SupportedDevice{
					VendorID: "0000",
					Class:    "00",
					SubClass: "01",
					DeviceID: "test",
				},
			}
			d, err = GetInventory(log)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(BeEmpty())

			supportedDevices = utils.SupportedDevices{
				"dev1": utils.SupportedDevice{
					VendorID: "0000",
					Class:    "00",
					SubClass: "00",
					DeviceID: "test2",
				},
			}
			d, err = GetInventory(log)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(BeEmpty())
		})

		var _ = It("will return []ethernetv1.Device partial inventory if net info was not available", func() {
			getPCIDevices = pcis
			getNetworkInfo = func() (*net.Info, error) {
				return nil, fmt.Errorf("failed to get network info")
			}

			supportedDevices = utils.SupportedDevices{
				"dev1": utils.SupportedDevice{
					VendorID: "0000",
					Class:    "00",
					SubClass: "00",
					DeviceID: "test",
				},
			}
			d, err := GetInventory(log)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).ToNot(BeEmpty())
			Expect(d[0].PCIAddress).To(Equal("00:00:00.0"))
			Expect(d[0].Name).To(Equal("testname"))
			Expect(d[0].Driver).To(BeEmpty())
			Expect(d[0].DriverVersion).To(BeEmpty())
			Expect(d[0].Firmware.MAC).To(BeEmpty())
			Expect(d[0].Firmware.Version).To(BeEmpty())
		})

		var _ = It("will return []ethernetv1.Device partial inventory if ethtool was not available", func() {
			getPCIDevices = pcis
			pciAddr := "00:00:00.0"
			getNetworkInfo = func() (*net.Info, error) {
				return &net.Info{
					NICs: []*net.NIC{
						{
							PCIAddress: &pciAddr,
							Name:       "eno0",
							MacAddress: "aa:bb:cc:dd:ee:ff",
						},
					},
				}, nil
			}
			execEthtool = func(string) ([]byte, error) {
				return nil, fmt.Errorf("Error when calling ethtool")
			}

			supportedDevices = utils.SupportedDevices{
				"dev1": utils.SupportedDevice{
					VendorID: "0000",
					Class:    "00",
					SubClass: "00",
					DeviceID: "test",
				},
			}
			d, err := GetInventory(log)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).ToNot(BeEmpty())
			Expect(d[0].PCIAddress).To(Equal("00:00:00.0"))
			Expect(d[0].Name).To(Equal("testname"))
			Expect(d[0].Driver).To(BeEmpty())
			Expect(d[0].DriverVersion).To(BeEmpty())
			Expect(d[0].Firmware.MAC).To(Equal("aa:bb:cc:dd:ee:ff"))
			Expect(d[0].Firmware.Version).To(BeEmpty())
		})

		var _ = It("will return inventory", func() {
			getPCIDevices = pcis
			pciAddr := "00:00:00.0"
			getNetworkInfo = func() (*net.Info, error) {
				return &net.Info{
					NICs: []*net.NIC{
						{
							PCIAddress: &pciAddr,
							Name:       "eno0",
							MacAddress: "aa:bb:cc:dd:ee:ff",
						},
					},
				}, nil
			}

			execEthtool = func(string) ([]byte, error) {
				return []byte(
					`driver: i40e
version: 2.8.20-k
firmware-version: 3.31 0x80000d31 1.1767.0
expansion-rom-version: 
bus-info: 0000:00:00.0
supports-statistics: yes
supports-test: yes
supports-eeprom-access: yes
supports-register-dump: yes
supports-priv-flags: yes
`), nil
			}

			supportedDevices = utils.SupportedDevices{
				"dev1": utils.SupportedDevice{
					VendorID: "0000",
					Class:    "00",
					SubClass: "00",
					DeviceID: "test",
				},
			}
			d, err := GetInventory(log)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).ToNot(BeEmpty())
			Expect(d[0].PCIAddress).To(Equal("00:00:00.0"))
			Expect(d[0].Name).To(Equal("testname"))
			Expect(d[0].Driver).To(Equal("i40e"))
			Expect(d[0].DriverVersion).To(Equal("2.8.20-k"))
			Expect(d[0].Firmware.MAC).To(Equal("aa:bb:cc:dd:ee:ff"))
			Expect(d[0].Firmware.Version).To(Equal("3.31 0x80000d31 1.1767.0"))
		})
	})
})
