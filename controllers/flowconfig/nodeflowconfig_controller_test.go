package flowconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"
	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
	mocks "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow/mocks"
	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/utils"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestGetFlowCreateRequests(t *testing.T) {
	var (
		data = `
---
apiVersion: flowconfig.intel.com/v1
kind: NodeFlowConfig
metadata:
  name: testk8snode
  namespace: default
spec:
  rules:
    - pattern:
        - type: RTE_FLOW_ITEM_TYPE_VLAN
          spec:
            tci: 1234
        - type: RTE_FLOW_ITEM_TYPE_IPV4
          spec:
            hdr:
              dst_addr: 192.168.100.10
          mask:
            hdr:
              dst_addr: 255.255.255.0
        - type: RTE_FLOW_ITEM_TYPE_END
      action:
        - type: RTE_FLOW_ACTION_TYPE_VF
          conf:
            id: 1
        - type: RTE_FLOW_ACTION_TYPE_END
      portId: 1
      attr:
        ingress: 1
`
	)

	policy := &flowconfigv1.NodeFlowConfig{}

	jObj, _ := yaml.ToJSON([]byte(data))
	//fmt.Printf("%s\n", string(jObj))

	//dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(data)), 1000)

	if err := json.Unmarshal(jObj, policy); err != nil {
		t.Errorf("error decoding yaml into NodeFlowConfig object: %v", err)
	}

	//if err := dec.Decode(policy); err != nil {
	//	t.Errorf("error decoding yaml into NodeFlowConfig object: %v", err)
	//}
	//fmt.Printf("%+v\n", policy)

	// fmt.Printf("number or rules: %d\n", len(policy.Spec.Rules))
	for i, r := range policy.Spec.Rules {
		// fmt.Printf("item: %+v\n", r)
		flowReq, err := getFlowCreateRequests(r)
		if err != nil {
			t.Errorf("error creating flowRequest object: %v", err)
		}
		_ = flowReq
		fmt.Printf("flowReq[%d]: %s\n", i, flowReq.Pattern[i].Spec)
	}

	// Check dst_addr value
	testRawSpec, err := getFlowCreateRequests(policy.Spec.Rules[0])
	if err != nil {
		t.Errorf("error creating flowRequest object: %v", err)
	}
	rteFlowItemIpv4 := &flow.RteFlowItemIpv4{}
	err = ptypes.UnmarshalAny(testRawSpec.Pattern[1].Spec, rteFlowItemIpv4)
	if err != nil {
		t.Errorf("error unmarshaling object *anypb.Any: %v", err)
	}

	dstAddr := flow.Uint32ToIP(rteFlowItemIpv4.Hdr.DstAddr)
	assert.Equal(t, dstAddr.String(), "192.168.100.10")
}

// TestGetFlowCreateHash compare calculated hash values from RequestFlowCreate instances
func TestGetFlowCreateHash(t *testing.T) {
	specAny1, err := utils.GetFlowItemAny("RTE_FLOW_ITEM_TYPE_IPV4", []byte(`{"hdr":{"dst_addr": "192.168.100.10"}}`))
	if err != nil {
		t.Error("error getting FlowItemAny from raw bytes")
	}

	req1 := &flow.RequestFlowCreate{
		PortId: 0,
		Pattern: []*flow.RteFlowItem{
			{Type: flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_ETH},
			{
				Type: flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_ETH,
				Spec: specAny1,
			},
			{Type: flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_END},
		},
		Action: []*flow.RteFlowAction{
			{Type: flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_DROP},
			{Type: flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_END},
		},
	}

	req2 := &flow.RequestFlowCreate{
		PortId: 0,
		Pattern: []*flow.RteFlowItem{
			{Type: flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_ETH},
			{
				Type: flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_ETH,
				Spec: specAny1,
			},
			{Type: flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_END},
		},
		Action: []*flow.RteFlowAction{
			{Type: flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_DROP},
			{Type: flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_END},
		},
	}

	specAny3, err := utils.GetFlowItemAny("RTE_FLOW_ITEM_TYPE_IPV4", []byte(`{"hdr":{"dst_addr": "192.168.100.11"}}`))
	if err != nil {
		t.Error("error getting FlowItemAny from raw bytes")
	}
	req3 := &flow.RequestFlowCreate{
		PortId: 0,
		Pattern: []*flow.RteFlowItem{
			{Type: flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_ETH},
			{
				Type: flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_ETH,
				Spec: specAny3,
			},
			{Type: flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_END},
		},
		Action: []*flow.RteFlowAction{
			{Type: flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_DROP},
			{Type: flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_END},
		},
	}

	hash1 := getFlowCreateHash(req1)
	hash2 := getFlowCreateHash(req2)
	hash3 := getFlowCreateHash(req3)

	// req1 and req2 are two different instances with same properties
	if hash1 != hash2 {
		t.Fail()
	}

	// req1 and req3 are two different instances with different properties(IP address changed!) so they should have different hash values
	if hash1 == hash3 {
		t.Fail()
	}
}

