// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package fwddp_manager

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DEFAULT_CLUSTER_CONFIG_NAME = "config"
)

var (
	nodeConfigPrototype = &ethernetv1.EthernetNodeConfig{
		ObjectMeta: v1.ObjectMeta{
			Namespace: NAMESPACE,
		},
		Spec: ethernetv1.EthernetNodeConfigSpec{
			Config: []ethernetv1.DeviceNodeConfig{},
		},
		Status: ethernetv1.EthernetNodeConfigStatus{
			Devices: []ethernetv1.Device{},
		},
	}

	clusterConfigPrototype = &ethernetv1.EthernetClusterConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      DEFAULT_CLUSTER_CONFIG_NAME,
			Namespace: NAMESPACE,
		},
		Spec: ethernetv1.EthernetClusterConfigSpec{
			NodeSelector: map[string]string{},
			DeviceConfig: ethernetv1.DeviceConfig{
				DDPURL: "192.168.1.1:5000/testDDP",
				FWURL:  "192.168.1.1:5000/testFW",
			},
		},
	}

	nodePrototype = &corev1.Node{
		ObjectMeta: v1.ObjectMeta{
			Name: "node-dummy",
			Labels: map[string]string{
				"ethernet.intel.com/intel-ethernet-present": "",
			},
		},
	}
)

