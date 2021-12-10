// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package fwddp_manager

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"
)

// EthernetClusterConfigReconciler reconciles a EthernetClusterConfig object
type EthernetClusterConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var NAMESPACE = os.Getenv("ETHERNET_NAMESPACE")

//+kubebuilder:rbac:groups=ethernet.intel.com,resources=ethernetclusterconfigs;ethernetnodeconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ethernet.intel.com,resources=ethernetclusterconfigs/status;ethernetnodeconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ethernet.intel.com,resources=ethernetclusterconfigs/finalizers;ethernetnodeconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=machineconfiguration.openshift.io,resources=machineconfigs,verbs=create;get
//+kubebuilder:rbac:groups="",resources=nodes,verbs=list;watch
//+kubebuilder:rbac:groups=apps,resources=daemonsets;deployments;deployments/finalizers,verbs=*
//+kubebuilder:rbac:groups="",resources=namespaces;serviceaccounts;configmaps,verbs=*
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings;clusterroles;clusterrolebindings,verbs=*

func (r *EthernetClusterConfigReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ethernetclusterconfig", req.NamespacedName)
	log.V(2).Info("Reconciling EthernetClusterConfig")

	// Get EthernetClusterConfigList
	clusterConfigs := &ethernetv1.EthernetClusterConfigList{}
	err := r.List(context.TODO(), clusterConfigs, client.InNamespace(NAMESPACE))
	if err != nil {
		log.Error(err, "Failed to list EthernetClusterConfigs")
		return ctrl.Result{}, err
	}

	// Get nodes where intel etherenet devices were discovered
	nodes := &corev1.NodeList{}
	labels := &client.MatchingLabels{"ethernet.intel.com/intel-ethernet-present": ""}
	err = r.List(context.TODO(), nodes, labels)
	if err != nil {
		log.Error(err, "Failed to list Nodes")
		return ctrl.Result{}, err
	}

	clusterConfigurationMatcher := createClusterConfigMatcher(r.getOrInitializeEthernetNodeConfig, log)
	for _, node := range nodes.Items {
		configurationContext, err := clusterConfigurationMatcher.match(node, clusterConfigs.Items)
		if err != nil {
			log.Error(err, "failed to match EthernetClusterConfig(s) to a node", "node", node.Name)
			continue
		}
		if err := r.synchronizeNodeConfigSpec(configurationContext); err != nil {
			log.Error(err, "failed to create/update NodeConfig", "node", node.Name)
			continue
		}
	}

	return ctrl.Result{}, err
}

type DeviceConfigContext map[string]ethernetv1.EthernetClusterConfig
type NodeConfigurationCtx func() (ethernetv1.EthernetNodeConfig, DeviceConfigContext)
type nodeConfigProvider func(nodeName string) (*ethernetv1.EthernetNodeConfig, error)
type clusterConfigMatcher struct {
	getNodeConfig nodeConfigProvider
	log           logr.Logger
}

func createClusterConfigMatcher(ap nodeConfigProvider, l logr.Logger) *clusterConfigMatcher {
	return &clusterConfigMatcher{
		getNodeConfig: ap,
		log:           l,
	}
}

func (pm *clusterConfigMatcher) match(node corev1.Node, allConfigs []ethernetv1.EthernetClusterConfig) (NodeConfigurationCtx, error) {

	nodePolicies := matchConfigsForNode(&node, allConfigs)
	nodeConfig, err := pm.getNodeConfig(node.Name)
	if err != nil {
		pm.log.Error(err, "fail when reading EthernetNodeConfig", "name", node.Name)
		return nil, err
	}

	deviceConfigContext := pm.prepareDeviceConfigContext(nodeConfig, nodePolicies)
	return pm.prepareNodeConfigContext(*nodeConfig, deviceConfigContext), nil
}

func (pm *clusterConfigMatcher) prepareNodeConfigContext(nc ethernetv1.EthernetNodeConfig, dc DeviceConfigContext) NodeConfigurationCtx {
	return func() (ethernetv1.EthernetNodeConfig, DeviceConfigContext) { return nc, dc }
}

