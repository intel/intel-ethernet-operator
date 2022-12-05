// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	flowapi "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
	sutils "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/sriovutils"
	"google.golang.org/protobuf/proto"
	any "google.golang.org/protobuf/types/known/anypb"
)

func GetFlowItemAny(flowType string, b []byte) (*any.Any, error) {

	flowTypeVal, ok := flowapi.RteFlowItemType_value[flowType]
	if !ok {
		return nil, fmt.Errorf("invalid  flow type %s", flowType)
	}

	// Handle Eth item differently
	if flowapi.RteFlowItemType_RTE_FLOW_ITEM_TYPE_ETH == flowapi.RteFlowItemType(flowTypeVal) {
		return GetEthAnyObj(b)
	}

	// Handle IPv4 headers differently
	if flowapi.RteFlowItemType_RTE_FLOW_ITEM_TYPE_IPV4 == flowapi.RteFlowItemType(flowTypeVal) {
		return GetIPv4AnyObj(b)
	}

	flowObj := flowapi.GetFlowItemObj(flowapi.RteFlowItemType(flowTypeVal))
	if flowObj == nil {
		// It should not get here
		return nil, fmt.Errorf("nil object received for item type %s", flowType)
	}

	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()

	err := dec.Decode(&flowObj)
	if err != nil {
		return nil, fmt.Errorf("could not decode bytes %s to ptypes.Any %v", string(b), err)
	}

	// Marshal into protobuf Any
	anyObj, err := any.New(flowObj.(proto.Message))
	if err != nil {
		return nil, fmt.Errorf("error marshalling into ptypes.Any: %v", err)
	}
	return anyObj, nil

}

func GetIPv4AnyObj(b []byte) (*any.Any, error) {
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	ipv4 := new(flowapi.Ipv4)

	err := dec.Decode(&ipv4)
	if err != nil {
		return nil, fmt.Errorf("could not decode bytes %s to ptypes.Any %v", string(b), err)
	}

	rteFlowItemIpv4, err := ipv4.ToRteFlowItemIpv4()
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling to ptypes.Any: %v", err)
	}
	anyObj, err := any.New(rteFlowItemIpv4)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to ptypes.Any: %v", err)
	}
	return anyObj, nil
}

func GetEthAnyObj(b []byte) (*any.Any, error) {
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	eth := new(flowapi.Eth)

	err := dec.Decode(&eth)
	if err != nil {
		return nil, fmt.Errorf("could not decode bytes %s to ptypes.Any %v", string(b), err)
	}

	rteFlowItemEth, err := eth.ToRteFlowItemEth()
	if err != nil {
		return nil, fmt.Errorf("error unmarshiling to ptypes.Any: %v", err)
	}
	anyObj, err := any.New(rteFlowItemEth)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to ptypes.Any: %v", err)
	}
	return anyObj, nil
}

func GetFlowActionAnyForWebhook(actionType string, b []byte) (*any.Any, error) {
	actionTypeVal, ok := flowapi.RteFlowActionType_value[actionType]
	if !ok {
		// due to the fact that this is action type not defined in proto it has to be handled separately
		if actionType == flowapi.RTE_FLOW_ACTION_TYPE_VFPCIADDR {
			return handleActionVfPciAddr(b, true, nil)
		} else {
			return nil, fmt.Errorf("invalid action type %s", actionType)
		}
	}

	actionObj := flowapi.GetFlowActionObj(flowapi.RteFlowActionType(actionTypeVal))
	if actionObj == nil {
		// It should not get here
		return nil, fmt.Errorf("nil object received for action type %s", actionType)
	}

	if err := json.Unmarshal(b, actionObj); err != nil {
		return nil, fmt.Errorf("error unmarshalling bytes %s to ptypes.Any: %v", string(b), err)
	}

	// Marshal into protobuf Any
	anyObj, err := any.New(actionObj.(proto.Message))
	if err != nil {
		return nil, fmt.Errorf("error marshalling into ptypes.Any: %v", err)
	}
	return anyObj, nil
}

func GetFlowActionAny(actionType string, b []byte, sriovutils sutils.SriovUtils) (*any.Any, error) {
	actionTypeVal, ok := flowapi.RteFlowActionType_value[actionType]
	if !ok {
		// due to the fact that this is action type not defined in proto it has to be handled separately
		if actionType == flowapi.RTE_FLOW_ACTION_TYPE_VFPCIADDR {
			return handleActionVfPciAddr(b, false, sriovutils)
		} else {
			return nil, fmt.Errorf("invalid action type %s", actionType)
		}
	}

	actionObj := flowapi.GetFlowActionObj(flowapi.RteFlowActionType(actionTypeVal))
	if actionObj == nil {
		// It should not get here
		return nil, fmt.Errorf("nil object received for action type %s", actionType)
	}

	if err := json.Unmarshal(b, actionObj); err != nil {
		return nil, fmt.Errorf("error unmarshalling bytes %s to ptypes.Any: %v", string(b), err)
	}

	// Marshal into protobuf Any
	anyObj, err := any.New(actionObj.(proto.Message))
	if err != nil {
		return nil, fmt.Errorf("error marshalling into ptypes.Any: %v", err)
	}
	return anyObj, nil
}

// handleActionVfPciAddr manually extracts PCI address from user defined action message and pass it as a byte buffer
// to function that creates RTE_FLOW_ACTION_TYPE_VF actionAny object
func handleActionVfPciAddr(b []byte, isCalledByWebhook bool, sriovutils sutils.SriovUtils) (*any.Any, error) {
	actionObj := flowapi.RteFlowActionVfPciAddr{}

	if err := json.Unmarshal(b, &actionObj); err != nil {
		return nil, fmt.Errorf("error unmarshalling bytes %s to ptypes.Any: %v", string(b), err)
	}

	// when this method is called not by webhook, convert PCI address into VF index
	if !isCalledByWebhook {
		// get VF index from PCI address
		vfID, err := sriovutils.GetVFID(actionObj.Addr)
		if err != nil || vfID == -1 {
			return nil, fmt.Errorf("error unable to get VF ID for PCI: %s, Err: %v", actionObj.Addr, err)
		}

		// converted VF PCI address into VF index now can be passed once again to get ActionAny object
		return GetFlowActionAny("RTE_FLOW_ACTION_TYPE_VF", []byte(fmt.Sprintf("{\"id\":%d}", vfID)), sriovutils)
	}

	// verify if field addr contains string that represents valid PCI address - check its format
	re, err := regexp.Compile(`\b([0-9a-fA-F]{4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}.\d{1})`)
	if err != nil {
		return nil, fmt.Errorf("error unable to compile regExp")
	}

	if re.MatchString(actionObj.Addr) {
		// return RTE_FLOW_ACTION_TYPE_VF without converted VF PCI address into VF index, conversion is only possible on worker node where PCI address exists
		return GetFlowActionAny("RTE_FLOW_ACTION_TYPE_VF", []byte(fmt.Sprintf("{\"id\":%d}", 0)), sriovutils)
	}

	return nil, fmt.Errorf("error unable to handle VF PCI address - please verify its correctness")
}