var _ = Describe("EthernetControllerTest", func() {
	var _ = Describe("Reconciler", func() {
		var log = ctrl.Log.WithName("EthernetController-test")
		createNodeInventory := func(nodeName string, inventory []ethernetv1.Device) {
			nodeConfig := nodeConfigPrototype.DeepCopy()
			nodeConfig.Name = nodeName
			nodeConfig.Status.Devices = inventory
			Expect(k8sClient.Create(context.TODO(), nodeConfig)).ToNot(HaveOccurred())
			Expect(k8sClient.Status().Update(context.TODO(), nodeConfig)).ToNot(HaveOccurred())
		}

		createNode := func(name string, configurers ...func(n *corev1.Node)) *corev1.Node {
			node := nodePrototype.DeepCopy()
			node.Name = name
			for _, configure := range configurers {
				configure(node)
			}
			Expect(k8sClient.Create(context.TODO(), node)).ToNot(HaveOccurred())
			return node
		}

		createDeviceConfig := func(configName string, configurers ...func(cc *ethernetv1.EthernetClusterConfig)) *ethernetv1.EthernetClusterConfig {
			cc := clusterConfigPrototype.DeepCopy()
			cc.Name = configName
			for _, configure := range configurers {
				configure(cc)
			}
			Expect(k8sClient.Create(context.TODO(), cc)).ToNot(HaveOccurred())
			return cc
		}

		createDummyReconcileRequest := func() ctrl.Request {
			return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "", Name: ""}}
		}

		reconcile := func() *EthernetClusterConfigReconciler {
			reconciler := EthernetClusterConfigReconciler{k8sClient, log, scheme.Scheme}
			_, err := reconciler.Reconcile(context.TODO(), createDummyReconcileRequest())
			Expect(err).ToNot(HaveOccurred())
			return &reconciler
		}

		AfterEach(func() {
			ccl := new(ethernetv1.EthernetClusterConfigList)
			Expect(k8sClient.List(context.TODO(), ccl)).ToNot(HaveOccurred())
			for _, item := range ccl.Items {
				Expect(k8sClient.Delete(context.TODO(), &item)).ToNot(HaveOccurred())
			}

			ncl := new(ethernetv1.EthernetNodeConfigList)
			Expect(k8sClient.List(context.TODO(), ncl)).ToNot(HaveOccurred())
			for _, item := range ncl.Items {
				Expect(k8sClient.Delete(context.TODO(), &item)).ToNot(HaveOccurred())
			}

			Expect(k8sClient.DeleteAllOf(context.TODO(), &corev1.Node{})).ToNot(HaveOccurred())
		})

		When("cc does not match to any node", func() {
			It("node config should not be propagated", func() {
				n1 := createNode("n1")
				n2 := createNode("n2")

				createNodeInventory(n1.Name, []ethernetv1.Device{
					{
						VendorID: "vendor",
						DeviceID: "abc",
					},
				})

				createNodeInventory(n2.Name, []ethernetv1.Device{
					{
						VendorID: "vendor",
						DeviceID: "cba",
					},
				})

				createDeviceConfig("cc", func(cc *ethernetv1.EthernetClusterConfig) {
					cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
						VendorID: "notExistingVendor",
					}
					cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
						FWURL: "testurl",
					}
				})

				reconcile()

				nc := new(ethernetv1.EthernetNodeConfig)
				Expect(k8sClient.Get(context.TODO(), client.ObjectKey{Name: n1.Name, Namespace: NAMESPACE}, nc)).ToNot(HaveOccurred())
				Expect(nc.Spec.Config).To(BeEmpty())

				nc = new(ethernetv1.EthernetNodeConfig)
				Expect(k8sClient.Get(context.TODO(), client.ObjectKey{Name: n2.Name, Namespace: NAMESPACE}, nc)).ToNot(HaveOccurred())
				Expect(nc.Spec.Config).To(BeEmpty())

			})
		})

		When("two ccs does match to single accelerator on single node", func() {
			It("cc.spec with higher priority should be propagated to matching nc", func() {

				const (
					lowPriority  = 1
					highPriority = 100
				)

				n1 := createNode("n1")

				createNodeInventory(n1.Name, []ethernetv1.Device{
					{
						PCIAddress: "0000:15:00.1",
						VendorID:   "testvendor",
						DeviceID:   "testid",
					},
				})

				hpcc := createDeviceConfig("high-priority-cluster-config", func(cc *ethernetv1.EthernetClusterConfig) {
					cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
						VendorID: "testvendor",
					}
					cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
						DDPURL: "testdpdurl",
						FWURL:  "testfwurl",
					}
					cc.Spec.Priority = highPriority
				})

				_ = createDeviceConfig("low-priority-cluster-config", func(cc *ethernetv1.EthernetClusterConfig) {
					cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
						VendorID: "testvendor",
					}
					cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
						DDPURL: "testdpdurl_low",
						FWURL:  "testfwurl_low",
					}
					cc.Spec.Priority = lowPriority
				})

				_ = reconcile()

				cl := new(ethernetv1.EthernetNodeConfigList)
				Expect(k8sClient.List(context.TODO(), cl)).ToNot(HaveOccurred())
				Expect(cl.Items).To(HaveLen(1))
				nc := cl.Items[0]
				Expect(nc.Spec.Config).To(HaveLen(1))
				Expect(nc.Spec.Config[0].DeviceConfig.DDPURL).Should(Equal(hpcc.Spec.DeviceConfig.DDPURL))
				Expect(nc.Spec.Config[0].DeviceConfig.FWURL).Should(Equal(hpcc.Spec.DeviceConfig.FWURL))

			})

			Context("both of them have same priority", func() {
				It("only newer cc.spec should be propagated to matching nc", func() {

					n1 := createNode("n1")

					createNodeInventory(n1.Name, []ethernetv1.Device{
						{
							PCIAddress: "0000:15:00.1",
							VendorID:   "testvendor",
						},
					})

					createDeviceConfig("older-cluster-config", func(cc *ethernetv1.EthernetClusterConfig) {
						cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
							VendorID: "testvendor",
						}
						cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
							DDPURL: "testddpurl_old",
							FWURL:  "testfwurl_old",
						}
						cc.Spec.Priority = 1
					})

					//put some delay between one and another config creation
					time.Sleep(time.Nanosecond)
					newerCC := createDeviceConfig("newer-cluster-config", func(cc *ethernetv1.EthernetClusterConfig) {
						cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
							PCIAddress: "0000:15:00.1",
						}
						cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
							DDPURL: "testddpurl_new",
							FWURL:  "testfwurl_new",
						}
						cc.Spec.Priority = 1
					})

					_ = reconcile()

					cl := new(ethernetv1.EthernetNodeConfigList)
					Expect(k8sClient.List(context.TODO(), cl)).ToNot(HaveOccurred())
					Expect(cl.Items).To(HaveLen(1))
					nc := cl.Items[0]
					Expect(nc.Spec.Config).To(HaveLen(1))
					Expect(nc.Spec.Config[0].DeviceConfig.DDPURL).Should(Equal(newerCC.Spec.DeviceConfig.DDPURL))
					Expect(nc.Spec.Config[0].DeviceConfig.FWURL).Should(Equal(newerCC.Spec.DeviceConfig.FWURL))

				})
			})

		})

		When("cc has no node selector", func() {
			It("cc.spec should be propagated to all nodes having matching accelerator", func() {
				n1 := createNode("n1")
				n2 := createNode("n2")

				createNodeInventory(n1.Name, []ethernetv1.Device{
					{
						PCIAddress: "0000:15:00.1",
						VendorID:   "testvendor",
					},
				})

				createNodeInventory(n2.Name, []ethernetv1.Device{
					{
						PCIAddress: "0000:15:00.2",
						VendorID:   "testvendor",
					},
				})

				cc := createDeviceConfig("older-cluster-config", func(cc *ethernetv1.EthernetClusterConfig) {
					cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
						VendorID: "testvendor",
					}
					cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
						DDPURL: "testddpurl",
						FWURL:  "testfwurl",
					}
					cc.Spec.Priority = 1
				})

				_ = reconcile()

				nc := new(ethernetv1.EthernetNodeConfig)
				Expect(k8sClient.Get(context.TODO(), client.ObjectKey{Name: n1.Name, Namespace: NAMESPACE}, nc)).ToNot(HaveOccurred())
				Expect(nc.Spec.Config).To(HaveLen(1))
				Expect(nc.Spec.Config[0].DeviceConfig.DDPURL).Should(Equal(cc.Spec.DeviceConfig.DDPURL))
				Expect(nc.Spec.Config[0].DeviceConfig.FWURL).Should(Equal(cc.Spec.DeviceConfig.FWURL))

				nc = new(ethernetv1.EthernetNodeConfig)
				Expect(k8sClient.Get(context.TODO(), client.ObjectKey{Name: n2.Name, Namespace: NAMESPACE}, nc)).ToNot(HaveOccurred())
				Expect(nc.Spec.Config).To(HaveLen(1))
				Expect(nc.Spec.Config[0].DeviceConfig.DDPURL).Should(Equal(cc.Spec.DeviceConfig.DDPURL))
				Expect(nc.Spec.Config[0].DeviceConfig.FWURL).Should(Equal(cc.Spec.DeviceConfig.FWURL))
			})
		})

		When("when cc doesn't match to any node it ", func() {
			It("should not be reflected in any nc", func() {
				node := createNode("foo")
				createNodeInventory(node.Name, []ethernetv1.Device{
					{
						PCIAddress: "0000:15:00.1",
						VendorID:   "testvendor",
					},
				})

				createDeviceConfig("testconfig", func(cc *ethernetv1.EthernetClusterConfig) {
					cc.Spec.NodeSelector["foo"] = "bar"
					cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{PCIAddress: "0000:15:00.1"}
					cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
						DDPURL: "testddpurl",
						FWURL:  "testfwurl",
					}
				})

				_ = reconcile()

				nodeConfig := new(ethernetv1.EthernetNodeConfig)
				Expect(k8sClient.Get(context.TODO(), client.ObjectKey{Name: node.Name, Namespace: NAMESPACE}, nodeConfig)).ToNot(HaveOccurred())
				Expect(nodeConfig.Spec.Config).To(HaveLen(0))
			})
		})

		When("multiple cc referring to multiple accelerators existing on one node ", func() {
			It("will create one instance of NodeConfig", func() {
				node := createNode("foobar")
				createNodeInventory(node.Name, []ethernetv1.Device{
					{
						PCIAddress: "0000:15:00.1",
						DeviceID:   "id1",
						VendorID:   "testvendor",
					},
					{
						PCIAddress: "0000:15:00.2",
						DeviceID:   "id2",
						VendorID:   "testvendor",
					},
				})

				cc1 := createDeviceConfig("cc1", func(cc *ethernetv1.EthernetClusterConfig) {
					cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{DeviceID: "id1"}
					cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
						DDPURL: "testddpurl_foo",
						FWURL:  "testfwurl_foo",
					}
				})

				cc2 := createDeviceConfig("cc2", func(cc *ethernetv1.EthernetClusterConfig) {
					cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{DeviceID: "id2"}
					cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
						DDPURL: "testddpurl_bar",
						FWURL:  "testfwurl_bar",
					}
				})

				_ = reconcile()

				//Check if node config was created out of cluster config
				nodeConfigs := new(ethernetv1.EthernetNodeConfigList)
				Expect(k8sClient.List(context.TODO(), nodeConfigs)).ToNot(HaveOccurred())
				Expect(len(nodeConfigs.Items)).To(Equal(1))
				Expect(nodeConfigs.Items[0].Name).To(Equal(node.Name))
				Expect(nodeConfigs.Items[0].Spec.Config).To(HaveLen(2))

				Expect(nodeConfigs.Items[0].Spec.Config).Should(ContainElement(
					ethernetv1.DeviceNodeConfig{
						PCIAddress: "0000:15:00.1",
						DeviceConfig: ethernetv1.DeviceConfig{
							FWURL:  cc1.Spec.DeviceConfig.FWURL,
							DDPURL: cc1.Spec.DeviceConfig.DDPURL,
						},
					},
				))

				Expect(nodeConfigs.Items[0].Spec.Config).Should(ContainElement(
					ethernetv1.DeviceNodeConfig{
						PCIAddress: "0000:15:00.2",
						DeviceConfig: ethernetv1.DeviceConfig{
							FWURL:  cc2.Spec.DeviceConfig.FWURL,
							DDPURL: cc2.Spec.DeviceConfig.DDPURL,
						},
					},
				))

			})
		})

		When("PCIAddress structure is invalid ", func() {
			It("will fail to create an instance of NodeConfig", func() {

				cc := clusterConfigPrototype.DeepCopy()
				cc.Name = "foobar"

				// Valid pattern: ^[a-fA-F0-9]{2,4}:[a-fA-F0-9]{2}:[01][a-fA-F0-9]\.[0-7]$

				cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
					PCIAddress: "000:15:00.1",
				}
				err := k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceSelector.pciAddress: Invalid value:"))

				cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
					PCIAddress: "0000:1:00.1",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceSelector.pciAddress: Invalid value:"))

				cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
					PCIAddress: "0000:15:20.1",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceSelector.pciAddress: Invalid value:"))

				cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
					PCIAddress: "0000:15:0.1",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceSelector.pciAddress: Invalid value:"))

				cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
					PCIAddress: "0000:15:00",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceSelector.pciAddress: Invalid value:"))

				cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
					PCIAddress: "0000:15:00.10",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceSelector.pciAddress: Invalid value:"))

				cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
					PCIAddress: "0000:1*:00.0",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceSelector.pciAddress: Invalid value:"))

				cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
					PCIAddress: "0000:00.0",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceSelector.pciAddress: Invalid value:"))

				cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
					PCIAddress: "00:10:00.0",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceSelector.pciAddress: Invalid value:"))

				cc.Spec.DeviceSelector = ethernetv1.DeviceSelector{
					PCIAddress: "address",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceSelector.pciAddress: Invalid value:"))
			})
		})

		When("FWChecksum structure is invalid ", func() {
			It("will fail to create an instance of NodeConfig", func() {

				cc := clusterConfigPrototype.DeepCopy()
				cc.Name = "foobar"

				// Valid Pattern=`^[a-fA-F0-9]{32}$`
				cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
					FWChecksum: "1234567890123456789012345678901",
				}
				err := k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceConfig.fwChecksum: Invalid value:"))

				cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
					FWChecksum: "1234567890123456789012345678901g",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceConfig.fwChecksum: Invalid value:"))

				cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
					FWChecksum: "1234567890123456789012345678901*",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceConfig.fwChecksum: Invalid value:"))

				cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
					FWChecksum: "12345678901234567890-12345678901",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceConfig.fwChecksum: Invalid value:"))

				cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
					FWChecksum: "1234567890123456789.012345678901",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceConfig.fwChecksum: Invalid value:"))
			})
		})

		When("DDPChecksum structure is invalid ", func() {
			It("will fail to create an instance of NodeConfig", func() {

				cc := clusterConfigPrototype.DeepCopy()
				cc.Name = "foobar"

				// Valid Pattern=`^[a-fA-F0-9]{32}$`
				cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
					DDPChecksum: "1234567890123456789012345678901",
				}
				err := k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceConfig.ddpChecksum: Invalid value:"))

				cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
					DDPChecksum: "1234567890123456789012345678901g",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceConfig.ddpChecksum: Invalid value:"))

				cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
					DDPChecksum: "1234567890123456789012345678901*",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceConfig.ddpChecksum: Invalid value:"))

				cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
					DDPChecksum: "12345678901234567890-12345678901",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceConfig.ddpChecksum: Invalid value:"))

				cc.Spec.DeviceConfig = ethernetv1.DeviceConfig{
					DDPChecksum: "1234567890123456789.012345678901",
				}
				err = k8sClient.Create(context.TODO(), cc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.deviceConfig.ddpChecksum: Invalid value:"))
			})
		})

		When("Manager is not set ", func() {
			It("will return error", func() {
				var mgr ctrl.Manager

				reconciler := EthernetClusterConfigReconciler{}

				err := reconciler.SetupWithManager(mgr)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must provide a non-nil Manager"))
			})
		})
	})
})
