// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package daemon

import (
	"fmt"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/net"
	"github.com/jaypipes/pcidb"
	. "github.com/onsi/ginkgo/v2"
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
		nil,
	}, nil
}

var _ = Describe("InventoryTest", func() {
	log := ctrl.Log.WithName("FirmwareDaemon-test")
	var _ = Context("GetInventory", func() {
		var _ = It("will return error when is not able to get PCI devices", func() {
			getPCIDevices = func() ([]*ghw.PCIDevice, error) {
				return nil, fmt.Errorf("failed to get PCI")
			}
			d, err := GetInventory(log)
			Expect(err).To(HaveOccurred())
			Expect(d).To(BeEmpty())
		})

		var _ = It("will return empty []ethernetv1.Device if supported device was not found", func() {
			getPCIDevices = pcis
			compatibilityMap = &CompatibilityMap{
				"dev1": Compatibility{
					SupportedDevice: utils.SupportedDevice{
						VendorID: "0001",
						Class:    "00",
						SubClass: "00",
						DeviceID: "test",
					},
				},
			}
			d, err := GetInventory(log)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(BeEmpty())
			compatibilityMap = &CompatibilityMap{
				"dev1": Compatibility{
					SupportedDevice: utils.SupportedDevice{
						VendorID: "0000",
						Class:    "01",
						SubClass: "00",
						DeviceID: "test",
					},
				},
			}

			d, err = GetInventory(log)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(BeEmpty())

			compatibilityMap = &CompatibilityMap{
				"dev1": Compatibility{
					SupportedDevice: utils.SupportedDevice{
						VendorID: "0000",
						Class:    "00",
						SubClass: "01",
						DeviceID: "test",
					},
				},
			}

			d, err = GetInventory(log)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(BeEmpty())

			compatibilityMap = &CompatibilityMap{
				"dev1": Compatibility{
					SupportedDevice: utils.SupportedDevice{
						VendorID: "0000",
						Class:    "00",
						SubClass: "00",
						DeviceID: "test2",
					},
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

			compatibilityMap = &CompatibilityMap{
				"dev1": Compatibility{
					SupportedDevice: utils.SupportedDevice{
						VendorID: "0000",
						Class:    "00",
						SubClass: "00",
						DeviceID: "test",
					},
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
			Expect(d[0].DDP.PackageName).To(BeEmpty())
			Expect(d[0].DDP.Version).To(BeEmpty())
			Expect(d[0].DDP.TrackID).To(BeEmpty())
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
				return nil, fmt.Errorf("error when calling ethtool")
			}
			execDevlink = func(string) ([]byte, error) {
				return nil, fmt.Errorf("error when calling devlink")
			}

			compatibilityMap = &CompatibilityMap{
				"dev1": Compatibility{
					SupportedDevice: utils.SupportedDevice{
						VendorID: "0000",
						Class:    "00",
						SubClass: "00",
						DeviceID: "test",
					},
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
			Expect(d[0].DDP.PackageName).To(BeEmpty())
			Expect(d[0].DDP.Version).To(BeEmpty())
			Expect(d[0].DDP.TrackID).To(BeEmpty())
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

			execDevlink = func(string) ([]byte, error) {
				return []byte(`
pci/0000:51:00.0:
  driver ice
  serial_number b4-96-91-ff-ff-af-6d-68
  versions:
      fixed:
        board.id M17659-003
      running:
        fw.mgmt 5.4.5
        fw.mgmt.api 1.7
        fw.mgmt.build 0x391f7640
        fw.undi 1.2898.0
        fw.psid.api 2.40
        fw.bundle_id 0x80007064
        fw.app.name ICE OS Default Package
        fw.app 1.3.4.0
        fw.app.bundle_id 0x00000000
        fw.netlist 2.40.2000-6.22.0
        fw.netlist.build 0x0ee8f468
      stored:
        fw.undi 1.2898.0
        fw.psid.api 2.40
        fw.bundle_id 0x80007064
        fw.netlist 2.40.2000-6.22.0
        fw.netlist.build 0x0ee8f468
`), nil
			}

			compatibilityMap = &CompatibilityMap{
				"dev1": Compatibility{
					SupportedDevice: utils.SupportedDevice{
						VendorID: "0000",
						Class:    "00",
						SubClass: "00",
						DeviceID: "test",
					},
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
			Expect(d[0].DDP.PackageName).To(Equal("ICE OS Default Package"))
			Expect(d[0].DDP.Version).To(Equal("1.3.4.0"))
			Expect(d[0].DDP.TrackID).To(Equal("0x00000000"))
		})
	})
})