// TestListDCFPorts is a sample test that uses mock DCF client for NodeFlowConfig Reconciler
func TestListDCFPorts(t *testing.T) {
	mockDCF := &mocks.FlowServiceClient{}
	reconciler := &NodeFlowConfigReconciler{
		flowClient: mockDCF,
	}

	mockRes := &flow.ResponsePortList{
		Ports: []*flow.PortsInformation{
			{
				PortId:   0,
				PortMode: "dcf",
				PortPci:  "0000:01.01",
			},
		},
	}

	// Have mock return our expected mockRes
	mockDCF.On("ListPorts", context.TODO(), &flow.RequestListPorts{}).Return(mockRes, nil)

	ports, err := reconciler.listDCFPorts()
	if err != nil {
		t.Fail()
	}

	if len(ports) != 1 {
		t.Fail()
	}

}

// Controller tests
var _ = Describe("NodeFlowConfig controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	var (
		portID uint32 = 0
	)

	const (
		NodeFlowConfigNamespace = "default"

		timeout  = time.Second * 20
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)
	Context("when creating new NodeFlowConfig spec", func() {
		It("should update Status with DCF port info", func() {
			By("Creating new NodeFlowConfigSpec")
			ctx := context.Background()

			policy := &flowconfigv1.NodeFlowConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "flowconfig.intel.com/v1",
					Kind:       "NodeFlowConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeName,
					Namespace: NodeFlowConfigNamespace,
				},
				Spec: flowconfigv1.NodeFlowConfigSpec{
					Rules: []*flowconfigv1.FlowRules{
						{
							PortId: portID,
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
							Action: []*flowconfigv1.FlowAction{
								{
									Type: "RTE_FLOW_ACTION_TYPE_DROP",
								},
								{
									Type: "RTE_FLOW_ACTION_TYPE_END",
								},
							},
							Attr: &flowconfigv1.FlowAttr{
								Ingress: 1,
							},
						},
					},
				},
			}

			mockRes := &flow.ResponsePortList{
				Ports: []*flow.PortsInformation{
					{
						PortId:   0,
						PortMode: "dcf",
						PortPci:  "0000:01.01",
					},
				},
			}
			mockDCF.On("ListPorts", context.TODO(), &flow.RequestListPorts{}).Return(mockRes, nil)

			if policy.Spec.Rules != nil {
				var flowID uint32 = 0
				for range policy.Spec.Rules {
					mockValidateResponse := &flow.ResponseFlow{}
					mockDCF.On("Validate", context.TODO(), mock.AnythingOfType("*flow.RequestFlowCreate")).Return(mockValidateResponse, nil)

					mockCreateResponse := &flow.ResponseFlowCreate{FlowId: flowID}
					mockDCF.On("Create", context.TODO(), mock.AnythingOfType("*flow.RequestFlowCreate")).Return(mockCreateResponse, nil)

					mockDestroyReq := &flow.RequestFlowofPort{PortId: portID, FlowId: flowID}
					mockDCF.On("Destroy", context.TODO(), mockDestroyReq).Return(mockValidateResponse, nil)
					flowID++
				}
			}

			Expect(k8sClient.Create(ctx, policy)).Should(Succeed())

			/*
				After the policy spec is created, we expect the controller should update its internal state in its flowSets field and also update
				it's '.Status'
			*/

			policyObjLookupKey := types.NamespacedName{Name: nodeName, Namespace: NodeFlowConfigNamespace}
			createdPolicyObj := &flowconfigv1.NodeFlowConfig{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, policyObjLookupKey, createdPolicyObj)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Verify that Status is updated with correct Port Information
			Eventually(len(createdPolicyObj.Status.PortInfo), timeout, interval).Should(Equal(1))
		})

	})
})
