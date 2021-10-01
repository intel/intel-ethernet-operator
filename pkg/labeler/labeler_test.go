// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package labeler

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes/scheme"
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
			Expect(err).To(MatchError(ContainSubstring("not provided")))
			Expect(found).To(Equal(false))
		})

		var _ = It("will fail if getPCIDevices fails", func() {
			getPCIDevices = func() ([]*ghw.PCIDevice, error) { return nil, fmt.Errorf("ErrorStub") }

			supportedDevices := new(utils.SupportedDevices)
			Expect(utils.LoadSupportedDevices("testdata/devices.json", supportedDevices)).ToNot(HaveOccurred())

			found, err := findSupportedDevice(supportedDevices)
			Expect(err).To(HaveOccurred())
			Expect(found).To(Equal(false))
		})

		var _ = It("will return false if there is no devices found", func() {
			getPCIDevices = func() ([]*ghw.PCIDevice, error) {
				return []*ghw.PCIDevice{}, nil
			}

			supportedDevices := new(utils.SupportedDevices)
			Expect(utils.LoadSupportedDevices("testdata/devices.json", supportedDevices)).ToNot(HaveOccurred())

			found, err := findSupportedDevice(supportedDevices)
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

			supportedDevices := new(utils.SupportedDevices)
			Expect(utils.LoadSupportedDevices("testdata/devices.json", supportedDevices)).ToNot(HaveOccurred())

			found, err := findSupportedDevice(supportedDevices)
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
			Expect(k8sClient.Create(context.TODO(), node)).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			// Remove nodes
			nodes := &corev1.NodeList{}
			Expect(k8sClient.List(context.TODO(), nodes)).ToNot(HaveOccurred())
			for _, nodeToDelete := range nodes.Items {
				Expect(k8sClient.Delete(context.TODO(), &nodeToDelete)).ToNot(HaveOccurred())
			}
		})
		var _ = It("will fail if there is no cluster", func() {
			fakeGetInclusterConfigReturn = fmt.Errorf("cannot get InClusterConfig")
			Expect(setNodeLabel("anyName", "anyLabel", false)).To(MatchError(ContainSubstring("cannot get InClusterConfig")))
		})
		var _ = It("will fail when empty label", func() {
			Expect(setNodeLabel("nodename", "", false)).To(MatchError(ContainSubstring("label is empty")))
		})
		var _ = It("will pass if there is cluster", func() {
			Expect(setNodeLabel("nodename", "testlabel", false)).ToNot(HaveOccurred())
		})
	})
	var _ = Describe("DeviceDiscovery", func() {
		BeforeEach(func() {
			fakeGetInclusterConfigReturn = nil
			getInclusterConfigFunc = fakeGetInclusterConfig
		})
		var _ = It("will fail if load config fails", func() {
			deviceConfig = "testdata/not-existing.json"
			Expect(DeviceDiscovery()).To(MatchError(ContainSubstring("failed to load")))
		})
		var _ = It("will fail if findDevice fails", func() {
			getPCIDevices = func() ([]*ghw.PCIDevice, error) { return nil, fmt.Errorf("getPCIDevices error") }

			deviceConfig = "testdata/devices.json"
			Expect(DeviceDiscovery()).To(MatchError(ContainSubstring("getPCIDevices error")))
		})
		var _ = It("will fail if there is no NODENAME env", func() {
			_ = os.Unsetenv("NODENAME")
			Expect(os.Setenv("NODELABEL", "anyNodeLabelValue")).ToNot(HaveOccurred())
			getPCIDevices = func() ([]*ghw.PCIDevice, error) { return []*ghw.PCIDevice{}, nil }
			deviceConfig = "testdata/devices.json"
			Expect(DeviceDiscovery()).To(MatchError(ContainSubstring("nodeName is empty ")))
		})
		var _ = It("will fail if there is no k8s cluster", func() {
			fakeGetInclusterConfigReturn = fmt.Errorf("error")
			getPCIDevices = func() ([]*ghw.PCIDevice, error) { return []*ghw.PCIDevice{}, nil }
			deviceConfig = "testdata/devices.json"
			os.Setenv("NODENAME", "nodeName")
			os.Setenv("NODELABEL", "nodeLabelValue")
			Expect(DeviceDiscovery()).To(MatchError(ContainSubstring("Failed to get cluster config")))
		})
	})
})
