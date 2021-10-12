// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	nodeFlowConfigNamespace = "default"
	portID                  = 0
)

var (
	invalidIpv4LastLowerThanSpec = []*FlowRules{
		{
			PortId: portID,
			Pattern: []*FlowItem{
				{
					Type: "RTE_FLOW_ITEM_TYPE_IPV4",
					Spec: &runtime.RawExtension{
						Raw: []byte(`{ "hdr": { "src_addr": "10.56.217.9" } }`),
					},
					Last: &runtime.RawExtension{
						Raw: []byte(`{ "hdr": { "src_addr": "10.56.217.8" } }`),
					},
				},
				{
					Type: "RTE_FLOW_ITEM_TYPE_END",
				},
			},
			Action: []*FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_END",
				},
			},
			Attr: &FlowAttr{
				Ingress: 1,
			},
		},
	}
	invalidIpv4LastFieldOutOfRange = []*FlowRules{
		{
			PortId: portID,
			Pattern: []*FlowItem{
				{
					Type: "RTE_FLOW_ITEM_TYPE_IPV4",
					Spec: &runtime.RawExtension{
						Raw: []byte(`{ "hdr": { "version_ihl": 130 } }`),
					},
					Last: &runtime.RawExtension{
						Raw: []byte(`{ "hdr": { "version_ihl": 333 } }`),
					},
				},
				{
					Type: "RTE_FLOW_ITEM_TYPE_END",
				},
			},
			Action: []*FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_END",
				},
			},
			Attr: &FlowAttr{
				Ingress: 1,
			},
		},
	}
	invalidUdpSpecFieldOutOfRange = []*FlowRules{
		{
			PortId: portID,
			Pattern: []*FlowItem{
				{
					Type: "RTE_FLOW_ITEM_TYPE_UDP",
					Spec: &runtime.RawExtension{
						Raw: []byte(`{ "hdr": { "src_port": 1048561 } }`),
					},
				},
				{
					Type: "RTE_FLOW_ITEM_TYPE_END",
				},
			},
			Action: []*FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_END",
				},
			},
			Attr: &FlowAttr{
				Ingress: 1,
			},
		},
	}
	invalidEthFieldName = []*FlowRules{
		{
			PortId: portID,
			Pattern: []*FlowItem{
				{
					Type: "RTE_FLOW_ITEM_TYPE_ETH",
					Spec: &runtime.RawExtension{
						Raw: []byte(`{ "invalidField": 489523820 }`),
					},
				},
				{
					Type: "RTE_FLOW_ITEM_TYPE_END",
				},
			},
			Action: []*FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_END",
				},
			},
			Attr: &FlowAttr{
				Ingress: 1,
			},
		},
	}
	invalidPppoeFieldOutOfRange = []*FlowRules{
		{
			PortId: portID,
			Pattern: []*FlowItem{
				{
					Type: "RTE_FLOW_ITEM_TYPE_PPPOES",
					Spec: &runtime.RawExtension{
						Raw: []byte(`{ "version_type": 299 }`),
					},
				},
				{
					Type: "RTE_FLOW_ITEM_TYPE_END",
				},
			},
			Action: []*FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_END",
				},
			},
			Attr: &FlowAttr{
				Ingress: 1,
			},
		},
	}
	invalidPppoeProtoIdFieldOutOfRange = []*FlowRules{
		{
			PortId: portID,
			Pattern: []*FlowItem{
				{
					Type: "RTE_FLOW_ITEM_TYPE_PPPOE_PROTO_ID",
					Spec: &runtime.RawExtension{
						Raw: []byte(`{ "proto_id": 1048561 }`),
					},
				},
				{
					Type: "RTE_FLOW_ITEM_TYPE_END",
				},
			},
			Action: []*FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_END",
				},
			},
			Attr: &FlowAttr{
				Ingress: 1,
			},
		},
	}
	invalidVlanFieldOutOfRange = []*FlowRules{
		{
			PortId: portID,
			Pattern: []*FlowItem{
				{
					Type: "RTE_FLOW_ITEM_TYPE_VLAN",
					Spec: &runtime.RawExtension{
						Raw: []byte(`{ "tci": 12345 , "inner_type": 543211}`),
					},
				},
				{
					Type: "RTE_FLOW_ITEM_TYPE_END",
				},
			},
			Action: []*FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_END",
				},
			},
			Attr: &FlowAttr{
				Ingress: 1,
			},
		},
	}
	validUnsupportedItem = []*FlowRules{
		{
			PortId: portID,
			Pattern: []*FlowItem{
				{
					Type: "RTE_FLOW_ITEM_TYPE_IPV6",
					Spec: &runtime.RawExtension{
						Raw: []byte(`{ "hdr": { "vtc_flow": 12 } }`),
					},
				},
				{
					Type: "RTE_FLOW_ITEM_TYPE_END",
				},
			},
			Action: []*FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_END",
				},
			},
			Attr: &FlowAttr{
				Ingress: 1,
			},
		},
	}
	lastWithEmptySpec = []*FlowRules{
		{
			PortId: portID,
			Pattern: []*FlowItem{
				{
					Type: "RTE_FLOW_ITEM_TYPE_IPV4",
					Last: &runtime.RawExtension{
						Raw: []byte(`{ "hdr": { "version_ihl": 12 } }`),
					},
				},
				{
					Type: "RTE_FLOW_ITEM_TYPE_END",
				},
			},
			Action: []*FlowAction{
				{
					Type: "RTE_FLOW_ACTION_TYPE_END",
				},
			},
			Attr: &FlowAttr{
				Ingress: 1,
			},
		},
	}
)
