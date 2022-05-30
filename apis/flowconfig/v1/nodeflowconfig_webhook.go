// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	"fmt"

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

//+kubebuilder:webhook:path=/validate-flowconfig-intel-com-v1-nodeflowconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=flowconfig.intel.com,resources=nodeflowconfigs,verbs=create;update,versions=v1,name=vnodeflowconfig.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &NodeFlowConfig{}

func (rules *FlowRules) validate() error {
	if err := validateFlowPatterns(rules.Pattern); err != nil {
		return err
	}

	// validate flow actions
	for i, action := range rules.Action {
		rteFlowAction := new(flow.RteFlowAction)

		val, ok := flow.GetFlowActionType(action.Type)
		if !ok {
			return fmt.Errorf("action[%d] invalid: unknown type: %v", i, action.Type)
		}
		actionType := flow.RteFlowActionType(val)
		rteFlowAction.Type = actionType

		if action.Conf != nil {
			var err error
			rteFlowAction.Conf, err = utils.GetFlowActionAnyForWebhook(action.Type, action.Conf.Raw)
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

	// validate flow attributes
	if err := validateFlowAttr(rules.Attr); err != nil {
		return err
	}

	// validate port ID
	if err := validatePortId(rules.PortId); err != nil {
		return err
	}

	return nil
}

func validatePortId(id uint32) error {
	return nil
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
	spec := r.Spec
	for _, rule := range spec.Rules {
		err := rule.validate()
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
		err := rule.validate()
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
