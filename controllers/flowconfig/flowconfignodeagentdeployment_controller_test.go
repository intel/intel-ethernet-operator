// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flowconfig

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// WaitForPodCreation will wait for POD creation
// In the artificial env POD will never be in running state, due to missing container image
func WaitForPodCreation(core client.Client, podName, ns string, timeout, interval time.Duration) error {
	return wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		pod := &corev1.Pod{}
		err = core.Get(ctx, client.ObjectKey{
			Namespace: ns,
			Name:      podName,
		}, pod)
		fmt.Println("wait get err:", err, " pod.Status.Phase:", pod.Status.Phase)

		if err != nil {
			if strings.Contains(err.Error(), fmt.Sprintf("pods \"%s\" not found", podName)) {
				return false, nil
			}

			return false, err
		}

		return true, nil
	})
}

var _ = Describe("FlowConfigNodeAgentDeployment controller", func() {
	var (
		namespacePrototype = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-dummy",
			},
		}
	)

	const (
		namespaceDefault  = "default"
		namespaceIntel    = "intel-ethernet-operator-system"
		nodeName1         = "k8snode-1"
		nodeName2         = "k8snode-2"
		vfPoolName        = "intel.com/cvl_uft_admin"
		clusterTypeEnvKey = "ETHERNET_GENERIC_K8S"

		timeout  = 4 * time.Second
		interval = 250 * time.Millisecond
	)

	createNamespace := func(name string) *corev1.Namespace {
		namespace := namespacePrototype.DeepCopy()
		namespace.Name = name

		Expect(k8sClient.Create(context.TODO(), namespace)).ToNot(HaveOccurred())

		return namespace
	}

	getFlowConfigNodeAgentDeployment := func(namespace string, configurers ...func(flow *flowconfigv1.FlowConfigNodeAgentDeployment)) *flowconfigv1.FlowConfigNodeAgentDeployment {
		nodeAgent := &flowconfigv1.FlowConfigNodeAgentDeployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "flowconfig.intel.com/v1",
				Kind:       "FlowConfigNodeAgentDeployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "flowconfig-daemon-flowconfig-daemon",
				Namespace: namespace,
				Labels:    map[string]string{"control-plane": "flowconfig-daemon"},
			},
			Spec: flowconfigv1.FlowConfigNodeAgentDeploymentSpec{},
		}

		for _, configure := range configurers {
			configure(nodeAgent)
		}

		return nodeAgent
	}

	deleteFlowConfigNodeAgentDeployment := func(namespace string) {
		fcnaDeployment := &unstructured.Unstructured{}
		fcnaDeployment.SetName("flowconfig-daemon-flowconfig-daemon")
		fcnaDeployment.SetNamespace(namespace)
		fcnaDeployment.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "flowconfig.intel.com",
			Kind:    "FlowConfigNodeAgentDeployment",
			Version: "v1",
		})

		err := k8sClient.Delete(context.Background(), fcnaDeployment)
		Expect(err).To(BeNil())
	}

	verifyExpectedPODDefintion := func(namespace, podName, nodeName, networkString, poolName string, container, amount int, checkers ...func(pod *corev1.Pod)) {
		// wait for POD, expected to be created
		err := WaitForPodCreation(k8sClient, podName, namespace, timeout, interval)
		Expect(err).To(BeNil())

		pod := &corev1.Pod{}
		err = k8sClient.Get(context.Background(), client.ObjectKey{
			Name:      podName,
			Namespace: namespace}, pod)
		Expect(err).To(BeNil())

		Expect(pod.Name).Should(Equal(podName))
		Expect(pod.Namespace).Should(Equal(namespace))
		Expect(pod.Spec.NodeSelector).ShouldNot(BeNil())
		val, ok := pod.Spec.NodeSelector["kubernetes.io/hostname"]
		Expect(ok).Should(BeTrue())
		Expect(val).Should(Equal(nodeName))
		Expect(len(pod.Labels)).Should(Equal(0))
		limits := pod.Spec.Containers[container].Resources.Limits[corev1.ResourceName(poolName)]
		value, isError := limits.AsInt64()
		Expect(isError).To(BeTrue())
		Expect(value).Should(Equal(int64(amount)))

		requests := pod.Spec.Containers[container].Resources.Requests[corev1.ResourceName(poolName)]
		value, isError = requests.AsInt64()
		Expect(isError).To(BeTrue())
		Expect(value).Should(Equal(int64(amount)))
		Expect(pod.Annotations).Should(HaveKeyWithValue("k8s.v1.cni.cncf.io/networks", networkString))

		volumes := pod.Spec.Containers[container].VolumeMounts
		found := false
		for _, v := range volumes {
			if v.MountPath == "/lib/firmware/intel/ice/ddp" {
				found = true
			}
		}
		Expect(found).To(BeTrue())

		for _, check := range checkers {
			check(pod)
		}
	}

	Context("Cluster without nodes", func() {
		It("Verify that cluster does not have nodes", func() {
			nodeList := &corev1.NodeList{}
			err := k8sClient.List(context.Background(), nodeList)

			Expect(err).To(BeNil())
			Expect(len(nodeList.Items)).Should(Equal(0))
		})

		It("Verify that it is possible to create Custom Resource, but POD will not be created", func() {
			Eventually(func() bool {
				err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
					flow.Spec.DCFVfPoolName = vfPoolName
					flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
				}))
				return err == nil
			}, timeout, interval).Should(BeTrue())
			defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

			obj := &flowconfigv1.FlowConfigNodeAgentDeployment{}
			Eventually(func() string {
				return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault, 1*time.Second, obj).Error()
			}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", fmt.Sprintf("flowconfig-daemon-%s", nodeName1))))
		})
	})

	Context("Cluster with nodes, default namespace", func() {
		Context("Expects that controller will create POD", func() {
			It("Node with one resource, controller should create POD with that resource", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
				})
				defer deleteNode(node)

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(),
						getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
							flow.Spec.DCFVfPoolName = vfPoolName
							flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
						}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
					"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1)
			})

			It("Node with two expected resources, controller should add all resources to the POD", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(2, resource.DecimalSI)
				})
				defer deleteNode(node)

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
					"flowconfig-daemon-sriov-cvl0-admin, flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 2)
			})

			It("Node with mix of resources, controller should add only resources defined by CustomResource", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity["intel.com/extra"] = *resource.NewQuantity(2, resource.DecimalSI)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
					node.Status.Capacity["intel.com/dummy"] = *resource.NewQuantity(4, resource.DecimalSI)
				})
				defer deleteNode(node)

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
					"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1)
			})

			It("Two Nodes with different amount of resource, controller should create POD with that resources", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
				})
				defer deleteNode(node)

				node2 := createNode(nodeName2, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(2, resource.DecimalSI)
				})
				defer deleteNode(node2)

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
				defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName2), namespaceDefault)
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
					"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1)

				verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName2), nodeName2,
					"flowconfig-daemon-sriov-cvl0-admin, flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 2)
			})

			It("Verify if resource and limits are added to correct container (UFT)", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
				})
				defer deleteNode(node)

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(),
						getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
							flow.Spec.DCFVfPoolName = vfPoolName
							flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
						}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
					"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1, func(pod *corev1.Pod) {
						limitsList := corev1.ResourceList{}
						limitsList["memory"] = *resource.NewQuantity(209715200, resource.BinarySI)
						Expect(pod.Spec.Containers[0].Resources.Limits).ToNot(BeNil())
						Expect(pod.Spec.Containers[0].Resources.Limits).To(HaveLen(3))
						Expect(pod.Spec.Containers[0].Resources.Limits).To(HaveKeyWithValue(corev1.ResourceName("memory"),
							MatchQuantityObject(*resource.NewQuantity(209715200, resource.BinarySI))))
						Expect(pod.Spec.Containers[0].Resources.Limits).To(HaveKeyWithValue(corev1.ResourceName("hugepages-2Mi"),
							MatchQuantityObject(*resource.NewQuantity(2147483648, resource.BinarySI))))
						Expect(pod.Spec.Containers[0].Resources.Limits).To(HaveKeyWithValue(corev1.ResourceName("intel.com/cvl_uft_admin"),
							MatchQuantityObject(*resource.NewQuantity(1, resource.DecimalSI))))
						Expect(pod.Spec.Containers[0].Resources.Requests).ToNot(BeNil())
						Expect(pod.Spec.Containers[0].Resources.Requests).To(HaveLen(3))
						Expect(pod.Spec.Containers[0].Resources.Requests).To(HaveKeyWithValue(corev1.ResourceName("memory"),
							MatchQuantityObject(*resource.NewQuantity(209715200, resource.BinarySI))))
						Expect(pod.Spec.Containers[0].Resources.Requests).To(HaveKeyWithValue(corev1.ResourceName("hugepages-2Mi"),
							MatchQuantityObject(*resource.NewQuantity(2147483648, resource.BinarySI))))
						Expect(pod.Spec.Containers[0].Resources.Requests).To(HaveKeyWithValue(corev1.ResourceName("intel.com/cvl_uft_admin"),
							MatchQuantityObject(*resource.NewQuantity(1, resource.DecimalSI))))
					})
			})

			It("Delete POD created by controller, expected POD is recreated", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
				})
				defer deleteNode(node)

				flowconfig := getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
					flow.Spec.DCFVfPoolName = vfPoolName
					flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
				})

				Eventually(func() error {
					return k8sClient.Create(context.Background(), flowconfig)
				}, timeout, interval).Should(BeNil())

				defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				// wait for POD, expected to be created
				pod := &corev1.Pod{}
				Eventually(func() error {
					return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault, 1*time.Second, pod)
				}, timeout, interval).Should(BeNil())

				state := bool(true)
				pod.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
					{
						Kind:               "FlowConfigNodeAgentDeployment",
						APIVersion:         "flowconfig.intel.com/v1",
						Name:               flowconfig.ObjectMeta.Name,
						UID:                flowconfig.ObjectMeta.UID,
						Controller:         &state,
						BlockOwnerDeletion: &state,
					},
				}

				err := k8sClient.Update(context.Background(), pod)
				Expect(err).To(BeNil())

				By("Delete POD and wait for its recreation by controller")
				deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)

				Eventually(func() error {
					return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault, 1*time.Second, pod)
				}, timeout, interval).Should(BeNil())

				fmt.Println("POD ", pod.ObjectMeta.OwnerReferences)
			})

			It("Update CR by changing the DCFVfPoolName, expected POD should be recreated with new configuration", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
					node.Status.Capacity["intel.com/second_admin"] = *resource.NewQuantity(2, resource.DecimalSI)
				})
				defer deleteNode(node)

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
					"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1)

				By("Update CR by changing DCFVfPoolName")

				fcnaDeployment := &flowconfigv1.FlowConfigNodeAgentDeployment{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      "flowconfig-daemon-flowconfig-daemon",
					Namespace: namespaceDefault,
				}, fcnaDeployment)
				Expect(err).To(BeNil())

				fcnaDeployment.Spec.DCFVfPoolName = "intel.com/second_admin"
				err = k8sClient.Update(context.Background(), fcnaDeployment)
				Expect(err).To(BeNil())

				time.Sleep(interval) // give some time for the controller to do the job

				verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
					"flowconfig-daemon-sriov-cvl0-admin, flowconfig-daemon-sriov-cvl0-admin", fcnaDeployment.Spec.DCFVfPoolName, 0, 2)
			})

			It("Update CR by changing the NADAnnotation, expected POD should be recreated with new configuration", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
				})
				defer deleteNode(node)

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
					"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1)

				By("Update CR by changing NADAnnotation")

				fcnaDeployment := &flowconfigv1.FlowConfigNodeAgentDeployment{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      "flowconfig-daemon-flowconfig-daemon",
					Namespace: namespaceDefault,
				}, fcnaDeployment)
				Expect(err).To(BeNil())

				fcnaDeployment.Spec.NADAnnotation = "flowconfig-daemon-sriov-temp"
				err = k8sClient.Update(context.Background(), fcnaDeployment)
				Expect(err).To(BeNil())

				time.Sleep(interval) // give some time for the controller to do the job

				verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
					"flowconfig-daemon-sriov-temp", vfPoolName, 0, 1)
			})

			It("Two nodes, one without resources, expected to create only POD on node with resources", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
				})
				defer deleteNode(node)

				node2 := createNode(nodeName2, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity["vfPoolName"] = *resource.NewQuantity(2, resource.DecimalSI)
				})
				defer deleteNode(node2)

				tmpWriteBuffer := bytes.NewBuffer(nil)
				zlog := zap.New(zap.WriteTo(tmpWriteBuffer), zap.UseDevMode(true))
				logf.SetLogger(zlog)
				nodeAgentDeploymentRc.Log = zlog

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
					"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1)

				Eventually(func() string {
					return tmpWriteBuffer.String()
				}, timeout, interval).Should(ContainSubstring("No resources present on node"))

				// POD expected to not to be created
				obj := &flowconfigv1.FlowConfigNodeAgentDeployment{}
				Eventually(func() string {
					return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName2), namespaceDefault, 1*time.Second, obj).Error()
				}, "1s", interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", fmt.Sprintf("flowconfig-daemon-%s", nodeName2))))
			})
		})

		Context("Expects that controller will drop request, POD will not be created", func() {
			It("Node without defined resources", func() {
				node := createNode(nodeName1)
				defer deleteNode(node)

				tmpWriteBuffer := bytes.NewBuffer(nil)
				zlog := zap.New(zap.WriteTo(tmpWriteBuffer), zap.UseDevMode(true))
				logf.SetLogger(zlog)
				nodeAgentDeploymentRc.Log = zlog

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				// wait for POD, expected not to be created, due to missing resources on node
				Eventually(func() string {
					return tmpWriteBuffer.String()
				}, timeout, interval).Should(ContainSubstring("No resources present on node"))

				// POD expected to not to be created
				obj := &flowconfigv1.FlowConfigNodeAgentDeployment{}
				Eventually(func() string {
					return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault, 1*time.Second, obj).Error()
				}, "1s", interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", fmt.Sprintf("flowconfig-daemon-%s", nodeName1))))
			})

			It("Node with defined resources but equal to zero", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(0, resource.DecimalSI)
				})
				defer deleteNode(node)

				tmpWriteBuffer := bytes.NewBuffer(nil)
				zlog := zap.New(zap.WriteTo(tmpWriteBuffer), zap.UseDevMode(true))
				logf.SetLogger(zlog)
				nodeAgentDeploymentRc.Log = zlog

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				Eventually(func() string {
					return tmpWriteBuffer.String()
				}, timeout, interval).Should(ContainSubstring("No resources present on node"))

				// POD expected to not to be created
				obj := &flowconfigv1.FlowConfigNodeAgentDeployment{}
				Eventually(func() string {
					return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault, 1*time.Second, obj).Error()
				}, "1s", interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", fmt.Sprintf("flowconfig-daemon-%s", nodeName1))))
			})

			It("Node with resource, but different than the one defined in Custom Resource", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity["intel.com/dummy"] = *resource.NewQuantity(1, resource.DecimalSI)
				})
				defer deleteNode(node)

				tmpWriteBuffer := bytes.NewBuffer(nil)
				zlog := zap.New(zap.WriteTo(tmpWriteBuffer), zap.UseDevMode(true))
				logf.SetLogger(zlog)
				nodeAgentDeploymentRc.Log = zlog

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				Eventually(func() string {
					return tmpWriteBuffer.String()
				}, timeout, interval).Should(ContainSubstring("No resources present on node"))

				// wait for POD, expected not to be created, due to missing resources on node
				obj := &flowconfigv1.FlowConfigNodeAgentDeployment{}
				Eventually(func() string {
					return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault, 1*time.Second, obj).Error()
				}, "1s", interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", fmt.Sprintf("flowconfig-daemon-%s", nodeName1))))
			})

			It("One node, missing DCFVfPoolName and NADAnnotation in CR, expected no error", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
				})
				defer deleteNode(node)

				tmpWriteBuffer := bytes.NewBuffer(nil)
				zlog := zap.New(zap.WriteTo(tmpWriteBuffer), zap.UseDevMode(true))
				logf.SetLogger(zlog)
				nodeAgentDeploymentRc.Log = zlog

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				Eventually(func() string {
					return tmpWriteBuffer.String()
				}, timeout, interval).Should(ContainSubstring("NADAnnotation is not defined, will try to get one after 30 seconds"))

				// POD expected to not to be created
				obj := &flowconfigv1.FlowConfigNodeAgentDeployment{}
				Eventually(func() string {
					return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault, 1*time.Second, obj).Error()
				}, "1s", interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", fmt.Sprintf("flowconfig-daemon-%s", nodeName1))))
			})

			It("One node, missing DCFVfPoolName in CR, expected no error", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
				})
				defer deleteNode(node)

				tmpWriteBuffer := bytes.NewBuffer(nil)
				zlog := zap.New(zap.WriteTo(tmpWriteBuffer), zap.UseDevMode(true))
				logf.SetLogger(zlog)
				nodeAgentDeploymentRc.Log = zlog

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(),
						getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
							flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
						}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				Eventually(func() string {
					return tmpWriteBuffer.String()
				}, timeout, interval).Should(ContainSubstring("No resources present on node"))

				// POD expected to not to be created
				obj := &flowconfigv1.FlowConfigNodeAgentDeployment{}
				Eventually(func() string {
					return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault, 1*time.Second, obj).Error()
				}, "1s", interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", fmt.Sprintf("flowconfig-daemon-%s", nodeName1))))
			})

			It("One node, missing NADAnnotation in CR, expected no error", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(10, resource.DecimalSI)
				})

				defer deleteNode(node)

				tmpWriteBuffer := bytes.NewBuffer(nil)
				zlog := zap.New(zap.WriteTo(tmpWriteBuffer), zap.UseDevMode(true))
				logf.SetLogger(zlog)
				nodeAgentDeploymentRc.Log = zlog

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(),
						getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
							flow.Spec.DCFVfPoolName = vfPoolName
						}))
					return err == nil
				}, timeout, interval).Should(BeTrue())
				defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

				Eventually(func() string {
					return tmpWriteBuffer.String()
				}, timeout, interval).Should(ContainSubstring("NADAnnotation is not defined, will try to get one after 30 seconds"))

				// POD expected to not to be created
				obj := &flowconfigv1.FlowConfigNodeAgentDeployment{}
				Eventually(func() string {
					return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault, 1*time.Second, obj).Error()
				}, "1s", interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", fmt.Sprintf("flowconfig-daemon-%s", nodeName1))))
			})
		})

		// Testing framework does not support garbage collection in envtest.
		// Following https://book.kubebuilder.io/reference/envtest.html#testing-considerations
		// user should test OwnerReferences to confirm that object belongs to controller
		Context("Verify if controller correctly cleanups nodes", func() {
			It("Delete custom resources, expected that controller will delete POD on each node", func() {
				node := createNode(nodeName1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
					node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
				})
				defer deleteNode(node)

				Eventually(func() bool {
					err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
					return err == nil
				}, timeout, interval).Should(BeTrue())

				// wait for POD, expected to be created
				err := WaitForPodCreation(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault, timeout, interval)
				Expect(err).To(BeNil())

				pod := &corev1.Pod{}
				err = k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      fmt.Sprintf("flowconfig-daemon-%s", nodeName1),
					Namespace: namespaceDefault}, pod)
				Expect(err).To(BeNil())
				Expect(pod.Name).Should(Equal(fmt.Sprintf("flowconfig-daemon-%s", nodeName1)))

				instance := &flowconfigv1.FlowConfigNodeAgentDeployment{}
				err = k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      "flowconfig-daemon-flowconfig-daemon",
					Namespace: "default"}, instance)
				Expect(err).To(BeNil())

				By("Create expected OwnerReferences")
				state := bool(true)
				expectedOwnerReference := metav1.OwnerReference{
					Kind:               "FlowConfigNodeAgentDeployment",
					APIVersion:         "flowconfig.intel.com/v1",
					Name:               instance.ObjectMeta.Name,
					UID:                instance.ObjectMeta.UID,
					Controller:         &state,
					BlockOwnerDeletion: &state,
				}

				pod.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
					{
						Kind:               "FlowConfigNodeAgentDeployment",
						APIVersion:         "flowconfig.intel.com/v1",
						Name:               instance.ObjectMeta.Name,
						UID:                instance.ObjectMeta.UID,
						Controller:         &state,
						BlockOwnerDeletion: &state,
					},
				}

				err = k8sClient.Update(context.Background(), pod)
				Expect(err).To(BeNil())
				By("Delete CR and check POD references - POD will be not deleted due to envtest constraints")
				deleteFlowConfigNodeAgentDeployment(namespaceDefault)
				defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)

				time.Sleep(interval)
				err = k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      fmt.Sprintf("flowconfig-daemon-%s", nodeName1),
					Namespace: namespaceDefault}, pod)
				Expect(err).To(BeNil())
				Expect(pod.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))
			})
		})
	})

	Context("Cluster with nodes, CR is going to be created in custom namespace other than default one", func() {
		// This is done by purpose, controller runtime envTest framework does not offer full functionality and in result it is not possible to delete custom namespace
		// https://github.com/kubernetes-sigs/controller-runtime/issues/880
		// In result, order of tests matters and all tests that are using custom namespace has to be done here
		It("Create namespace", func() {
			_ = createNamespace(namespaceIntel)
			// defer deleteNamespace(ns)
		})

		It("Node with one resource, controller should create POD with that resource", func() {
			node := createNode(nodeName1, func(node *corev1.Node) {
				node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
				node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
			})
			defer deleteNode(node)

			Eventually(func() bool {
				err := k8sClient.Create(context.Background(), getFlowConfigNodeAgentDeployment(namespaceIntel, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
					flow.Spec.DCFVfPoolName = vfPoolName
					flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
				}))
				return err == nil
			}, timeout, interval).Should(BeTrue())
			defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceIntel)
			defer deleteFlowConfigNodeAgentDeployment(namespaceIntel)

			verifyExpectedPODDefintion(namespaceIntel, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
				"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1)
		})
	})

	Context("Verify if controller correctly handles changes within cluster", func() {
		It("Add node to cluster, expect that POD will be created on it", func() {
			node := createNode(nodeName1, func(node *corev1.Node) {
				node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
				node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
			})
			defer deleteNode(node)

			By("Create custom resource")
			Eventually(func() bool {
				err := k8sClient.Create(context.Background(),
					getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
				return err == nil
			}, timeout, interval).Should(BeTrue())
			defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
			defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName2), namespaceDefault)
			defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

			By("Verify if POD was created")
			verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
				"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1)

			By("Verify that there is no second POD")
			obj := &flowconfigv1.FlowConfigNodeAgentDeployment{}
			Eventually(func() string {
				return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName2), namespaceDefault, 1*time.Second, obj).Error()
			}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", fmt.Sprintf("flowconfig-daemon-%s", nodeName2))))

			By("Create second node in cluster")
			node2 := createNode(nodeName2, func(node *corev1.Node) {
				node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
				node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
			})
			defer deleteNode(node2)

			By("Verify that for second node a corresponding POD was created")
			verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName2), nodeName2,
				"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1)
		})

		It("Increase number of resources on node, expect that POD will be recreated with new resources", func() {
			node := createNode(nodeName1, func(node *corev1.Node) {
				node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
				node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
			})
			defer deleteNode(node)

			By("Create custom resource")
			Eventually(func() bool {
				err := k8sClient.Create(context.Background(),
					getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
				return err == nil
			}, timeout, interval).Should(BeTrue())
			defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
			defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

			By("Verify if POD was created")
			verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
				"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1)

			By("Update node resources")
			node.Status.Capacity[vfPoolName] = *resource.NewQuantity(2, resource.DecimalSI)
			err := k8sClient.Status().Update(context.Background(), node)
			Expect(err).Should(BeNil())

			By("Verify that POD resources has been updated")
			time.Sleep(interval)
			verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
				"flowconfig-daemon-sriov-cvl0-admin, flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 2)
		})

		It("Decrease number of resources on node, expect that POD will be recreated with new resources", func() {
			node := createNode(nodeName1, func(node *corev1.Node) {
				node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
				node.Status.Capacity[vfPoolName] = *resource.NewQuantity(2, resource.DecimalSI)
			})
			defer deleteNode(node)

			By("Create custom resource")
			Eventually(func() bool {
				err := k8sClient.Create(context.Background(),
					getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
				return err == nil
			}, timeout, interval).Should(BeTrue())
			defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

			By("Verify if POD was created")
			verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
				"flowconfig-daemon-sriov-cvl0-admin, flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 2)

			By("Update node resources - decrease to 1")
			node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
			err := k8sClient.Status().Update(context.Background(), node)
			Expect(err).Should(BeNil())

			By("Verify that POD resources has been updated")
			time.Sleep(interval)
			verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), nodeName1,
				"flowconfig-daemon-sriov-cvl0-admin", vfPoolName, 0, 1)

			By("Update node resources - decrease to 0")
			node.Status.Capacity[vfPoolName] = *resource.NewQuantity(0, resource.DecimalSI)
			err = k8sClient.Status().Update(context.Background(), node)
			Expect(err).Should(BeNil())

			By("Verify that POD was removed from node")
			obj := &flowconfigv1.FlowConfigNodeAgentDeployment{}
			Eventually(func() string {
				return GetObject(k8sClient, fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault, 1*time.Second, obj).Error()
			}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", fmt.Sprintf("flowconfig-daemon-%s", nodeName1))))
		})
	})

	Context("Different type of clusters", func() {
		setClusterTypeEnv := func(envValue string) {
			err := os.Setenv(clusterTypeEnvKey, envValue)
			Expect(err).NotTo(HaveOccurred())
			// Reload Pod template
			podTemp, err := nodeAgentDeploymentRc.getPodTemplate()
			Expect(err).Should(BeNil())
			nodeAgentDeploymentRc.flowConfigPod = podTemp
		}
		unsetClusterTypeEnv := func() {
			err := os.Unsetenv(clusterTypeEnvKey)
			Expect(err).NotTo(HaveOccurred())
			// Reload Pod template to default
			podTemp, err := nodeAgentDeploymentRc.getPodTemplate()
			Expect(err).Should(BeNil())
			nodeAgentDeploymentRc.flowConfigPod = podTemp
		}

		checkHostVolPath := func(envVal, hostDDPpath string) {
			setClusterTypeEnv(envVal)
			defer unsetClusterTypeEnv()

			node := createNode(nodeName1, func(node *corev1.Node) {
				node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
				node.Status.Capacity[vfPoolName] = *resource.NewQuantity(1, resource.DecimalSI)
			})
			defer deleteNode(node)

			Eventually(func() bool {
				err := k8sClient.Create(context.Background(),
					getFlowConfigNodeAgentDeployment(namespaceDefault, func(flow *flowconfigv1.FlowConfigNodeAgentDeployment) {
						flow.Spec.DCFVfPoolName = vfPoolName
						flow.Spec.NADAnnotation = "flowconfig-daemon-sriov-cvl0-admin"
					}))
				return err == nil
			}, timeout, interval).Should(BeTrue())
			defer deletePod(fmt.Sprintf("flowconfig-daemon-%s", nodeName1), namespaceDefault)
			defer deleteFlowConfigNodeAgentDeployment(namespaceDefault)

			verifyExpectedPODDefintion(namespaceDefault, fmt.Sprintf("flowconfig-daemon-%s", nodeName1),
				nodeName1,
				"flowconfig-daemon-sriov-cvl0-admin",
				vfPoolName, 0, 1, func(pod *corev1.Pod) {
					podVolumes := pod.Spec.Volumes
					ddpVolExists := false
					for _, v := range podVolumes {
						if v.VolumeSource.HostPath != nil && v.VolumeSource.HostPath.Path == hostDDPpath {
							ddpVolExists = true
						}
					}
					Expect(ddpVolExists).To(BeTrue())
				})
		}

		Context("On a vanilla K8s", func() {
			It("Should have correct DDP volume mount in Pod spec", func() {
				checkHostVolPath("true", k8sDdpUpdatePath)
			})
		})
		Context("On a OCP cluster", func() {
			It("Should have correct DDP volume mount in Pod spec", func() {
				checkHostVolPath("false", ocpDdpUpdatePath)
			})
		})
	})

	Context("Explicit function call", func() {
		It("getPodResources - without resources", func() {
			pod := createPod("some", namespaceDefault)
			Expect(nodeAgentDeploymentRc.getPodResources(pod, 0, corev1.ResourceName("utf"))).Should(Equal(int64(0)))
		})

		It("getPodResources - with resources but incorrect unit - milli", func() {
			pod := createPod("some", namespaceDefault, func(pod *corev1.Pod) {
				limits := corev1.ResourceList{}
				pod.Spec.Containers[0].Resources.Limits = limits
				pod.Spec.Containers[0].Resources.Limits["someRes"] = *resource.NewMilliQuantity(10, resource.BinarySI)
			})
			Expect(nodeAgentDeploymentRc.getPodResources(pod, 0, corev1.ResourceName("someRes"))).Should(Equal(int64(0)))
		})

		It("getPodTemplate() - missing file with POD template", func() {
			podTemplatePath, err := filepath.Abs(podTemplateFile)
			Expect(err).Should(BeNil())

			err = os.Rename(podTemplatePath, podTemplatePath+"_new")
			Expect(err).Should(BeNil())
			defer func() {
				podTemplatePath, err := filepath.Abs(podTemplateFile)
				Expect(err).Should(BeNil())
				err = os.Rename(podTemplatePath+"_new", podTemplatePath)
				Expect(err).Should(BeNil())
			}()

			pod, err := nodeAgentDeploymentRc.getPodTemplate()
			Expect(err).ShouldNot(BeNil())
			Expect(fmt.Sprint(err)).Should(ContainSubstring("error reading"))
			Expect(pod).Should(BeNil())
		})

		It("getPodTemplate() - POD template that does not define UFT container", func() {
			By("Missing fileModify file with POD template")
			podTemplatePath, err := filepath.Abs(podTemplateFile)
			Expect(err).Should(BeNil())

			input, err := ioutil.ReadFile(podTemplatePath)
			Expect(err).Should(BeNil())

			fileAsString := bytes.NewBuffer(input).String()
			fileAsString = strings.Replace(fileAsString, "name: uft", "name: external", 1)

			err = ioutil.WriteFile(podTemplatePath, []byte(fileAsString), 0644)
			Expect(err).Should(BeNil())

			defer func() {
				podTemplatePath, err := filepath.Abs(podTemplateFile)
				Expect(err).Should(BeNil())

				input, err := ioutil.ReadFile(podTemplatePath)
				Expect(err).Should(BeNil())

				fileAsString := bytes.NewBuffer(input).String()
				fileAsString = strings.Replace(fileAsString, "name: external", "name: uft", 1)

				err = ioutil.WriteFile(podTemplatePath, []byte(fileAsString), 0644)
				Expect(err).Should(BeNil())
			}()

			pod, err := nodeAgentDeploymentRc.getPodTemplate()
			Expect(err).ShouldNot(BeNil())
			Expect(fmt.Sprint(err)).Should(ContainSubstring("uft container not found in podSpec"))
			Expect(pod).Should(BeNil())
		})
	})
})
