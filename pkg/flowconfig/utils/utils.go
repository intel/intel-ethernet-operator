package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	flowapi "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
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

	// Unmarshal into protobuf Any
	anyObj, err := ptypes.MarshalAny(flowObj.(proto.Message))
	if err != nil {
		return nil, fmt.Errorf("error unmarshiling to ptypes.Any: %v", err)
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
		return nil, fmt.Errorf("error unmarshiling to ptypes.Any: %v", err)
	}
	anyObj, err := ptypes.MarshalAny(rteFlowItemIpv4)
	if err != nil {
		return nil, fmt.Errorf("error unmarshiling to ptypes.Any: %v", err)
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
	anyObj, err := ptypes.MarshalAny(rteFlowItemEth)
	if err != nil {
		return nil, fmt.Errorf("error unmarshiling to ptypes.Any: %v", err)
	}
	return anyObj, nil
}

func GetFlowActionAny(actionType string, b []byte) (*any.Any, error) {

	actionTypeVal, ok := flowapi.RteFlowActionType_value[actionType]
	if !ok {
		return nil, fmt.Errorf("invalid action type %s", actionType)
	}
	actionObj := flowapi.GetFlowActionObj(flowapi.RteFlowActionType(actionTypeVal))

	if actionObj == nil {
		// It should not get here
		return nil, fmt.Errorf("nil object received for action type %s", actionType)
	}

	if err := json.Unmarshal(b, actionObj); err != nil {
		return nil, fmt.Errorf("error unmarshalling bytes %s to ptypes.Any: %v", string(b), err)
	}

	// Unmarshal into protobuf Any
	anyObj, err := ptypes.MarshalAny(actionObj.(proto.Message))
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling to ptypes.Any: %v", err)
	}
	return anyObj, nil

}