func (pm *clusterConfigMatcher) prepareDeviceConfigContext(nodeConfig *ethernetv1.EthernetNodeConfig, configs []ethernetv1.EthernetClusterConfig) DeviceConfigContext {
	deviceConfigContext := make(DeviceConfigContext)
	for _, current := range configs {
		selector := current.Spec.DeviceSelector
		for _, device := range nodeConfig.Status.Devices {
			if selector.Matches(device) {
				if _, ok := deviceConfigContext[device.PCIAddress]; !ok {
					deviceConfigContext[device.PCIAddress] = current
					continue
				}

				previous := deviceConfigContext[device.PCIAddress]
				switch {
				case current.Spec.Priority > previous.Spec.Priority: //override with higher prioritized config
					deviceConfigContext[device.PCIAddress] = current
				case current.Spec.Priority == previous.Spec.Priority: //multiple configs with same priority; drop older one
					//TODO: Update Timestamp would be better than CreationTime
					if current.CreationTimestamp.After(previous.CreationTimestamp.Time) {
						pm.log.V(2).
							Info("Dropping older ClusterConfig",
								"Node", nodeConfig.Name,
								"SriovFecClusterConfig", previous.Name,
								"Priority", previous.Spec.Priority,
								"CreationTimestamp", previous.CreationTimestamp.String())

						deviceConfigContext[device.PCIAddress] = current
					}

				case current.Spec.Priority < previous.Spec.Priority: //drop current with lower priority
					pm.log.V(2).
						Info("Dropping low prioritized ClusterConfig",
							"node", nodeConfig.Name,
							"SriovFecClusterConfig", current.Name,
							"priority", current.Spec.Priority)
				}
			}
		}
	}
	return deviceConfigContext
}

func matchConfigsForNode(node *corev1.Node, allConfigs []ethernetv1.EthernetClusterConfig) (nodeConfigs []ethernetv1.EthernetClusterConfig) {
	nodeLabels := labels.Set(node.Labels)
	for _, config := range allConfigs {
		nodeSelector := labels.Set(config.Spec.NodeSelector)
		if nodeSelector.AsSelector().Matches(nodeLabels) {
			nodeConfigs = append(nodeConfigs, config)
		}
	}
	return
}

func (r *EthernetClusterConfigReconciler) getOrInitializeEthernetNodeConfig(name string) (*ethernetv1.EthernetNodeConfig, error) {
	nc := new(ethernetv1.EthernetNodeConfig)
	if err := r.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: NAMESPACE}, nc); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		nc.Name = name
		nc.Namespace = NAMESPACE
		nc.Spec.Config = []ethernetv1.DeviceNodeConfig{}
	}
	return nc, nil
}

func (r *EthernetClusterConfigReconciler) synchronizeNodeConfigSpec(ncc NodeConfigurationCtx) error {
	copyWithEmptySpec := func(nc ethernetv1.EthernetNodeConfig) *ethernetv1.EthernetNodeConfig {
		newNC := nc.DeepCopy()
		newNC.Spec = ethernetv1.EthernetNodeConfigSpec{}
		return newNC
	}

	currentNodeConfig, deviceConfigContext := ncc()
	newNodeConfig := copyWithEmptySpec(currentNodeConfig)
	for pciAddress, cc := range deviceConfigContext {
		dnc := ethernetv1.DeviceNodeConfig{PCIAddress: pciAddress}
		dnc.DeviceConfig = cc.Spec.DeviceConfig
		newNodeConfig.Spec.Config = append(newNodeConfig.Spec.Config, dnc)
		newNodeConfig.Spec.DrainSkip = newNodeConfig.Spec.DrainSkip || cc.Spec.DrainSkip
		newNodeConfig.Spec.ForceReboot = newNodeConfig.Spec.ForceReboot || cc.Spec.ForceReboot
	}

	switch {
	case len(newNodeConfig.Spec.Config) == 0 && len(currentNodeConfig.Spec.Config) != 0:
		return r.Delete(context.TODO(), &currentNodeConfig)
	default:
		if !equality.Semantic.DeepDerivative(newNodeConfig.Spec, currentNodeConfig.Spec) {
			return r.Update(context.TODO(), newNodeConfig)
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EthernetClusterConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ethernetv1.EthernetClusterConfig{}).
		Complete(r)
}
