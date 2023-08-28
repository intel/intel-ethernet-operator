// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2023 Intel Corporation

package v1

import (
	"fmt"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
	"github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/pkg/flowconfig/utils"
)

func validateFlowPatterns(patterns []*FlowItem) error {
	for i, item := range patterns {

		// validate single RteFlowItem spec
		if err := validateRteFlowItem(item); err != nil {
			return fmt.Errorf("pattern[%v] invalid: %v", i, err)
		}

		// ensure that last action is RTE_FLOW_ITEM_TYPE_END
		if i == len(patterns)-1 {
			if item.Type != "RTE_FLOW_ITEM_TYPE_END" {
				return fmt.Errorf("invalid: last pattern must be RTE_FLOW_ITEM_TYPE_END")
			}
		}
	}

	return nil
}

func validateRteFlowItem(item *FlowItem) error {
	rteFlowItem := new(flow.RteFlowItem)

	val, ok := flow.RteFlowItemType_value[item.Type]
	if !ok {
		return fmt.Errorf("invalid flow item type: '%s'", item.Type)
	}
	flowType := flow.RteFlowItemType(val)
	rteFlowItem.Type = flowType

	if item.Spec != nil {
		specAny, err := utils.GetFlowItemAny(item.Type, item.Spec.Raw)
		if err != nil {
			return fmt.Errorf("invalid 'spec' in pattern type %s: '%v'", flowType, err)
		}
		rteFlowItem.Spec = specAny
		if err := validateItem(rteFlowItem.Type, "spec", nil, rteFlowItem.Spec); err != nil {
			return fmt.Errorf("validateItem(): error validating %s spec: '%v'", rteFlowItem.Type, err)
		}
	}

	if item.Last != nil {
		lastAny, err := utils.GetFlowItemAny(item.Type, item.Last.Raw)
		if err != nil {
			return fmt.Errorf("invalid 'last' in pattern type %s: '%v'", flowType, err)
		}
		rteFlowItem.Last = lastAny
		if err := validateItem(rteFlowItem.Type, "last", rteFlowItem.Spec, rteFlowItem.Last); err != nil {
			return fmt.Errorf("validateItem(): error validating %s last: '%v'", rteFlowItem.Type, err)
		}
	}

	if item.Mask != nil {
		maskAny, err := utils.GetFlowItemAny(item.Type, item.Mask.Raw)
		if err != nil {
			return fmt.Errorf("invalid 'mask' in pattern type %s: '%v'", flowType, err)
		}
		rteFlowItem.Mask = maskAny
		if err := validateItem(rteFlowItem.Type, "mask", rteFlowItem.Spec, rteFlowItem.Mask); err != nil {
			return fmt.Errorf("validateItem(): error validating %s mask: '%v'", rteFlowItem.Type, err)
		}
	}
	return nil
}

func validateItem(itemType flow.RteFlowItemType, itemName string, spec, item *any.Any) error {
	if spec == nil && itemName != "spec" {
		return fmt.Errorf("%s spec must be specified", itemType)
	}

	switch itemType {
	case flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_ETH:
		return validateRteFlowItemEth(itemName, spec, item)
	case flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_VLAN:
		return validateRteFlowItemVlan(itemName, spec, item)
	case flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_IPV4:
		return validateRteFlowItemIpv4(itemName, spec, item)
	case flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_UDP:
		return validateRteFlowItemUdp(itemName, spec, item)
	case flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_PPPOES,
		flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_PPPOED:
		return validateRteFlowItemPppoe(itemName, spec, item)
	case flow.RteFlowItemType_RTE_FLOW_ITEM_TYPE_PPPOE_PROTO_ID:
		return validateRteFlowItemPppoeProtoId(itemName, spec, item)
	default:
		nodeflowconfiglog.Info("validating other flow item", "type", itemType)
		return nil
	}
}

func validateFlowAttr(attributes *FlowAttr) error {
	// validate flow attribute
	attr := &flow.RteFlowAttr{
		Group:    attributes.Group,
		Priority: attributes.Priority,
		Ingress:  attributes.Ingress,
		Egress:   attributes.Egress,
		Transfer: attributes.Transfer,
		Reserved: attributes.Reserved,
	}

	if err := validateRteFlowAttr(attr); err != nil {
		return err
	}

	return nil
}

func validateRteFlowAttr(attr *flow.RteFlowAttr) error {
	if attr.Ingress > 0x1 {
		return fmt.Errorf("invalid attr.ingress: '%x', must be of value {0,1}", attr.Ingress)
	}
	if attr.Egress > 0x1 {
		return fmt.Errorf("invalid attr.egress: '%x', must be of value {0,1}", attr.Egress)
	}
	if attr.Transfer > 0x1 {
		return fmt.Errorf("invalid attr.transfer: '%x', must be of value {0,1}", attr.Transfer)
	}

	return nil
}
