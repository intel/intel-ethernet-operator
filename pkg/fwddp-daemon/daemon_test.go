// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

package daemon

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	gerrors "errors"

	"github.com/go-logr/logr"
	ethernetv1 "github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/apis/ethernet/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

type TestData struct {
	NodeConfig ethernetv1.EthernetNodeConfig
	Inventory  []ethernetv1.Device
	Node       core.Node
}

func (d *TestData) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: d.NodeConfig.Namespace,
		Name:      d.NodeConfig.Name,
	}
}

func initTestData() TestData {
	return TestData{
		NodeConfig: ethernetv1.EthernetNodeConfig{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
			Spec: ethernetv1.EthernetNodeConfigSpec{
				Config: []ethernetv1.DeviceNodeConfig{
					{
						PCIAddress: "0000:00:00.1",
						DeviceConfig: ethernetv1.DeviceConfig{
							FWURL: "http://testfwurl",
						},
					},
				},
			},
		},
		Node: core.Node{
			ObjectMeta: v1.ObjectMeta{
				Name:   "test",
				Labels: map[string]string{"fpga.ethernet.com/intel-ethernet-present": ""},
			},
		},
		Inventory: []ethernetv1.Device{
			{
				PCIAddress:    "0000:00:00.0",
				Name:          "TestName",
				Driver:        "TestDriver",
				DriverVersion: "TestDriverVersion",
				Firmware: ethernetv1.FirmwareInfo{
					MAC:     "aa:bb:cc:dd:ee:ff",
					Version: "TestFWVersion",
				},
			},
		},
	}
}

func initReconciler(toBeInitialized *NodeConfigReconciler, nodeName, namespace string) error {
	cset, err := clientset.NewForConfig(config)
	if err != nil {
		return err
	}

	r, err := NewNodeConfigReconciler(k8sClient, cset, log, nodeName, namespace)
	if err != nil {
		return err
	}

	*toBeInitialized = *r
	return nil
}

var data = TestData{}

