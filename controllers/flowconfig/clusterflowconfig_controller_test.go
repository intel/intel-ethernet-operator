package flowconfig

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"
	flowapi "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cluster Flow Config Controller tests", func() {
	const (
		podNameDefault   = "pod-default"
		namespaceDefault = "default"
	)

	createPod := func(podName, namespace string, configurers ...func(pod *corev1.Pod)) *corev1.Pod {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: namespace,
			},
		}

		for _, config := range configurers {
			config(pod)
		}

		return pod
	}

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

	Context("Verify getNodeActionsFromClusterActions()", func() {
		DescribeTable("Expect results", func(actions []*flowconfigv1.ClusterFlowAction, pod *corev1.Pod, expectedActions []*flowconfigv1.FlowAction) {
			retAction := clusterFlowConfigRc.getNodeActionsFromClusterActions(actions, pod)
			Expect(expectedActions).Should(Equal(retAction))
		},
			Entry("nil input", nil, nil, []*flowconfigv1.FlowAction{}),
			Entry("empty input", []*flowconfigv1.ClusterFlowAction{}, nil, []*flowconfigv1.FlowAction{}),
			Entry("input one action without end, output action with one end action",
				createClusterFlowAction([]flowconfigv1.ClusterFlowActionType{flowconfigv1.ClusterFlowActionType(flowapi.RteFlowActionType_RTE_FLOW_ACTION_TYPE_DROP)}),
				nil,
				[]*flowconfigv1.FlowAction{
					&flowconfigv1.FlowAction{Type: flowapi.RteFlowActionType_RTE_FLOW_ACTION_TYPE_DROP.String()},
					&flowconfigv1.FlowAction{Type: flowapi.RteFlowActionType_RTE_FLOW_ACTION_TYPE_END.String()}},
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
