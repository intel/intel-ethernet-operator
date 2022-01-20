// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var clusterflowconfiglog = logf.Log.WithName("clusterflowconfig-resource")

func (r *ClusterFlowConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-flowconfig-intel-com-v1-clusterflowconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=flowconfig.intel.com,resources=clusterflowconfigs,verbs=create;update,versions=v1,name=vclusterflowconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ClusterFlowConfig{}

func (rules *ClusterFlowRule) validate() error {
	if err := validateFlowPatterns(rules.Pattern); err != nil {
		return err
	}

	if err := validateClusterFlowAction(rules.Action); err != nil {
		return err
	}

	// validate flow attribute
	if err := validateFlowAttr(rules.Attr); err != nil {
		return err
	}

	return nil
}

func validateClusterFlowAction(actions []*ClusterFlowAction) error {
	for i, action := range actions {
		switch action.Type {
		case ToPodInterface:
			{
				if action.Conf == nil {
					return fmt.Errorf("action %s at %d have empty configuration", action.Type, i)
				}

				podInterfaceName := &ToPodInterfaceConf{}

				if err := json.Unmarshal(action.Conf.Raw, podInterfaceName); err != nil {
					return fmt.Errorf("unable to unmarshal action %s at %d raw data %v", action.Type.String(), i, err)
				}

				if podInterfaceName.NetInterfaceName == "" {
					return fmt.Errorf("POD network interface name cannot be empty action %s at %d", action.Type.String(), i)
				}
			}
		default:
			return fmt.Errorf("invalid action type: %s at %d", action.Type.String(), i)
		}
	}

	return nil
}

func validatePodSelector(podSelector *metav1.LabelSelector) error {
	if podSelector == nil {
		return fmt.Errorf("PodSelector is undefined, please add it. Note: to select all pods use empty selector")
	}

	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterFlowConfig) ValidateCreate() error {
	clusterflowconfiglog.Info("validate create", "name", r.Name)

	spec := r.Spec
	for _, rule := range spec.Rules {
		if err := rule.validate(); err != nil {
			return err
		}
	}

	if err := validatePodSelector(spec.PodSelector); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterFlowConfig) ValidateUpdate(old runtime.Object) error {
	clusterflowconfiglog.Info("validate update", "name", r.Name)

	spec := r.Spec
	for _, rule := range spec.Rules {
		if err := rule.validate(); err != nil {
			return err
		}
	}

	if err := validatePodSelector(spec.PodSelector); err != nil {
		return err
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ClusterFlowConfig) ValidateDelete() error {
	clusterflowconfiglog.Info("validate delete", "name", r.Name)

	// nothing to do on deletion
	return nil
}
