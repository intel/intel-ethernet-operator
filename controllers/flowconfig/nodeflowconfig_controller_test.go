// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flowconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sriovutils "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	mock "github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"
	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
	mocks "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow/mocks"
	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/utils"
)

// Controller tests
var _ = Describe("NodeFlowConfig controller", func() {

	const (
		nodeFlowConfigNamespace = "default"

		timeout  = time.Second * 20
		interval = time.Millisecond * 250
	)

	Context("when the controller is reconciling", func() {

		// Define utility constants for object names and testing timeouts/durations and intervals.
		var (
			portID uint32 = 0
			ctx    context.Context
			policy *flowconfigv1.NodeFlowConfig
		)

		BeforeEach(func() {
			ctx = context.Background()
			policy = &flowconfigv1.NodeFlowConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "flowconfig.intel.com/v1",
					Kind:       "NodeFlowConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      nodeName,
					Namespace: nodeFlowConfigNamespace,
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
		})

		var createdPolicyObj *flowconfigv1.NodeFlowConfig

		Context("when a new NodeFlowConfig spec is created", func() {
			It("should update the controller's internal config", func() {
				Eventually(func() bool {
					err := k8sClient.Create(ctx, policy)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				// Add delays after creating api object before retrieving it again
				time.Sleep(time.Second * 2)

				/*
					After the policy spec is created, we expect the controller should update its internal state in its flowSets field and also update
					it's '.Status'
				*/
				policyObjLookupKey := types.NamespacedName{Name: nodeName, Namespace: nodeFlowConfigNamespace}
				createdPolicyObj = &flowconfigv1.NodeFlowConfig{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, policyObjLookupKey, createdPolicyObj)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				By("updating its Status with DCF port info")
				Expect(len(createdPolicyObj.Status.PortInfo)).Should(Equal(1))

				By("updating its flowSets with the new NodeFlowConfig")
				Expect(nodeFlowConfigRc.flowSets.Size()).Should(Equal(1))
			})
		})

		Context("when a NodeFlowConfig spec is updated with duplicate flow rules", func() {
			It("should not be added to the controller's internal config", func() {
				Eventually(func() bool {
					err := k8sClient.Update(ctx, createdPolicyObj)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				// Add delays after creating api object before retrieving it again
				time.Sleep(time.Second * 2)

				/*
					After the policy spec is updated (i.e. duplicated), we expect the controller to identify the new rule as a duplicate and should not update its internal state in its flowSets
				*/
				By("not updating its flowSets with a duplicate entry")
				Expect(nodeFlowConfigRc.flowSets.Size()).Should(Equal(1))
			})
		})

		Context("when a NodeFlowConfig spec is deleted", func() {
			It("should reset the controller's internal config", func() {
				Eventually(func() bool {
					err := k8sClient.Delete(ctx, createdPolicyObj)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				// Add delays after deleting api object before validating the controller's default config
				time.Sleep(time.Second * 2)
				/*
					When a NodeFlowConfig object is deleted, we expect the controller to delete all rules from its default config.
				*/
				By("deleting the spec from the controller's flowSets")
				Expect(nodeFlowConfigRc.flowSets.Size()).Should(Equal(0))
			})
		})
	})

	Context("Creating a hash value from a RequestFlowCreate object", func() {
		specAny1, err := utils.GetFlowItemAny("RTE_FLOW_ITEM_TYPE_IPV4", []byte(`{"hdr":{"dst_addr": "192.168.100.10"}}`))
		Expect(err).Should(BeNil())

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
		Expect(err).Should(BeNil())

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

		It("should create create the same hash for requests with the same properties", func() {
			Expect(hash1).Should(Equal(hash2))
		})

		It("should create unique hashes for requests with different properties", func() {
			Expect(hash1).ShouldNot(Equal(hash3))
		})
	})

	Context("when listing DCF ports", func() {
		It("should store port information in the controller", func() {
			mockFlowServiceClient := &mocks.FlowServiceClient{}
			reconciler := &NodeFlowConfigReconciler{
				flowClient: mockFlowServiceClient,
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
			mockFlowServiceClient.On("ListPorts", context.TODO(), &flow.RequestListPorts{}).Return(mockRes, nil)

			ports, err := reconciler.listDCFPorts()

			Expect(err).Should(BeNil())
			Expect(len(ports)).Should(Equal(1))
		})

		It("Should rethrow any error thrown by the DCF", func() {
			mockFlowServiceClient := &mocks.FlowServiceClient{}
			reconciler := &NodeFlowConfigReconciler{
				flowClient: mockFlowServiceClient,
			}

			mockError := fmt.Errorf("mock error")
			mockFlowServiceClient.On("ListPorts", context.TODO(), &flow.RequestListPorts{}).Return(nil, mockError)

			ports, err := reconciler.listDCFPorts()

			Expect(ports).Should(BeNil())
			Expect(err).Should(Equal(mockError))
		})
	})

	Context("When creating a FlowCreateRequests from a flow rule", func() {
		Context("with a valid yaml", func() {
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
			err := json.Unmarshal(jObj, policy)
			Expect(err).Should(BeNil())

			for _, r := range policy.Spec.Rules {
				_, err := getFlowCreateRequests(r)
				Expect(err).Should(BeNil())
			}

			// Check dst_addr value
			testRawSpec, err := getFlowCreateRequests(policy.Spec.Rules[0])
			Expect(err).Should(BeNil())

			It("should inherit all flow patterns from the rule", func() {
				rteFlowItemIpv4 := &flow.RteFlowItemIpv4{}
				err = testRawSpec.Pattern[1].Spec.UnmarshalTo(rteFlowItemIpv4)
				Expect(err).Should(BeNil())

				dstAddr := flow.Uint32ToIP(rteFlowItemIpv4.Hdr.DstAddr)
				Expect(dstAddr.String()).Should(Equal("192.168.100.10"))
			})

			It("Should inherit all flow actions from the rule", func() {
				rteFlowActionTypeVF := &flow.RteFlowActionVf{}
				err = testRawSpec.Action[0].Conf.UnmarshalTo(rteFlowActionTypeVF)
				Expect(err).Should(BeNil())

				actionId := rteFlowActionTypeVF.Id
				Expect(actionId).Should(Equal(uint32(1)))
			})

			It("Should inherit all flow attributes from the rule", func() {
				flowAttrIngress := testRawSpec.Attr.Ingress
				Expect(flowAttrIngress).Should(Equal(uint32(1)))
			})
		})

		Context("with a valid yaml with invalid data", func() {
			Context("pattern is invalid", func() {
				It("should throw an error if pattern type is invalid", func() {
					invalidPattern := []*flowconfigv1.FlowItem{
						{
							Type: "INVALID_PATTERN_TYPE",
						},
					}

					flowRules := &flowconfigv1.FlowRules{
						Pattern: invalidPattern,
					}

					flowReqs, err := getFlowCreateRequests(flowRules)
					Expect(flowReqs).Should(BeNil())

					expectedErr := fmt.Errorf("invalid flow item type %s", "INVALID_PATTERN_TYPE")
					Expect(err.Error()).Should(Equal(expectedErr.Error()))
				})

				It("should throw an error if spec pattern is invalid", func() {
					invalidSpecPattern := &runtime.RawExtension{

						Raw: []byte("not a valid json"),
					}

					invalidPattern := []*flowconfigv1.FlowItem{
						{
							Type: "RTE_FLOW_ITEM_TYPE_VLAN",
							Spec: invalidSpecPattern,
						},
					}

					flowRules := &flowconfigv1.FlowRules{
						Pattern: invalidPattern,
					}

					flowReqs, err := getFlowCreateRequests(flowRules)
					Expect(flowReqs).Should(BeNil())

					expectedErrSegment := "error getting Spec pattern for flowtype"
					Expect(strings.Contains(err.Error(), expectedErrSegment)).Should(BeTrue())
				})

				It("should throw an error if last pattern is invalid", func() {
					invalidLastPattern := &runtime.RawExtension{
						Raw: []byte("not a valid json"),
					}

					invalidPattern := []*flowconfigv1.FlowItem{
						{
							Type: "RTE_FLOW_ITEM_TYPE_VLAN",
							Last: invalidLastPattern,
						},
					}

					flowRules := &flowconfigv1.FlowRules{
						Pattern: invalidPattern,
					}

					flowReqs, err := getFlowCreateRequests(flowRules)
					Expect(flowReqs).Should(BeNil())

					expectedErrSegment := "error getting Last pattern for flowtype"
					Expect(strings.Contains(err.Error(), expectedErrSegment)).Should(BeTrue())
				})

				It("should throw an error if mask pattern is invalid", func() {
					invalidMaskPattern := &runtime.RawExtension{
						Raw: []byte("not a valid json"),
					}

					invalidPattern := []*flowconfigv1.FlowItem{
						{
							Type: "RTE_FLOW_ITEM_TYPE_VLAN",
							Mask: invalidMaskPattern,
						},
					}

					flowRules := &flowconfigv1.FlowRules{
						Pattern: invalidPattern,
					}

					flowReqs, err := getFlowCreateRequests(flowRules)
					Expect(flowReqs).Should(BeNil())

					expectedErrSegment := "error getting Mask pattern for flowtype"
					Expect(strings.Contains(err.Error(), expectedErrSegment)).Should(BeTrue())
				})
			})

		})
		Context("action is invalid", func() {
			It("should throw an error if action type is invalid", func() {
				invalidAction := []*flowconfigv1.FlowAction{
					{
						Type: "INVALID_ACTION_TYPE",
					},
				}

				flowRules := &flowconfigv1.FlowRules{
					Action: invalidAction,
				}

				flowReqs, err := getFlowCreateRequests(flowRules)
				Expect(flowReqs).Should(BeNil())

				expectedErr := fmt.Errorf("invalid action type %s", "INVALID_ACTION_TYPE")
				Expect(err.Error()).Should(Equal(expectedErr.Error()))
			})

			It("should throw an error if conf is invalid", func() {
				invalidSpecPattern := &runtime.RawExtension{
					Raw: []byte("not a valid json"),
				}

				invalidAction := []*flowconfigv1.FlowAction{
					{
						Type: "RTE_FLOW_ACTION_TYPE_JUMP",
						Conf: invalidSpecPattern,
					},
				}

				flowRules := &flowconfigv1.FlowRules{
					Action: invalidAction,
				}

				flowReqs, err := getFlowCreateRequests(flowRules)
				Expect(flowReqs).Should(BeNil())

				expectedErrSegment := "error getting Spec pattern for flowtype"
				Expect(strings.Contains(err.Error(), expectedErrSegment)).Should(BeTrue())
			})
		})
	})

	Context("When creating rules", func() {

		var (
			reqFlowCreate         *flow.RequestFlowCreate
			mockFlowServiceClient *mocks.FlowServiceClient
			mockError             error
			toAdd                 map[string]*flow.RequestFlowCreate
		)

		BeforeEach(func() {
			reqFlowCreate = &flow.RequestFlowCreate{
				PortId: uint32(0),
			}
			mockFlowServiceClient = &mocks.FlowServiceClient{}
			mockError = fmt.Errorf("this error is forced")
			toAdd = make(map[string]*flow.RequestFlowCreate)
		})

		Context("error occurs during validation with DCF", func() {
			It("should return a DCFError", func() {
				mockFlowServiceClient.On("Validate", context.TODO(), reqFlowCreate).Return(nil, mockError)

				reconciler := &NodeFlowConfigReconciler{
					Log:        logf.Log.WithName("scoped"),
					flowClient: mockFlowServiceClient,
				}

				toAdd[getFlowCreateHash(reqFlowCreate)] = reqFlowCreate

				expectedErr := fmt.Sprintf("error validating flow create request: %v", mockError)
				err := reconciler.createRules(toAdd)
				Expect(err.Error()).Should(Equal(expectedErr))
			})
		})

		Context("response contains error info", func() {
			It("should return a RteFlowError", func() {
				mockRes := &flow.ResponseFlow{
					ErrorInfo: &flow.RteFlowError{
						Type: 1,
						Mesg: "mock error",
					},
				}

				mockFlowServiceClient.On("Validate", context.TODO(), reqFlowCreate).Return(mockRes, nil)

				reconciler := &NodeFlowConfigReconciler{
					Log:        logf.Log.WithName("scoped"),
					flowClient: mockFlowServiceClient,
				}

				toAdd[getFlowCreateHash(reqFlowCreate)] = reqFlowCreate

				expectedErr := "received validation error: mock error"
				err := reconciler.createRules(toAdd)
				Expect(err.Error()).Should(Equal(expectedErr))
			})
		})

		Context("Error occurs during creation of rule", func() {
			It("should return a DCFError", func() {
				mockRes := &flow.ResponseFlow{}
				mockFlowServiceClient.On("Validate", context.TODO(), reqFlowCreate).Return(mockRes, nil)
				mockFlowServiceClient.On("Create", context.TODO(), reqFlowCreate).Return(nil, mockError)

				reconciler := &NodeFlowConfigReconciler{
					Log:        logf.Log.WithName("scoped"),
					flowClient: mockFlowServiceClient,
				}

				toAdd[getFlowCreateHash(reqFlowCreate)] = reqFlowCreate

				expectedErr := "error creating flow rules: this error is forced"
				err := reconciler.createRules(toAdd)
				Expect(err.Error()).Should(Equal(expectedErr))
			})
		})

		Context("Creation response contains error info", func() {
			It("should return a RteFlowError", func() {
				mockRes := &flow.ResponseFlow{}
				mockCreateRes := &flow.ResponseFlowCreate{
					ErrorInfo: &flow.RteFlowError{
						Type: 1,
						Mesg: "mock error",
					},
				}
				mockFlowServiceClient.On("Validate", context.TODO(), reqFlowCreate).Return(mockRes, nil)
				mockFlowServiceClient.On("Create", context.TODO(), reqFlowCreate).Return(mockCreateRes, nil)

				reconciler := &NodeFlowConfigReconciler{
					Log:        logf.Log.WithName("scoped"),
					flowClient: mockFlowServiceClient,
				}

				toAdd[getFlowCreateHash(reqFlowCreate)] = reqFlowCreate

				expectedErr := "received flow create error: mock error"
				err := reconciler.createRules(toAdd)
				Expect(err.Error()).Should(Equal(expectedErr))
			})
		})
	})

	Context("when converting PCI address into VF index", func() {
		var (
			policy *flowconfigv1.NodeFlowConfig
			portID uint32 = 0

			createNodeFlowConfig = func(nodeName string, portID uint32, configurers ...func(config *flowconfigv1.NodeFlowConfig)) *flowconfigv1.NodeFlowConfig {
				policy := &flowconfigv1.NodeFlowConfig{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "flowconfig.intel.com/v1",
						Kind:       "NodeFlowConfig",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      nodeName,
						Namespace: nodeFlowConfigNamespace,
					},
					Spec: flowconfigv1.NodeFlowConfigSpec{
						Rules: []*flowconfigv1.FlowRules{
							{
								PortId: portID,
								Attr: &flowconfigv1.FlowAttr{
									Ingress: 1,
								},
							},
						},
					},
				}

				for _, config := range configurers {
					config(policy)
				}

				return policy
			}
		)

		It("Verify full flow", func() {
			policy = createNodeFlowConfig(nodeName, portID, func(config *flowconfigv1.NodeFlowConfig) {
				config.Spec.Rules[0].Action = []*flowconfigv1.FlowAction{
					{
						Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR",
						Conf: &runtime.RawExtension{
							Raw: []byte(`{ "addr": "0000:0a:11.1" }`),
						},
					},
					{
						Type: "RTE_FLOW_ACTION_TYPE_END",
					},
				}
			})

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

					mockDestroyReq := &flow.RequestFlowofPort{PortId: 0, FlowId: flowID}
					mockDCF.On("Destroy", context.TODO(), mockDestroyReq).Return(mockValidateResponse, nil)
					flowID++
				}
			}

			Eventually(func() bool {
				err := k8sClient.Create(context.Background(), policy)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			time.Sleep(2 * time.Second)
			defer func() {
				Eventually(func() bool {
					err := k8sClient.Delete(context.Background(), policy)
					return err == nil
				}, timeout, interval).Should(BeTrue())
			}()
		})

		It("Verify that function will be able to convert PCI -> VF ID correct PCI address", func() {
			fs := &sriovutils.FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/", "sys/bus/pci/devices/0000:01:00.0/"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:10.0/physfn": "../0000:01:00.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn0": "../0000:01:08.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn1": "../0000:01:09.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn2": "../0000:01:10.0",
				},
			}
			defer fs.Use()()

			action := []*flowconfigv1.FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR",
					Conf: &runtime.RawExtension{
						Raw: []byte(`{ "addr": "0000:01:10.0" }`),
					},
				},
			}

			flowRules := &flowconfigv1.FlowRules{
				Action: action,
			}

			flowReqs, err := getFlowCreateRequests(flowRules)
			Expect(err).Should(BeNil())
			Expect(flowReqs).ShouldNot(BeNil())
			Expect(flowReqs.PortId).Should(Equal(uint32(0)))
			Expect(flowReqs.Attr).Should(BeNil())
			Expect(flowReqs.Pattern).Should(BeNil())
			Expect(len(flowReqs.Action)).Should(Equal(1))
			Expect(flowReqs.Action[0].Type).Should(Equal(flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_VF))

			rteFlowActionTypeVF := &flow.RteFlowActionVf{}
			err = flowReqs.Action[0].Conf.UnmarshalTo(rteFlowActionTypeVF)
			Expect(err).Should(BeNil())
			Expect(rteFlowActionTypeVF.Id).Should(Equal(uint32(2)))
		})

		It("Verify that function will be not able to convert PCI -> VF ID - due to wrong PCI address", func() {
			fs := &sriovutils.FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/", "sys/bus/pci/devices/0000:01:00.0/"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:10.0/physfn": "../0000:01:00.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn0": "../0000:01:08.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn1": "../0000:01:09.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn2": "../0000:01:10.0",
				},
			}
			defer fs.Use()()

			action := []*flowconfigv1.FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR",
					Conf: &runtime.RawExtension{
						Raw: []byte(`{ "addr": "0000:0a:55.1" }`),
					},
				},
			}

			flowRules := &flowconfigv1.FlowRules{
				Action: action,
			}

			flowReqs, err := getFlowCreateRequests(flowRules)
			Expect(flowReqs).Should(BeNil())

			expectedErr := fmt.Errorf("error getting Spec pattern for flowtype %v : error unable to get VF ID for PCI: 0000:0a:55.1, Err: %v", nil, nil)
			Expect(err.Error()).Should(Equal(expectedErr.Error()))
		})

		It("Verify that function will be not able to convert PCI -> VF ID - missing PCI address", func() {
			action := []*flowconfigv1.FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR",
				},
			}

			flowRules := &flowconfigv1.FlowRules{
				Action: action,
			}

			flowReqs, err := getFlowCreateRequests(flowRules)
			Expect(err).Should(BeNil())
			Expect(flowReqs).ShouldNot(BeNil())
			Expect(flowReqs.PortId).Should(Equal(uint32(0)))
			Expect(flowReqs.Attr).Should(BeNil())
			Expect(flowReqs.Pattern).Should(BeNil())
			Expect(len(flowReqs.Action)).Should(Equal(1))
			Expect(flowReqs.Action[0].Type).Should(Equal(flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_VF))

			rteFlowActionTypeVF := &flow.RteFlowActionVf{}
			err = flowReqs.Action[0].Conf.UnmarshalTo(rteFlowActionTypeVF)
			Expect(err).ShouldNot(BeNil())
		})
	})
})