var _ = Describe("DaemonTests", func() {
	reconciler := new(NodeConfigReconciler)
	var _ = BeforeEach(func() {
		data = initTestData()
		compatMapPath = "testdata/supported_devices.json"

		getInventory = func(_ logr.Logger) ([]ethernetv1.Device, error) {
			return data.Inventory, nil
		}
		downloadFile = func(path, url, checksum string) error {
			return nil
		}
		untarFile = func(srcPath string, dstPath string, log logr.Logger) error {
			return nil
		}
		unpackDDPArchive = func(srcPath string, dstPath string, log logr.Logger) error {
			return nil
		}
		nvmupdateExec = func(cmd *exec.Cmd, log logr.Logger) error {
			return nil
		}

		artifactsFolder = "./workdir/nvmupdate/"
	})

	var _ = Context("Reconciler", func() {
		BeforeEach(func() {
		})

		AfterEach(func() {
			nn := data.GetNamespacedName()
			if err := k8sClient.Get(context.TODO(), nn, &data.NodeConfig); err == nil {
				data.NodeConfig.Spec = ethernetv1.EthernetNodeConfigSpec{
					Config: []ethernetv1.DeviceNodeConfig{},
				}
				Expect(k8sClient.Update(context.TODO(), &data.NodeConfig)).NotTo(HaveOccurred())
				_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: nn})
				Expect(err).ToNot(HaveOccurred())
				Expect(k8sClient.Delete(context.TODO(), &data.NodeConfig)).ToNot(HaveOccurred())
			} else if errors.IsNotFound(err) {
				log.Info("Requested NodeConfig does not exists", "NodeConfig", &data.NodeConfig)
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(k8sClient.Delete(context.TODO(), &data.Node)).To(Succeed())
		})

		var _ = It("will not create a new node config reconciler, because of invalid config file", func() {

			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())

			cset, err := clientset.NewForConfig(config)
			Expect(err).ToNot(HaveOccurred())

			_ = os.MkdirAll("workdir", 0777)
			defer os.RemoveAll("workdir")

			compatMapPath = "workdir/wrong_filename.json"
			tmpfile, err := os.OpenFile(compatMapPath, os.O_RDWR|os.O_CREATE, 0777)
			Expect(err).ToNot(HaveOccurred())
			tmpfile.Close()

			_, err = NewNodeConfigReconciler(k8sClient, cset, log, data.NodeConfig.Name, data.NodeConfig.Namespace)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Failed to unmarshal config"))
		})

		var _ = It("will create empty NodeConfig if not exits", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: data.GetNamespacedName()})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Devices).To(HaveLen(0))
		})

		var _ = It("will update inventory on Reconcile()", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: data.GetNamespacedName()})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Devices).To(HaveLen(0))

			_, err = reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: data.GetNamespacedName()})
			Expect(err).ToNot(HaveOccurred())

			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Devices).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Devices[0]).To(Equal(data.Inventory[0]))
		})

		var _ = It("will update condition to Inventory up to date if Spec.Config is empty", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())

			data.NodeConfig.Spec.Config = []ethernetv1.DeviceNodeConfig{}

			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())
			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateNotRequested)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal("Inventory up to date"))
		})

		var _ = It("will update condition to UpdateFailed if no matching devices were found", func() {

			findFw = func(localpath string) (string, error) { return localpath, nil }

			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())
			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			nvmupdateExec = func(cmd *exec.Cmd, log logr.Logger) error {
				return fmt.Errorf("FAILING NVME UPDATE ON PURPOSE")
			}

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateFailed)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal("FAILING NVME UPDATE ON PURPOSE"))
		})

		var _ = It("will update condition to UpdateFailed if not able to download firmware", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			downloadErr := gerrors.New("unable to download")
			downloadFile = func(path, url, checksum string) error {
				return downloadErr
			}

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateFailed)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal(downloadErr.Error()))
		})

		var _ = It("will call for reboot, expects ok and reboot", func() {
			defer os.RemoveAll(artifactsFolder)

			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			downloadFile = func(localpath, url, checksum string) error {

				updateDir := path.Join(artifactsFolder, data.Inventory[0].PCIAddress)
				updatePath := updateResultPath(updateDir)

				err := os.MkdirAll(updateDir, 0777)
				Expect(err).ToNot(HaveOccurred())

				tmpfile, err := os.OpenFile(updatePath, os.O_RDWR|os.O_CREATE, 0777)
				Expect(err).ToNot(HaveOccurred())
				defer tmpfile.Close()

				nvmupdateOutput := `<?xml version="1.0" encoding="UTF-8"?>
						   <DeviceUpdate lang="en">
						           <RebootRequired> 1 </RebootRequired>
						   </DeviceUpdate>`

				_, err = tmpfile.Write([]byte(nvmupdateOutput))
				Expect(err).ToNot(HaveOccurred())

				return nil
			}

			wasRebootCalled := false
			findFw = func(localpath string) (string, error) { return localpath, nil }

			execCmd = func(args []string, log logr.Logger) (string, error) {
				for _, part := range args {
					if strings.Contains(part, "reboot") {
						wasRebootCalled = true
					}
				}
				return "", nil
			}

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			Expect(wasRebootCalled).To(Equal(true))

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdatePostUpdateReboot)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal("Post-update node reboot"))
		})

		var _ = It("will call for reboot, expects ok but no reboot", func() {
			defer os.RemoveAll(artifactsFolder)

			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			downloadFile = func(localpath, url, checksum string) error {

				updateDir := path.Join(artifactsFolder, data.Inventory[0].PCIAddress)
				updatePath := updateResultPath(updateDir)

				err := os.MkdirAll(updateDir, 0777)
				Expect(err).ToNot(HaveOccurred())

				tmpfile, err := os.OpenFile(updatePath, os.O_RDWR|os.O_CREATE, 0777)
				Expect(err).ToNot(HaveOccurred())
				defer tmpfile.Close()

				nvmupdateOutput := `<?xml version="1.0" encoding="UTF-8"?>
						   <DeviceUpdate lang="en">
						           <RebootRequired> 0 </RebootRequired>
						   </DeviceUpdate>`

				_, err = tmpfile.Write([]byte(nvmupdateOutput))
				Expect(err).ToNot(HaveOccurred())

				return nil
			}

			wasRebootCalled := false
			execCmd = func(args []string, log logr.Logger) (string, error) {
				for _, part := range args {
					if strings.Contains(part, "reboot") {
						wasRebootCalled = true
					}
				}
				return "", nil
			}

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())
			Expect(wasRebootCalled).To(Equal(false))

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateSucceeded)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal("Updated successfully"))
		})

		var _ = It("will call for reboot, expect failure", func() {
			defer os.RemoveAll(artifactsFolder)

			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			downloadFile = func(localpath, url, checksum string) error {

				updateDir := path.Join(artifactsFolder, data.Inventory[0].PCIAddress)
				updatePath := updateResultPath(updateDir)

				err := os.MkdirAll(updateDir, 0777)
				Expect(err).ToNot(HaveOccurred())

				tmpfile, err := os.OpenFile(updatePath, os.O_RDWR|os.O_CREATE, 0777)
				Expect(err).ToNot(HaveOccurred())

				defer tmpfile.Close()

				nvmupdateOutput := `<?xml version="1.0" encoding="UTF-8"?>
						   <DeviceUpdate lang="en">
						           <RebootRequired> 1 </RebootRequired>
						   </DeviceUpdate>`

				_, err = tmpfile.Write([]byte(nvmupdateOutput))
				Expect(err).ToNot(HaveOccurred())

				return nil
			}

			wasRebootCalled := false

			execCmd = func(args []string, log logr.Logger) (string, error) {
				for _, part := range args {
					if strings.Contains(part, "reboot") {
						wasRebootCalled = true
					}
				}

				return "", gerrors.New("failed to reboot")
			}

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			Expect(wasRebootCalled).To(Equal(true))

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateFailed)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(ContainSubstring("failed to reboot"))
		})

		var _ = It("will update condition to UpdateFailed if not able to untar firmware", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			untarErr := gerrors.New("unable to untar")
			untarFile = func(srcPath string, dstPath string, log logr.Logger) error {
				return untarErr
			}

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateFailed)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal(untarErr.Error()))
		})

		var _ = It("will update condition to UpdateFailed if firmware updater binary fails", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			nvmupdateExec = func(cmd *exec.Cmd, log logr.Logger) error {
				return fmt.Errorf("FAILING NVME UPDATE ON PURPOSE")
			}

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateFailed)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(SatisfyAny(
				ContainSubstring("FAILING NVME UPDATE ON PURPOSE")))
		})

		var _ = It("will update condition to UpdateFailed if firmware update fails", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			fwErr := gerrors.New("unable to update firmware")
			nvmupdateExec = func(cmd *exec.Cmd, log logr.Logger) error {
				return fwErr
			}

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateFailed)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal(fwErr.Error()))
		})
		var _ = It("will update condition to UpdateSucceeded after successful firmware update", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			wasRebootCalled := false
			execCmd = func(args []string, log logr.Logger) (string, error) {
				for _, part := range args {
					if strings.Contains(part, "reboot") {
						wasRebootCalled = true
					}
				}
				return "", nil
			}

			rootAttr := &syscall.SysProcAttr{
				Credential: &syscall.Credential{Uid: 0, Gid: 0},
			}
			nvmupdateExec = func(cmd *exec.Cmd, log logr.Logger) error {
				Expect(cmd.SysProcAttr).To(Equal(rootAttr))
				Expect(cmd.Dir).To(Equal(path.Join(artifactsFolder, data.NodeConfig.Spec.Config[0].PCIAddress)))
				return nil
			}

			downloadFile = func(localpath, url, checksum string) error {

				updateDir := path.Join(artifactsFolder, data.Inventory[0].PCIAddress)
				updatePath := updateResultPath(updateDir)

				err := os.MkdirAll(updateDir, 0777)
				Expect(err).ToNot(HaveOccurred())

				tmpfile, err := os.OpenFile(updatePath, os.O_RDWR|os.O_CREATE, 0777)
				Expect(err).ToNot(HaveOccurred())
				defer tmpfile.Close()

				nvmupdateOutput := `<?xml version="1.0" encoding="UTF-8"?>
						   <DeviceUpdate lang="en">
						           <RebootRequired> 0 </RebootRequired>
						   </DeviceUpdate>`

				_, err = tmpfile.Write([]byte(nvmupdateOutput))
				Expect(err).ToNot(HaveOccurred())

				return nil
			}

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})

			Expect(err).ToNot(HaveOccurred())
			Expect(wasRebootCalled).To(Equal(false))
			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateSucceeded)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal("Updated successfully"))

			_, err = reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})

			Expect(err).ToNot(HaveOccurred())
			nodeConfigs = &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateSucceeded)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal("Updated successfully"))
		})

		var _ = It("if FWUrl is empty then FW update is skipped and status updated to UpdateSucceeded", func() {

			data.NodeConfig.Spec.Config[0].DeviceConfig.FWURL = ""

			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			rootAttr := &syscall.SysProcAttr{
				Credential: &syscall.Credential{Uid: 0, Gid: 0},
			}
			nvmupdateExec = func(cmd *exec.Cmd, log logr.Logger) error {
				Expect(cmd.SysProcAttr).To(Equal(rootAttr))
				Expect(cmd.Dir).To(Equal(path.Join(artifactsFolder, data.NodeConfig.Spec.Config[0].PCIAddress)))
				return nil
			}

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateSucceeded)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal("Updated successfully"))
		})

		var _ = It("will fail because of PCIAddress not matching pattern", func() {

			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())

			// Valid pattern: ^[a-fA-F0-9]{2,4}:[a-fA-F0-9]{2}:[01][a-fA-F0-9]\.[0-7]$

			data.NodeConfig.Spec.Config[0].PCIAddress = "0:00:00.1"
			err := k8sClient.Create(context.TODO(), &data.NodeConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.config.PCIAddress: Invalid value:"))

			data.NodeConfig.Spec.Config[0].PCIAddress = "0000:00:00.a"
			err = k8sClient.Create(context.TODO(), &data.NodeConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.config.PCIAddress: Invalid value:"))

			data.NodeConfig.Spec.Config[0].PCIAddress = "0:00:00"
			err = k8sClient.Create(context.TODO(), &data.NodeConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.config.PCIAddress: Invalid value:"))

			data.NodeConfig.Spec.Config[0].PCIAddress = "0:00:00"
			err = k8sClient.Create(context.TODO(), &data.NodeConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.config.PCIAddress: Invalid value:"))

			data.NodeConfig.Spec.Config[0].PCIAddress = "0000:00:20.1"
			err = k8sClient.Create(context.TODO(), &data.NodeConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.config.PCIAddress: Invalid value:"))

			data.NodeConfig.Spec.Config[0].PCIAddress = "0000:00:0.1"
			err = k8sClient.Create(context.TODO(), &data.NodeConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.config.PCIAddress: Invalid value:"))

			data.NodeConfig.Spec.Config[0].PCIAddress = "0000:0:00.1"
			err = k8sClient.Create(context.TODO(), &data.NodeConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.config.PCIAddress: Invalid value:"))

			data.NodeConfig.Spec.Config[0].PCIAddress = "0000:00:00.*"
			err = k8sClient.Create(context.TODO(), &data.NodeConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.config.PCIAddress: Invalid value:"))
		})

		var _ = It("will update condition to UpdateFailed if not able to download DDP", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())

			data.NodeConfig.Spec.Config[0].DeviceConfig.FWURL = ""
			data.NodeConfig.Spec.Config[0].DeviceConfig.DDPURL = "http://testddpurl"
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			downloadErr := gerrors.New("unable to download DDP")
			downloadFile = func(path, url, checksum string) error {
				return downloadErr
			}

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateFailed)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal(downloadErr.Error()))
		})

		var _ = It("will update condition to UpdateFailed if not able to unpack DDP", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())

			data.NodeConfig.Spec.Config[0].DeviceConfig.FWURL = ""
			data.NodeConfig.Spec.Config[0].DeviceConfig.DDPURL = "http://testddpurl"
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			unzipErr := gerrors.New("unable to unzip")
			unpackDDPArchive = func(srcPath string, dstPath string, log logr.Logger) error {
				return unzipErr
			}

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateFailed)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal(unzipErr.Error()))
		})

		var _ = It("will update condition to UpdateFailed if DDP update fails", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())

			data.NodeConfig.Spec.Config[0].DeviceConfig.FWURL = ""
			data.NodeConfig.Spec.Config[0].DeviceConfig.DDPURL = "http://testddpurl"
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdateFailed)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(ContainSubstring("expected to find exactly 1 file ending with '.pkg', but found 0"))
		})

		var _ = It("will force reboot node on successful DDP update", func() {
			Expect(k8sClient.Create(context.TODO(), &data.Node)).To(Succeed())

			data.NodeConfig.Spec.Config[0].DeviceConfig.FWURL = ""
			data.NodeConfig.Spec.Config[0].DeviceConfig.DDPURL = "http://testddpurl"
			Expect(k8sClient.Create(context.TODO(), &data.NodeConfig)).To(Succeed())

			data.Inventory[0].PCIAddress = "0000:00:00.1"

			Expect(initReconciler(reconciler, data.NodeConfig.Name, data.NodeConfig.Namespace)).To(Succeed())

			tempFile, err := ioutil.TempFile("/tmp", "daemontest")
			Expect(err).To(Succeed())
			defer tempFile.Close()

			findDdp = func(targetPath string) (string, error) {
				return tempFile.Name(), nil
			}
			ocpDdpUpdatePath = "/tmp"

			wasRebootCalled := false

			execCmd = func(args []string, log logr.Logger) (string, error) {
				for _, part := range args {
					if strings.Contains(part, "reboot") {
						wasRebootCalled = true
					}
					if strings.Contains(part, "Device Serial") {
						return "devId\n", nil
					}
				}
				return "", nil
			}

			_, err = reconciler.Reconcile(context.TODO(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: data.NodeConfig.Namespace,
				Name:      data.NodeConfig.Name,
			}})
			Expect(err).ToNot(HaveOccurred())

			Expect(wasRebootCalled).To(Equal(true))

			nodeConfigs := &ethernetv1.EthernetNodeConfigList{}
			Expect(k8sClient.List(context.TODO(), nodeConfigs)).To(Succeed())
			Expect(nodeConfigs.Items).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions).To(HaveLen(1))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Reason).To(Equal(string(UpdatePostUpdateReboot)))
			Expect(nodeConfigs.Items[0].Status.Conditions[0].Message).To(Equal("Post-update node reboot"))
		})
	})
})
