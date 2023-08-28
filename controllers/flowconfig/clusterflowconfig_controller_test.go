// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2023 Intel Corporation

package flowconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	flowconfigv1 "github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/apis/flowconfig/v1"
	flowapi "github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	timeout  = 4 * time.Second
	interval = 250 * time.Millisecond
)

func WaitForObjectCreation(core client.Client, objectName, ns string, timeout, interval time.Duration, object client.Object) error {
	return wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		err = GetObject(core, objectName, ns, timeout, object)
		if err != nil {
			if strings.Contains(err.Error(), fmt.Sprintf("\"%s\" not found", objectName)) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func GetObject(core client.Client, objectName, ns string, timeout time.Duration, object client.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := core.Get(ctx, client.ObjectKey{
		Namespace: ns,
		Name:      objectName,
	}, object)

	return err
}

var _ = Describe("Cluster Flow Config Controller tests", func() {
	const (
		podNameDefault   = "pod-default"
		namespaceDefault = "default"
	)

	createRawExtension := func(interfaceName string) *runtime.RawExtension {
		nameConfig := &flowconfigv1.ToPodInterfaceConf{NetInterfaceName: interfaceName}
		rawBytes, err := json.Marshal(nameConfig)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		return &runtime.RawExtension{Raw: rawBytes}
	}

	createClusterFlowAction := func(types []flowconfigv1.ClusterFlowActionType) []*flowconfigv1.ClusterFlowAction {
		actions := make([]*flowconfigv1.ClusterFlowAction, 0)
		for _, actType := range types {
			actions = append(actions, &flowconfigv1.ClusterFlowAction{
				Type: actType,
			})
		}

		return actions
	}

	getClusterFlowConfig := func(configurers ...func(flowConfig *flowconfigv1.ClusterFlowConfig)) *flowconfigv1.ClusterFlowConfig {
		obj := &flowconfigv1.ClusterFlowConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "sriov.intel.com/v1",
				Kind:       "ClusterFlowConfig",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "",
			},
			Spec: flowconfigv1.ClusterFlowConfigSpec{},
		}

		for _, config := range configurers {
			config(obj)
		}

		return obj
	}

	deleteClusterFlowConfig := func(name, ns string) {
		clusterFlowConfig := &flowconfigv1.ClusterFlowConfig{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
		}

		Expect(k8sClient.Delete(context.Background(), clusterFlowConfig)).Should(BeNil())
		Eventually(func() string {
			return GetObject(k8sClient, clusterFlowConfig.ObjectMeta.Name, clusterFlowConfig.ObjectMeta.Namespace,
				1*time.Second, &flowconfigv1.ClusterFlowConfig{}).Error()
		}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", clusterFlowConfig.ObjectMeta.Name)))
	}

	addPodAnnotations := func(pod *corev1.Pod) {
		pod.ObjectMeta.Annotations = make(map[string]string)
		pod.ObjectMeta.Annotations["k8s.v1.cni.cncf.io/network-status"] = `[
{
	"name": "sriov-network_a",
	"interface": "net0",
	"device-info": {
		"type": "pci",
		"version": "1.0.0",
		"pci": {
			"pci-address": "0000:18:02.5",
			"pf-pci-address": "0000:18:00.0"
		}
	}
}]`
	}

	getClusterFlowRules := func() []*flowconfigv1.ClusterFlowRule {
		return []*flowconfigv1.ClusterFlowRule{
			{
				Pattern: []*flowconfigv1.FlowItem{
					{
						Type: "RTE_FLOW_ITEM_TYPE_ETH",
					},
					{
						Type: "RTE_FLOW_ITEM_TYPE_IPV4",
						Spec: &runtime.RawExtension{
							Raw: []byte(`{ "hdr": { "src_addr": "10.56.217.9" } }`),
						},
						Mask: &runtime.RawExtension{
							Raw: []byte(`{ "hdr": { "src_addr": "255.255.255.255" } }`),
						},
					},
					{
						Type: "RTE_FLOW_ITEM_TYPE_END",
					},
				},
				Action: []*flowconfigv1.ClusterFlowAction{
					{
						Type: flowconfigv1.ToPodInterface,
						Conf: &runtime.RawExtension{
							Raw: []byte(`{ "podInterface": "net0" }`),
						},
					},
				},
				Attr: &flowconfigv1.FlowAttr{
					Ingress: 1,
				},
			},
		}
	}

	compareNodeFlowConfigRule := func(object *flowconfigv1.NodeFlowConfig, name string, patternTypes []string, rulesSize int) error {
		if object.ObjectMeta.Name != name {
			return fmt.Errorf("Invalid name: %s", object.ObjectMeta.Name)
		}

		if len(object.Spec.Rules) != rulesSize {
			return fmt.Errorf("Invalid Rules size %d expected %d", len(object.Spec.Rules), rulesSize)
		}

		found := false
		for _, r := range object.Spec.Rules {
			if len(patternTypes) != len(r.Pattern) {
				continue
			}

			for i := 0; i < len(patternTypes); i++ {
				if patternTypes[i] != r.Pattern[i].Type {
					break
				}
				found = true
			}

			if found {
				break
			}
		}

		if !found {
			return fmt.Errorf("NodeFlowConfig %s does not contain patterns %v", object.ObjectMeta.Name, patternTypes)
		}

		return nil
	}

	Context("Verify ClusterFlowConfig reconcile loop", func() {
		const (
			NODE_NAME_1 = "node-worker-1"
			NODE_NAME_2 = "node-worker-2"
		)

		It("Verify that cluster does not have nodes", func() {
			nodeList := &corev1.NodeList{}
			Expect(k8sClient.List(context.Background(), nodeList)).To(BeNil())
			Expect(len(nodeList.Items)).Should(Equal(0))
		})

		When("Expecting that NodeFlowConfig CR will not be created", func() {
			It("Missing ClusterFlowConfig", func() {
				result, err := clusterFlowConfigRc.Reconcile(context.TODO(), ctrl.Request{})
				Expect(result).Should(Equal(ctrl.Result{}))
				Expect(err).To(BeNil())
			})

			It("Add ClusterFlowConfig with empty specification, POD is missing", func() {
				node := createNode(NODE_NAME_1, func(node *corev1.Node) {
					node.Status.Capacity = make(map[corev1.ResourceName]resource.Quantity)
				})
				defer deleteNode(node)

				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)

				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				Eventually(func() string {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, &flowconfigv1.NodeFlowConfig{}).Error()
				}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", NODE_NAME_1)))
			})

			It("Add ClusterFlowConfig with POD selectors, without flow rules, POD is misssing", func() {
				node := createNode(NODE_NAME_1)
				defer deleteNode(node)

				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)

				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				Eventually(func() string {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, &flowconfigv1.NodeFlowConfig{}).Error()
				}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", NODE_NAME_1)))
			})

			It("Add ClusterFlowConfig with POD selectors, without flow rules and node name, POD exists", func() {
				node := createNode(NODE_NAME_1)
				defer deleteNode(node)

				pod := createPod("test-pod", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"testKey": "testName"}
				})
				defer deletePod(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)

				Expect(k8sClient.Create(context.TODO(), pod)).ToNot(HaveOccurred())

				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)

				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				Eventually(func() string {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, &flowconfigv1.NodeFlowConfig{}).Error()
				}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", NODE_NAME_1)))
			})

			It("Add ClusterFlowConfig with POD selectors, with flow rules and node name, POD without network status", func() {
				node := createNode(NODE_NAME_1)
				defer deleteNode(node)

				pod := createPod("test-pod", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"testKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_1
					pod.ObjectMeta.Annotations = make(map[string]string)
				})
				defer deletePod(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), pod)).ToNot(HaveOccurred())

				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				Eventually(func() string {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, &flowconfigv1.NodeFlowConfig{}).Error()
				}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", NODE_NAME_1)))
			})

			It("Add ClusterFlowConfig with POD selector, POD has completely different set of labels", func() {
				node := createNode(NODE_NAME_1)
				defer deleteNode(node)

				pod := createPod("test-pod", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"label1": "val", "label2": "val"}
					pod.Spec.NodeName = NODE_NAME_1
					addPodAnnotations(pod)
				})
				defer deletePod(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), pod)).ToNot(HaveOccurred())

				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"crKey": "val"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				Eventually(func() string {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, &flowconfigv1.NodeFlowConfig{}).Error()
				}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", NODE_NAME_1)))
			})

			It("Add ClusterFlowConfig with POD selector, POD and CR have one common out of three labels, CR have second label that does not occurs in POD", func() {
				node := createNode(NODE_NAME_1)
				defer deleteNode(node)

				pod := createPod("test-pod", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"label1": "val", "label2": "val", "label3": "val"}
					pod.Spec.NodeName = NODE_NAME_1
					addPodAnnotations(pod)
				})
				defer deletePod(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), pod)).ToNot(HaveOccurred())

				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"label1": "val", "testKey": "val"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				Eventually(func() string {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, &flowconfigv1.NodeFlowConfig{}).Error()
				}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", NODE_NAME_1)))
			})
		})

		When("Expecting that NodeFlowConfig CR will be created", func() {
			var node *corev1.Node
			var pod *corev1.Pod

			BeforeEach(func() {
				node = createNode(NODE_NAME_1)

				pod = createPod("test-pod", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"testKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_1
					addPodAnnotations(pod)
				})
				Expect(k8sClient.Create(context.TODO(), pod)).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				deletePod(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
				pod = &corev1.Pod{}

				deleteNode(node)
				node = &corev1.Node{}
			})

			It("Add ClusterFlowConfig with POD selectors, without flow rules, with node name", func() {
				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				object := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())

				Expect(object.ObjectMeta.Name).To(Equal(NODE_NAME_1))
				Expect(object.Spec).To(Equal(flowconfigv1.NodeFlowConfigSpec{}))

				By("Delete NodeFlowConfig created by the ClusterFlowConfig controller")
				Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())
			})

			It("Add ClusterFlowConfig with POD selectors, with flow rules and node name", func() {
				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				object := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())

				Expect(compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())

				By("Delete NodeFlowConfig created by the ClusterFlowConfig controller")
				Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())
			})

			It("Add ClusterFlowConfig with POD selector, POD and CR have two common out of three labels", func() {
				pod2 := createPod("test-pod-1", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"label1": "val", "label2": "val", "label3": "val"}
					pod.Spec.NodeName = NODE_NAME_1
					addPodAnnotations(pod)
				})
				defer deletePod(pod2.ObjectMeta.Name, pod2.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), pod2)).ToNot(HaveOccurred())

				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"label1": "val", "label2": "val"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				object := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())

				Expect(compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())

				By("Delete NodeFlowConfig created by the ClusterFlowConfig controller")
				Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())
			})

			It("One ClusterFlowConfig CR with correct data, two nodes, only one have matched POD, expected create NodeFlowConfig on one", func() {
				node2 := createNode(NODE_NAME_2)
				defer deleteNode(node2)

				By("Delete POD that is deployed on first node and deploy POD on second node")
				deletePod(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)

				pod = createPod("test-pod", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"testKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_2
					addPodAnnotations(pod)
				})
				Expect(k8sClient.Create(context.TODO(), pod)).ToNot(HaveOccurred())

				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				object := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_2, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())
				Expect(compareNodeFlowConfigRule(object, NODE_NAME_2, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())

				defer func() {
					By("Delete NodeFlowConfig created by the ClusterFlowConfig controller")
					Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())
				}()

				object2 := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object2)
				}, timeout, interval).ShouldNot(BeNil())
			})

			It("One ClusterFlowConfig CR with correct data, two nodes, on both POD with correct labels, creates NodeFlowConfig on both", func() {
				node2 := createNode(NODE_NAME_2)
				defer deleteNode(node2)

				pod2 := createPod("test-pod-2", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"testKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_2
					addPodAnnotations(pod)
				})
				defer deletePod(pod2.ObjectMeta.Name, pod2.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), pod2)).ToNot(HaveOccurred())

				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				object := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())

				Expect(compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())

				defer func() {
					By("Delete NodeFlowConfig created by the ClusterFlowConfig controller")
					Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())
				}()

				object2 := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_2, "default", 1*time.Second, object2)
				}, timeout, interval).Should(BeNil())

				Expect(compareNodeFlowConfigRule(object2, NODE_NAME_2, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())

				defer func() {
					By("Delete NodeFlowConfig created by the ClusterFlowConfig controller")
					Expect(k8sClient.Delete(context.Background(), object2)).To(BeNil())
				}()
			})

			It("Two ClusterFlowConfig CRs, different name, the same set of rules, expected not duplicated rules in NodeFlowConfig", func() {
				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				object := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())
				defer func() {
					By("Delete NodeFlowConfig created by the ClusterFlowConfig controller")
					Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())
				}()

				Expect(compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())

				clusterConfig2 := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Name = "other-name"
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer deleteClusterFlowConfig(clusterConfig2.ObjectMeta.Name, clusterConfig2.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig2)).ToNot(HaveOccurred())
				Eventually(func() error {
					err := GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
					if err != nil {
						return err
					}
					return compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)

				}, timeout, interval).Should(BeNil())
			})

			It("Two ClusterFlowConfig CR with correct data, two nodes, on both POD with correct labels, creates NodeFlowConfig on both with correct set of rules", func() {
				node2 := createNode(NODE_NAME_2)
				defer deleteNode(node2)

				pod2 := createPod("test-pod-2", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"secondKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_2
					addPodAnnotations(pod)
				})
				defer deletePod(pod2.ObjectMeta.Name, pod2.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), pod2)).ToNot(HaveOccurred())

				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				clusterConfig2 := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Name = "other-rules-definition"
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"secondKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
					flowConfig.Spec.Rules[0].Pattern[0].Type = "RTE_FLOW_ITEM_TYPE_VLAN"
					flowConfig.Spec.Rules[0].Pattern = append(flowConfig.Spec.Rules[0].Pattern, &flowconfigv1.FlowItem{Type: "RTE_FLOW_ITEM_TYPE_VLAN"})
				})
				defer deleteClusterFlowConfig(clusterConfig2.ObjectMeta.Name, clusterConfig2.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig2)).ToNot(HaveOccurred())

				object := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())
				defer func() {
					By("Delete NodeFlowConfig created by the ClusterFlowConfig controller")
					Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())
				}()

				Expect(compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())

				object2 := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_2, "default", 1*time.Second, object2)
				}, timeout, interval).Should(BeNil())
				defer func() {
					By("Delete NodeFlowConfig created by the ClusterFlowConfig controller")
					Expect(k8sClient.Delete(context.Background(), object2)).To(BeNil())
				}()

				Expect(compareNodeFlowConfigRule(object2, NODE_NAME_2, []string{"RTE_FLOW_ITEM_TYPE_VLAN", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END", "RTE_FLOW_ITEM_TYPE_VLAN"}, 1)).To(BeNil())

			})

			It("Two ClusterFlowConfig with different set of labels, one node, two PODs with different set of labels matched to CRs, NodeFlowConfig is overwritten", func() {
				pod2 := createPod("test-pod-2", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"znotherKey": "anotherName"}
					pod.Spec.NodeName = NODE_NAME_1
					addPodAnnotations(pod)
				})
				defer deletePod(pod2.ObjectMeta.Name, pod2.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), pod2)).ToNot(HaveOccurred())

				clusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				object := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())
				defer func() {
					By("Delete NodeFlowConfig created by the ClusterFlowConfig controller")
					Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())
				}()
				Expect(compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())

				clusterConfig2 := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Name = "other-rules-definition"
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"znotherKey": "anotherName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
					flowConfig.Spec.Rules[0].Pattern[0].Type = "RTE_FLOW_ITEM_TYPE_VLAN"
					flowConfig.Spec.Rules[0].Pattern = append(flowConfig.Spec.Rules[0].Pattern, &flowconfigv1.FlowItem{Type: "RTE_FLOW_ITEM_TYPE_VLAN"})
				})
				defer deleteClusterFlowConfig(clusterConfig2.ObjectMeta.Name, clusterConfig2.ObjectMeta.Namespace)
				Expect(k8sClient.Create(context.TODO(), clusterConfig2)).ToNot(HaveOccurred())

				By("Give some time to update NodeFlowConfigs and check results")
				object2 := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					err := GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object2)
					if err != nil {
						return err
					}
					return compareNodeFlowConfigRule(object2, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_VLAN", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END", "RTE_FLOW_ITEM_TYPE_VLAN"}, 1)
				}, timeout, interval).Should(BeNil())
			})
		})

		When("Expecting that NodeFlowConfig CR will be updated", func() {
			var node *corev1.Node
			var pod *corev1.Pod
			var clusterConfig *flowconfigv1.ClusterFlowConfig
			var object *flowconfigv1.NodeFlowConfig

			BeforeEach(func() {
				node = createNode(NODE_NAME_1)
				pod = createPod("test-pod", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"testKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_1
					addPodAnnotations(pod)
				})
				err := deployPodAndUpdatePhase(pod, corev1.PodRunning, 20, 1)
				Expect(err).ToNot(HaveOccurred())

				clusterConfig = getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())

				object = &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())

				Expect(compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())
			})

			AfterEach(func() {
				deleteNode(node)
				deletePod(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
				deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)

				clusterFlowConfig := &flowconfigv1.ClusterFlowConfig{}
				Eventually(func() string {
					return GetObject(k8sClient, clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace, 1*time.Second, clusterFlowConfig).Error()
				}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", clusterConfig.ObjectMeta.Name)))

				By("Delete NodeFlowConfig created by the ClusterFlowConfig controller")
				Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())

				Eventually(func() string {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object).Error()
				}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", NODE_NAME_1)))
			})

			It("Update existing ClusterFlowConfig by adding new rules, POD selectors stays the same", func() {
				clusterConfig.Spec.Rules[0].Pattern[0].Type = "RTE_FLOW_ITEM_TYPE_VLAN"
				newPattern := &flowconfigv1.FlowItem{Type: "RTE_FLOW_ITEM_TYPE_END"}
				clusterConfig.Spec.Rules[0].Pattern = append(clusterConfig.Spec.Rules[0].Pattern, newPattern)
				Expect(k8sClient.Update(context.Background(), clusterConfig)).To(BeNil())

				By("Update object NodeFlowConfig")
				Eventually(func() error {
					if err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.Namespace,
						Name:      NODE_NAME_1,
					}, object); err != nil {
						return err
					}

					return compareNodeFlowConfigRule(object, NODE_NAME_1,
						[]string{"RTE_FLOW_ITEM_TYPE_VLAN", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END", "RTE_FLOW_ITEM_TYPE_END"}, 1)
				}, timeout, interval).Should(BeNil())
			})

			It("Create two ClusterFlowConfig CRs that targets the same POD spec, NodeFlowConfig expected to be created, rules merged", func() {
				secondClusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Name = "second-cr"
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
					flowConfig.Spec.Rules[0].Pattern[0].Type = "RTE_FLOW_ITEM_TYPE_VLAN"
				})

				Expect(k8sClient.Create(context.TODO(), secondClusterConfig)).ToNot(HaveOccurred())
				defer deleteClusterFlowConfig(secondClusterConfig.ObjectMeta.Name, secondClusterConfig.ObjectMeta.Namespace)

				Eventually(func() error {
					if err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.Namespace,
						Name:      NODE_NAME_1,
					}, object); err != nil {
						return err
					}

					if err := compareNodeFlowConfigRule(object, NODE_NAME_1,
						[]string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 2); err != nil {
						return err
					}

					if err := compareNodeFlowConfigRule(object, NODE_NAME_1,
						[]string{"RTE_FLOW_ITEM_TYPE_VLAN", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 2); err != nil {
						return err
					}

					return nil
				}, timeout, interval).Should(BeNil())
			})

			It("On two nodes create two different PODs and CRs. For each node different NodeFlowConfig is expected", func() {
				node2 := createNode(NODE_NAME_2)
				defer deleteNode(node2)

				pod2 := createPod("test-pod-2", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"anotherKey": "someValue"}
					pod.Spec.NodeName = NODE_NAME_2
					addPodAnnotations(pod)
				})
				err := deployPodAndUpdatePhase(pod2, corev1.PodRunning, 20, 1)
				Expect(err).ToNot(HaveOccurred())
				defer deletePod(pod2.ObjectMeta.Name, pod2.ObjectMeta.Namespace)

				object2 := &flowconfigv1.NodeFlowConfig{}

				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_2, "default", 1*time.Second, object2)
				}, timeout, interval).ShouldNot(BeNil())

				defer func() {
					_ = k8sClient.Delete(context.Background(), object2)
					Eventually(func() string {
						return GetObject(k8sClient, NODE_NAME_2, "default", 1*time.Second, &flowconfigv1.NodeFlowConfig{}).Error()
					}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", NODE_NAME_2)))
				}()

				Expect(object2.ObjectMeta.Name).To(Equal(""))
				Expect(len(object2.Spec.Rules)).To(Equal(0))

				secondClusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Name = "second-cr"
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"anotherKey": "someValue"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
					flowConfig.Spec.Rules[0].Pattern[0].Type = "RTE_FLOW_ITEM_TYPE_VLAN"
				})
				Expect(k8sClient.Create(context.TODO(), secondClusterConfig)).ToNot(HaveOccurred())
				defer deleteClusterFlowConfig(secondClusterConfig.ObjectMeta.Name, secondClusterConfig.ObjectMeta.Namespace)

				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_2, "default", 1*time.Second, object2)
				}, timeout, interval).Should(BeNil())

				By("Check NodeFlowConfig on second worker node")
				Eventually(func() error {
					if err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object2.Namespace,
						Name:      NODE_NAME_2,
					}, object2); err != nil {
						return err
					}
					return compareNodeFlowConfigRule(object2, NODE_NAME_2, []string{"RTE_FLOW_ITEM_TYPE_VLAN", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)
				}, timeout, interval).Should(BeNil())

				By("Check NodeFlowConfig on first worker node")
				Eventually(func() error {
					if err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.Namespace,
						Name:      NODE_NAME_1,
					}, object); err != nil {
						return err
					}
					return compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)
				}, timeout, interval).Should(BeNil())

				By("Create third ClusterFlowConfig, expect only one NodeFlowConfig to be updated")
				thirdClusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Name = "third-cr"
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
					flowConfig.Spec.Rules[0].Pattern[0].Type = "RTE_FLOW_ITEM_TYPE_VLAN"
				})
				Expect(k8sClient.Create(context.TODO(), thirdClusterConfig)).ToNot(HaveOccurred())
				defer deleteClusterFlowConfig(thirdClusterConfig.ObjectMeta.Name, thirdClusterConfig.ObjectMeta.Namespace)

				By("Check NodeFlowConfig on second worker node")
				Eventually(func() error {
					if err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object2.Namespace,
						Name:      NODE_NAME_2,
					}, object2); err != nil {
						return err
					}
					return compareNodeFlowConfigRule(object2, NODE_NAME_2, []string{"RTE_FLOW_ITEM_TYPE_VLAN", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)
				}, timeout, interval).Should(BeNil())

				By("Check NodeFlowConfig on first worker node")
				Eventually(func() error {
					if err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.Namespace,
						Name:      NODE_NAME_1,
					}, object); err != nil {
						return err
					}

					if err := compareNodeFlowConfigRule(object, NODE_NAME_1,
						[]string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 2); err != nil {
						return err
					}

					if err := compareNodeFlowConfigRule(object, NODE_NAME_1,
						[]string{"RTE_FLOW_ITEM_TYPE_VLAN", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 2); err != nil {
						return err
					}

					return nil
				}, timeout, interval).Should(BeNil())
			})
		})

		When("ClusterFlowConfig CR is deleted", func() {
			var node *corev1.Node
			var pod *corev1.Pod
			var clusterConfig *flowconfigv1.ClusterFlowConfig
			var object *flowconfigv1.NodeFlowConfig

			BeforeEach(func() {
				node = createNode(NODE_NAME_1)

				pod = createPod("test-pod", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"testKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_1
					addPodAnnotations(pod)
				})
				Expect(k8sClient.Create(context.TODO(), pod)).ToNot(HaveOccurred())

				clusterConfig = getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())
				object = &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())

				Expect(compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())
			})

			AfterEach(func() {
				deleteNode(node)
				deletePod(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)

				By("Delete NodeFlowConfig created by the ClusterFlowConfig controller to avoid clashes with other tests")
				Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())
			})

			It("Delete ClusterFlowConfig CR, expect all rules from NodeFlowConfig to be deleted too", func() {
				deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Eventually(func() bool {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: clusterConfig.ObjectMeta.Namespace,
						Name:      NODE_NAME_1,
					}, object)
					fmt.Println(err)
					if err != nil {
						return false
					}

					if len(object.Spec.Rules) != 0 {
						fmt.Println("Object rules were not cleared")
						return false
					}

					return true
				}, timeout, interval).Should(BeTrue())

				Eventually(func() int {
					return len(clusterFlowConfigRc.Cluster2NodeRulesHashMap)
				}, timeout, interval).Should(Equal(0))
			})

			It("Add second ClusterFlowConfig, delete first one, and expect only part of rules to be removed", func() {
				secondClusterConfig := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Name = "second-cr"
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
					flowConfig.Spec.Rules[0].Pattern[0].Type = "RTE_FLOW_ITEM_TYPE_VLAN"
					newPattern := &flowconfigv1.FlowItem{Type: "RTE_FLOW_ITEM_TYPE_END"}
					flowConfig.Spec.Rules[0].Pattern = append(flowConfig.Spec.Rules[0].Pattern, newPattern)
				})
				Expect(k8sClient.Create(context.TODO(), secondClusterConfig)).ToNot(HaveOccurred())
				defer deleteClusterFlowConfig(secondClusterConfig.ObjectMeta.Name, secondClusterConfig.ObjectMeta.Namespace)

				deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				Eventually(func() bool {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.Namespace,
						Name:      object.Name,
					}, object)
					fmt.Println(err)
					if err != nil {
						return false
					}

					if len(object.Spec.Rules) == 1 {
						return true
					}

					fmt.Println("Invalid length", len(object.Spec.Rules))

					return false
				}, timeout, interval).Should(BeTrue())

				Eventually(func() int {
					return len(clusterFlowConfigRc.Cluster2NodeRulesHashMap)
				}, timeout, interval).Should(Equal(1))

				Expect(len(object.Spec.Rules[0].Pattern)).To(Equal(4))
			})

			It("Add second worker node, wait for NodeFlowConfig creation, delete ClusterFlowConfig, all rules should be removed from both NodeFlowConfig", func() {
				node2 := createNode(NODE_NAME_2)
				defer deleteNode(node2)

				pod2 := createPod("test-pod-2", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"testKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_2
					addPodAnnotations(pod)
				})
				err := deployPodAndUpdatePhase(pod2, corev1.PodRunning, 20, 1)
				Expect(err).ToNot(HaveOccurred())
				defer deletePod(pod2.ObjectMeta.Name, pod2.ObjectMeta.Namespace)

				object2 := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_2, "default", 1*time.Second, object2)
				}, timeout, interval).Should(BeNil())
				defer func() {
					_ = k8sClient.Delete(context.Background(), object2)
				}()

				Expect(compareNodeFlowConfigRule(object2, NODE_NAME_2, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())

				deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)

				Eventually(func() bool {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.Namespace,
						Name:      object.Name,
					}, object)
					fmt.Println(err)
					if err != nil {
						return false
					}

					if len(object.Spec.Rules) != 0 {
						fmt.Println("Object rules were not cleared")
						return false
					}

					return true
				}, timeout, interval).Should(BeTrue())

				Eventually(func() bool {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object2.Namespace,
						Name:      object2.Name,
					}, object2)
					fmt.Println(err)
					if err != nil {
						return false
					}

					if len(object2.Spec.Rules) != 0 {
						fmt.Println("Object2 rules were not cleared")
						return false
					}

					return true
				}, timeout, interval).Should(BeTrue())

				Eventually(func() int {
					return len(clusterFlowConfigRc.Cluster2NodeRulesHashMap)
				}, timeout, interval).Should(Equal(0))
			})

			PIt("Add second ClusterFlowConfig, different name, the same set of rules, delete second CR, expected duplicated rules to be not removed from NodeFlowConfig", func() {
				clusterConfig2 := getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Name = "other-name"
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				defer func() {
					if clusterConfig2 != nil {
						deleteClusterFlowConfig(clusterConfig2.ObjectMeta.Name, clusterConfig2.ObjectMeta.Namespace)
					}
				}()
				Expect(k8sClient.Create(context.TODO(), clusterConfig2)).ToNot(HaveOccurred())

				Consistently(func() error {
					return compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)
				}, "35s", "8s").Should(BeNil())

				By("Delete second ClusterFlowConfig")
				deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)

				Consistently(func() error {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.ObjectMeta.Namespace,
						Name:      object.ObjectMeta.Name,
					}, object)
					if err != nil {
						return err
					}
					return compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)
				}, "65s", "9s").Should(BeNil())
			})
		})

		When("New node is added to cluster", func() {
			var node, node2 *corev1.Node
			var pod *corev1.Pod
			var clusterConfig *flowconfigv1.ClusterFlowConfig
			var object *flowconfigv1.NodeFlowConfig

			BeforeEach(func() {
				node = createNode(NODE_NAME_1)
				node2 = createNode(NODE_NAME_2)

				pod = createPod("test-pod", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"testKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_1
					addPodAnnotations(pod)
				})
				Expect(k8sClient.Create(context.TODO(), pod)).ToNot(HaveOccurred())

				clusterConfig = getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())
				object = &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())

				Expect(compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())
			})

			AfterEach(func() {
				deleteNode(node)
				deleteNode(node2)
				deletePod(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
				deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)

				By("Delete NodeFlowConfig created by the ClusterFlowConfig controller to avoid clashes with other tests")
				Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())
			})

			It("without POD that matches ClusterFlowConfig CR, expected to not create NodeFlowConfig", func() {
				pod2 := createPod("test-pod-2", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"randomKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_2
					addPodAnnotations(pod)
				})
				err := deployPodAndUpdatePhase(pod2, corev1.PodRunning, 20, 1)
				Expect(err).ToNot(HaveOccurred())
				defer deletePod(pod2.ObjectMeta.Name, pod2.ObjectMeta.Namespace)

				object2 := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() string {
					return GetObject(k8sClient, NODE_NAME_2, "default", 1*time.Second, object2).Error()
				}, timeout, interval).Should(ContainSubstring(fmt.Sprintf("\"%s\" not found", NODE_NAME_2)))
			})

			It("with POD that matches ClusterFlowConfig CR, expected to create NodeFlowConfig", func() {
				pod2 := createPod("test-pod-2", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"testKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_2
					addPodAnnotations(pod)
				})
				err := deployPodAndUpdatePhase(pod2, corev1.PodRunning, 20, 1)
				Expect(err).ToNot(HaveOccurred())
				defer deletePod(pod2.ObjectMeta.Name, pod2.ObjectMeta.Namespace)

				object2 := &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_2, "default", 1*time.Second, object2)
				}, timeout, interval).Should(BeNil())

				Expect(compareNodeFlowConfigRule(object2, NODE_NAME_2, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())
				By("Delete NodeFlowConfig created by the ClusterFlowConfig controller to avoid clashes with other tests")
				Expect(k8sClient.Delete(context.Background(), object2)).To(BeNil())
			})
		})

		When("POD is created/deleted on worker node that already have NodeFlowConfig instance, expect that NodeFlowConfig will be updated", func() {
			var node *corev1.Node
			var pod *corev1.Pod
			var clusterConfig, clusterConfig2 *flowconfigv1.ClusterFlowConfig
			var object *flowconfigv1.NodeFlowConfig

			BeforeEach(func() {
				node = createNode(NODE_NAME_1)

				clusterConfig2 = getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.ObjectMeta.Name = "another-cr"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"otherKey": "testName",
						},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
					flowConfig.Spec.Rules[0].Pattern[0].Type = "RTE_FLOW_ITEM_TYPE_VLAN"
					newPattern := &flowconfigv1.FlowItem{Type: "RTE_FLOW_ITEM_TYPE_END"}
					flowConfig.Spec.Rules[0].Pattern = append(flowConfig.Spec.Rules[0].Pattern, newPattern)
				})
				Expect(k8sClient.Create(context.TODO(), clusterConfig2)).ToNot(HaveOccurred())

				pod = createPod("test-pod", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"testKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_1
					addPodAnnotations(pod)
				})
				err := deployPodAndUpdatePhase(pod, corev1.PodRunning, 20, 1)
				Expect(err).ToNot(HaveOccurred())

				clusterConfig = getClusterFlowConfig(func(flowConfig *flowconfigv1.ClusterFlowConfig) {
					flowConfig.ObjectMeta.Namespace = "default"
					flowConfig.Spec.PodSelector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testKey": "testName"},
					}
					flowConfig.Spec.Rules = getClusterFlowRules()
				})
				Expect(k8sClient.Create(context.TODO(), clusterConfig)).ToNot(HaveOccurred())
				object = &flowconfigv1.NodeFlowConfig{}
				Eventually(func() error {
					return GetObject(k8sClient, NODE_NAME_1, "default", 1*time.Second, object)
				}, timeout, interval).Should(BeNil())

				Expect(compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)).To(BeNil())
			})

			AfterEach(func() {
				deleteNode(node)
				deletePod(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
				deleteClusterFlowConfig(clusterConfig.ObjectMeta.Name, clusterConfig.ObjectMeta.Namespace)
				deleteClusterFlowConfig(clusterConfig2.ObjectMeta.Name, clusterConfig2.ObjectMeta.Namespace)

				By("Delete NodeFlowConfig created by the ClusterFlowConfig controller to avoid clashes with other tests")
				Expect(k8sClient.Delete(context.Background(), object)).To(BeNil())
			})

			It("Create POD with labels that matches second ClusterFlowConfig, expect NodeFlowConfig will be updated with new rules, old rules are removed", func() {
				pod2 := createPod("test-pod-2", "default", func(pod *corev1.Pod) {
					pod.ObjectMeta.Labels = map[string]string{"otherKey": "testName"}
					pod.Spec.NodeName = NODE_NAME_1
					addPodAnnotations(pod)
				})
				err := deployPodAndUpdatePhase(pod2, corev1.PodRunning, 20, 1)
				Expect(err).ToNot(HaveOccurred())
				defer deletePod(pod2.ObjectMeta.Name, pod2.ObjectMeta.Namespace)

				Eventually(func() error {
					if err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.Namespace,
						Name:      object.Name,
					}, object); err != nil {
						return err
					}
					return compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_VLAN", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END", "RTE_FLOW_ITEM_TYPE_END"}, 1)
				}, timeout, interval).Should(BeNil())
			})

			It("Delete POD with matching CR labels, expected to remove rules from NodeFlowConfig", func() {
				deletePod(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)

				Eventually(func() bool {
					if err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.Namespace,
						Name:      object.Name,
					}, object); err != nil {
						fmt.Println("Error", err)
						return false
					}

					return len(object.Spec.Rules) == 0
				}, timeout, interval).Should(BeTrue())

				// to avoid error recreate POD that will be removed in AfterEach function
				pod = createPod("test-pod", "default")
				Expect(k8sClient.Create(context.TODO(), pod)).ToNot(HaveOccurred())
			})

			It("Update labels in POD, so they do not match any ClusterFlowConfig CR, expect NodeFlowConfig to be cleared", func() {
				pod.ObjectMeta.Labels["testKey"] = "unexpectedValue"
				Expect(k8sClient.Update(context.Background(), pod)).To(BeNil())
				Eventually(func() bool {
					if err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.Namespace,
						Name:      object.Name,
					}, object); err != nil {
						fmt.Println("Error", err)
						return false
					}

					return len(object.Spec.Rules) == 0
				}, timeout, interval).Should(BeTrue())
			})

			It("Update labels in POD, so they do match the second ClusterFlowConfig CR, expect NodeFlowConfig to be updated with new rules", func() {
				pod.ObjectMeta.Labels["otherKey"] = "testName"
				Expect(k8sClient.Update(context.Background(), pod)).To(BeNil())
				Eventually(func() error {
					if err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.Namespace,
						Name:      object.Name,
					}, object); err != nil {
						return err
					}

					return compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_VLAN", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END", "RTE_FLOW_ITEM_TYPE_END"}, 1)
				}, timeout, interval).Should(BeNil())
			})

			It("Update POD annotations, NodeFlowConfig should not be affected, actually Reconcile loop should not be called at all", func() {
				addPodAnnotations(pod)
				Eventually(func() error {
					if err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Namespace: object.Namespace,
						Name:      object.Name,
					}, object); err != nil {
						return err
					}

					return compareNodeFlowConfigRule(object, NODE_NAME_1, []string{"RTE_FLOW_ITEM_TYPE_ETH", "RTE_FLOW_ITEM_TYPE_IPV4", "RTE_FLOW_ITEM_TYPE_END"}, 1)
				}, timeout, interval).Should(BeNil())
			})
		})
	})

	Context("Verify getNodeActionsFromClusterActions()", func() {
		DescribeTable("Expect results", func(actions []*flowconfigv1.ClusterFlowAction, pod *corev1.Pod, expectedActions []*flowconfigv1.FlowAction) {
			retAction, err := clusterFlowConfigRc.getNodeActionsFromClusterActions(actions, pod)
			Expect(expectedActions).Should(Equal(retAction))
			Expect(err).Should(BeNil())
		},
			Entry("nil input", nil, nil, []*flowconfigv1.FlowAction{}),
			Entry("empty input", []*flowconfigv1.ClusterFlowAction{}, nil, []*flowconfigv1.FlowAction{}),
			Entry("input one action without end, output action with one end action",
				createClusterFlowAction([]flowconfigv1.ClusterFlowActionType{flowconfigv1.ClusterFlowActionType(flowapi.RteFlowActionType_RTE_FLOW_ACTION_TYPE_DROP)}),
				nil,
				[]*flowconfigv1.FlowAction{
					{Type: flowapi.RteFlowActionType_RTE_FLOW_ACTION_TYPE_DROP.String()},
					{Type: flowapi.RteFlowActionType_RTE_FLOW_ACTION_TYPE_END.String()}},
			),
		)
	})

	Context("Verify getNodeActionForPodInterface()", func() {
		DescribeTable("Expect nil action and error", func(conf *runtime.RawExtension, pod *corev1.Pod) {
			action, err := clusterFlowConfigRc.getNodeActionForPodInterface(conf, pod)
			Expect(action).Should(BeNil())
			Expect(err).ShouldNot(BeNil())
		},
			Entry("pod nil pointer", createRawExtension("some"), nil),
			Entry("pod and raw extension nil", nil, nil),
			Entry("pod nil, raw extension unmarshal error", func() *runtime.RawExtension {
				typeAction := &flowconfigv1.ClusterFlowAction{Type: 20}
				rawBytes, err := json.Marshal(typeAction)
				if err != nil {
					fmt.Println(err)
					return nil
				}
				return &runtime.RawExtension{Raw: rawBytes}
			}(), nil),
			Entry("pod without annotations", createRawExtension("some"), createPod(podNameDefault, namespaceDefault)),
			Entry("pod with annotations but without network-status", createRawExtension("some"), createPod(podNameDefault, namespaceDefault, func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations = make(map[string]string)
				pod.ObjectMeta.Annotations["some-label"] = "some-text"
			})),
			Entry("pod with network-status inside annotations and with JSON error", createRawExtension("net0"), createPod(podNameDefault, namespaceDefault, func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations = make(map[string]string)
				pod.ObjectMeta.Annotations["k8s.v1.cni.cncf.io/network-status"] = `[
{
	"name": "sriov-network_a",
	"interface": "net1",
	"device-info": {
		"type": "pci",
	}
}]`
			})),
			Entry("pod with network-status inside annotations but missing interface", createRawExtension("net0"), createPod(podNameDefault, namespaceDefault, func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations = make(map[string]string)
				pod.ObjectMeta.Annotations["k8s.v1.cni.cncf.io/network-status"] = `[
{
	"name": "sriov-network_a",
	"interface": "net1",
	"device-info": {
		"type": "pci",
		"version": "1.0.0"
	}
}]`
			})),
			Entry("pod with network-status inside annotations incorrect type", createRawExtension("net1"), createPod(podNameDefault, namespaceDefault, func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations = make(map[string]string)
				pod.ObjectMeta.Annotations["k8s.v1.cni.cncf.io/network-status"] = `[
{
	"name": "sriov-network_a",
	"interface": "net1",
	"device-info": {
		"type": "unknown",
		"version": "1.0.0",
		"pci": {
			"pci-address": "0000:18:02.5",
			"pf-pci-address": "0000:18:00.0"
		}
	}
}]`
			})),
			Entry("pod with network-status inside annotations missing pci-address", createRawExtension("net1"), createPod(podNameDefault, namespaceDefault, func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations = make(map[string]string)
				pod.ObjectMeta.Annotations["k8s.v1.cni.cncf.io/network-status"] = `[
{
	"name": "sriov-network_a",
	"interface": "net1",
	"device-info": {
		"type": "pci",
		"version": "1.0.0",
		"pci": {
			"pf-pci-address": "0000:18:00.0"
		}
	}
}]`
			})),
		)

		DescribeTable("Expect action and nil error", func(conf *runtime.RawExtension, pciAddress string, pod *corev1.Pod) {
			action, err := clusterFlowConfigRc.getNodeActionForPodInterface(conf, pod)
			Expect(err).Should(BeNil())
			Expect(action).ShouldNot(BeNil())
			Expect(action.Type).Should(Equal(flowapi.RTE_FLOW_ACTION_TYPE_VFPCIADDR))
			Expect(action.Conf).ShouldNot(BeNil())
			pciConf := &flowapi.RteFlowActionVfPciAddr{}
			err = json.Unmarshal([]byte(action.Conf.Raw), pciConf)
			Expect(err).Should(BeNil())
			Expect(pciConf.Addr).Should(Equal(pciAddress))
		},
			Entry("pod with network-status inside annotations correct data", createRawExtension("net0"), "0000:18:02.5", createPod(podNameDefault, namespaceDefault, func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations = make(map[string]string)
				pod.ObjectMeta.Annotations["k8s.v1.cni.cncf.io/network-status"] = `[
{
	"name": "sriov-network_a",
	"interface": "net0",
	"device-info": {
		"type": "pci",
		"version": "1.0.0",
		"pci": {
			"pci-address": "0000:18:02.5",
			"pf-pci-address": "0000:18:00.0"
		}
	}
}]`
			})),
			Entry("pod with network-status inside annotations correct data", createRawExtension("net1"), "0000:18:04.5", createPod(podNameDefault, namespaceDefault, func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations = make(map[string]string)
				pod.ObjectMeta.Annotations["k8s.v1.cni.cncf.io/network-status"] = `[
{
	"name": "sriov-network_a",
	"interface": "net0",
	"device-info": {
		"type": "pci",
		"version": "1.0.0",
		"pci": {
			"pci-address": "0000:18:02.5",
			"pf-pci-address": "0000:18:00.0"
		}
	}
},
{
	"name": "sriov-network_a",
	"interface": "net1",
	"device-info": {
		"type": "pci",
		"version": "1.0.0",
		"pci": {
			"pci-address": "0000:18:04.5",
			"pf-pci-address": "0000:18:00.0"
		}
	}
}]`
			})),
		)
	})
})
