// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/otcshare/intel-ethernet-operator/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	node                         *corev1.Node
	config                       *rest.Config
	k8sClient                    client.Client
	testEnv                      *envtest.Environment
	fakeGetInclusterConfigReturn error = nil
)

func fakeGetInclusterConfig() (*rest.Config, error) {
	return config, fakeGetInclusterConfigReturn
}

var _ = BeforeSuite(func(done Done) {
	var err error
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	config, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(config).ToNot(BeNil())

	k8sClient, err = client.New(config, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func TestLabeler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main suite")
}

var _ = Describe("Labeler", func() {
	var _ = Describe("getPCIDevices", func() {
		var _ = It("return PCI devices", func() {
			devices, err := getPCIDevices()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(devices)).ToNot(Equal(0))
		})
	})
	var _ = Describe("findSupportedDevice", func() {
		var _ = It("will fail if config is not provided", func() {
			found, err := findSupportedDevice(nil)
			Expect(err).To(HaveOccurred())
			Expect(found).To(Equal(false))
		})

		var _ = It("will fail if getPCIDevices fails", func() {
			getPCIDevices = func() ([]*ghw.PCIDevice, error) { return nil, fmt.Errorf("ErrorStub") }

			cfg, err := utils.LoadSupportedDevices("testdata/devices.json")
			Expect(err).ToNot(HaveOccurred())

			found, err := findSupportedDevice(&cfg)
			Expect(err).To(HaveOccurred())
			Expect(found).To(Equal(false))
		})

		var _ = It("will return false if there is no devices found", func() {
			getPCIDevices = func() ([]*ghw.PCIDevice, error) {
				return []*ghw.PCIDevice{}, nil
			}

			cfg, err := utils.LoadSupportedDevices("testdata/devices.json")
			Expect(err).ToNot(HaveOccurred())

			found, err := findSupportedDevice(&cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(Equal(false))
		})

		var _ = It("will return true if there is a device found", func() {
			getPCIDevices = func() ([]*ghw.PCIDevice, error) {
				var devices []*ghw.PCIDevice
				devices = append(devices,
					&ghw.PCIDevice{
						Vendor: &pcidb.Vendor{
							ID: "0000",
						},
						Class: &pcidb.Class{
							ID: "00",
						},
						Subclass: &pcidb.Subclass{
							ID: "02",
						},
						Product: &pcidb.Product{
							ID: "test",
						},
					},
					&ghw.PCIDevice{
						Vendor: &pcidb.Vendor{
							ID: "0001",
						},
						Class: &pcidb.Class{
							ID: "00",
						},
						Subclass: &pcidb.Subclass{
							ID: "00",
						},
						Product: &pcidb.Product{
							ID: "123",
						},
					},
				)
				return devices, nil
			}

			cfg, err := utils.LoadSupportedDevices("testdata/devices.json")
			Expect(err).ToNot(HaveOccurred())

			found, err := findSupportedDevice(&cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(Equal(true))
		})
	})
	var _ = Describe("setNodeLabel", func() {
		BeforeEach(func() {
			fakeGetInclusterConfigReturn = nil
			getInclusterConfigFunc = fakeGetInclusterConfig
			node = &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "nodename",
					Labels: map[string]string{
						"fpga.intel.com/intel-device-present": "",
					},
				},
			}
			err := k8sClient.Create(context.TODO(), node)
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			var err error
			// Remove nodes
			nodes := &corev1.NodeList{}
			err = k8sClient.List(context.TODO(), nodes)
			Expect(err).ToNot(HaveOccurred())

			for _, nodeToDelete := range nodes.Items {
				err = k8sClient.Delete(context.TODO(), &nodeToDelete)
				Expect(err).ToNot(HaveOccurred())
			}
		})
		var _ = It("will fail if there is no cluster", func() {
			fakeGetInclusterConfigReturn = fmt.Errorf("error")
			err := setNodeLabel("", "", false)
			Expect(err).To(HaveOccurred())
		})
		var _ = It("will fail when empty label", func() {
			err := setNodeLabel("nodename", "", false)
			Expect(err).To(HaveOccurred())
		})
		var _ = It("will pass if there is cluster", func() {
			err := setNodeLabel("nodename", "testlabel", false)
			Expect(err).ToNot(HaveOccurred())
		})
	})
	var _ = Describe("deviceDiscovery", func() {
		BeforeEach(func() {
			fakeGetInclusterConfigReturn = nil
			getInclusterConfigFunc = fakeGetInclusterConfig
		})
		var _ = It("will fail if load config fails", func() {
			err := deviceDiscovery("")
			Expect(err).To(HaveOccurred())
		})
		var _ = It("will fail if findDevice fails", func() {
			getPCIDevices = func() ([]*ghw.PCIDevice, error) { return nil, fmt.Errorf("ErrorStub") }
			err := deviceDiscovery("testdata/valid.json")
			Expect(err).To(HaveOccurred())
		})
		var _ = It("will fail if there is no NODENAME env", func() {
			getPCIDevices = func() ([]*ghw.PCIDevice, error) { return []*ghw.PCIDevice{}, nil }
			err := deviceDiscovery("testdata/valid.json")
			Expect(err).To(HaveOccurred())
		})
		var _ = It("will fail if there is no k8s cluster", func() {
			fakeGetInclusterConfigReturn = fmt.Errorf("error")
			getPCIDevices = func() ([]*ghw.PCIDevice, error) { return []*ghw.PCIDevice{}, nil }
			os.Setenv("NODENAME", "test")
			err := deviceDiscovery("testdata/valid.json")
			os.Unsetenv("NODENAME")
			Expect(err).To(HaveOccurred())
		})
	})
})
