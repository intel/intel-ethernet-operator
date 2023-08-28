// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2023 Intel Corporation

package flow

const (
	RTE_FLOW_ACTION_TYPE_VFPCIADDR = "RTE_FLOW_ACTION_TYPE_VFPCIADDR"
)

// RteFlowActionVfPciAddr action provides information about VF PCI address
type RteFlowActionVfPciAddr struct {
	Addr string `protobuf:"varint,1,opt,name=addr" json:"addr,omitempty"`
}

// GetFlowActionType will return action type id from generated proto code
// For user defined action will try to convert action to proto action type
func GetFlowActionType(actionType string) (int32, bool) {
	val, ok := RteFlowActionType_value[actionType]

	// check user defined action types and convert it to proto defined type
	if !ok {
		switch actionType {
		case RTE_FLOW_ACTION_TYPE_VFPCIADDR:
			val, ok = RteFlowActionType_value[RteFlowActionType_name[int32(RteFlowActionType_RTE_FLOW_ACTION_TYPE_VF)]]
		}
	}

	return val, ok
}
