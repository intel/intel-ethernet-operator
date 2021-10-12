// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	"fmt"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/utils"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var nodeflowconfiglog = logf.Log.WithName("nodeflowconfig-resource")

func (r *NodeFlowConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/validate-flowconfig-intel-com-v1-nodeflowconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=flowconfig.intel.com,resources=nodeflowconfigs,verbs=create;update,versions=v1,name=vnodeflowconfig.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &NodeFlowConfig{}

func validate(rules *FlowRules) error {
	// validate flow patterns
	for i, item := range rules.Pattern {

		// validate single RteFlowItem spec
		if err := validateRteFlowItem(item); err != nil {
			return fmt.Errorf("pattern[%d] invalid: %v", i, err)
		}

		// ensure that last action is RTE_FLOW_ITEM_TYPE_END
		if i == len(rules.Pattern)-1 {
			if item.Type != "RTE_FLOW_ITEM_TYPE_END" {
				return fmt.Errorf("invalid: last pattern must be RTE_FLOW_ITEM_TYPE_END")
			}
		}
	}

	// validate flow actions
	for i, action := range rules.Action {
		rteFlowAction := new(flow.RteFlowAction)

		val, ok := flow.RteFlowActionType_value[action.Type]
		if !ok {
			return fmt.Errorf("action[%d] invalid: unkown type: %v", i, action.Type)
		}
		actionType := flow.RteFlowActionType(val)
		rteFlowAction.Type = actionType

		if action.Conf != nil {
			var err error
			rteFlowAction.Conf, err = utils.GetFlowActionAny(action.Type, action.Conf.Raw)
			if err != nil {
				return fmt.Errorf("error: %s", err)
			}
		} else {
			rteFlowAction.Conf = nil
		}

		// validate single RteFlowAction spec
		if err := validateRteFlowAction(rteFlowAction); err != nil {
			return err
		}

		// ensure that last action is RTE_FLOW_ACTION_TYPE_END
		if i == len(rules.Action)-1 {
			if action.Type != "RTE_FLOW_ACTION_TYPE_END" {
				return fmt.Errorf("invalid: last action must be RTE_FLOW_ACTION_TYPE_END")
			}
		}
	}

	// validate flow attribute
	attr := &flow.RteFlowAttr{
		Group:    rules.Attr.Group,
		Priority: rules.Attr.Priority,
		Ingress:  rules.Attr.Ingress,
		Egress:   rules.Attr.Egress,
		Transfer: rules.Attr.Transfer,
		Reserved: rules.Attr.Reserved,
	}

	if err := validateRteFlowAttr(attr); err != nil {
		return err
	}

	// validate port ID
	if err := validatePortId(rules.PortId); err != nil {
		return err
	}

	return nil
}

func validateRteFlowAttr(attr *flow.RteFlowAttr) error {
	if attr.Ingress > 0x1 {
		return fmt.Errorf("invalid attr.ingress (%x), must be of value {0,1}", attr.Ingress)
	}
	if attr.Egress > 0x1 {
		return fmt.Errorf("invalid attr.egress (%x), must be of value {0,1}", attr.Egress)
	}
	if attr.Transfer > 0x1 {
		return fmt.Errorf("invalid attr.transfer (%x), must be of value {0,1}", attr.Transfer)
	}

	return nil
}

func validatePortId(id uint32) error {
	// TODO: Port ID validation
	return nil
}

func validateRteFlowItem(item *FlowItem) error {
	rteFlowItem := new(flow.RteFlowItem)

	val, ok := flow.RteFlowItemType_value[item.Type]
	if !ok {
		return fmt.Errorf("invalid flow item type: %s", item.Type)
	}
	flowType := flow.RteFlowItemType(val)
	rteFlowItem.Type = flowType

	if item.Spec != nil {
		specAny, err := utils.GetFlowItemAny(item.Type, item.Spec.Raw)
		if err != nil {
			return fmt.Errorf("invalid 'spec' in pattern(type %s): %v", flowType, err)
		}
		rteFlowItem.Spec = specAny
		if err := validateItem(rteFlowItem.Type, "spec", nil, rteFlowItem.Spec); err != nil {
			return fmt.Errorf("validateItem(): error validating %s spec: %v", rteFlowItem.Type, err)
		}
	}

	if item.Last != nil {
		lastAny, err := utils.GetFlowItemAny(item.Type, item.Last.Raw)
		if err != nil {
			return fmt.Errorf("invalid 'last' in pattern(type %s): %v", flowType, err)
		}
		rteFlowItem.Last = lastAny
		if err := validateItem(rteFlowItem.Type, "last", rteFlowItem.Spec, rteFlowItem.Last); err != nil {
			return fmt.Errorf("validateItem(): error validating %s last: %v", rteFlowItem.Type, err)
		}
	}

	if item.Mask != nil {
		maskAny, err := utils.GetFlowItemAny(item.Type, item.Mask.Raw)
		if err != nil {
			return fmt.Errorf("invalid 'mask' in pattern(type %s): %v", flowType, err)
		}
		rteFlowItem.Mask = maskAny
		if err := validateItem(rteFlowItem.Type, "mask", rteFlowItem.Spec, rteFlowItem.Mask); err != nil {
			return fmt.Errorf("validateItem(): error validating %s mask: %v", rteFlowItem.Type, err)
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

func validateRteFlowAction(rteFlowAction *flow.RteFlowAction) error {
	switch rteFlowAction.GetType() {
	case flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_VF:
		return validateRteFlowActionVf(rteFlowAction.Conf)
	case flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_END,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_VOID,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_PASSTHRU,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_FLAG,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_DROP,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_PF,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_DEC_MPLS_TTL,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_DEC_NW_TTL,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_COPY_TTL_OUT,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_COPY_TTL_IN,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_POP_VLAN,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_VXLAN_DECAP,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_NVGRE_DECAP,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_MAC_SWAP,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_DEC_TTL,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_INC_TCP_SEQ,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_DEC_TCP_SEQ,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_INC_TCP_ACK,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_DEC_TCP_ACK:
		return validateRteFlowActionConfigEmpty(rteFlowAction.Conf)
	case flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_JUMP,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_MARK,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_QUEUE,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_COUNT,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_RSS,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_PHY_PORT,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_PORT_ID,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_METER,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SECURITY,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_SET_MPLS_TTL,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_SET_NW_TTL,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_PUSH_VLAN,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_SET_VLAN_VID,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_SET_VLAN_PCP,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_POP_MPLS,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_PUSH_MPLS,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_VXLAN_ENCAP,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_NVGRE_ENCAP,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_RAW_ENCAP,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_RAW_DECAP,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_IPV4_SRC,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_IPV4_DST,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_IPV6_SRC,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_IPV6_DST,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_TP_SRC,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_TP_DST,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_TTL,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_MAC_SRC,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_MAC_DST,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_TAG,
		flow.RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_META:
		nodeflowconfiglog.Info("correct action type, but validation not implemented: %s", rteFlowAction.GetType())
	default:
		return fmt.Errorf("invalid action type: %s", rteFlowAction.GetType())
	}

	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeFlowConfig) ValidateCreate() error {
	nodeflowconfiglog.Info("validate create", "name", r.Name)

	// TODO: it might be worth to check if the requested node (r.Name) exists

	spec := r.Spec
	for _, rule := range spec.Rules {
		err := validate(rule)
		if err != nil {
			return err
		}
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeFlowConfig) ValidateUpdate(old runtime.Object) error {
	nodeflowconfiglog.Info("validate update", "name", r.Name)

	spec := r.Spec
	for _, rule := range spec.Rules {
		err := validate(rule)
		if err != nil {
			return err
		}
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NodeFlowConfig) ValidateDelete() error {
	nodeflowconfiglog.Info("validate delete", "name", r.Name)
	// nothing to do on deletion
	return nil
}
