// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flowconfig

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"
	flowapi "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
)

// ClusterFlowConfigReconciler reconciles a ClusterFlowConfig object
type ClusterFlowConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=clusterflowconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=clusterflowconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=clusterflowconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterFlowConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ClusterFlowConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cfcLogger := r.Log.WithValues("clusterflowconfig", req.NamespacedName)
	cfcLogger.Info("Reconciling ClusterFlowConfig")

	instance := &flowconfigv1.ClusterFlowConfig{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			cfcLogger.Info("ClusterFlowConfig resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		cfcLogger.Error(err, "Failed to get ClusterFlowConfig.")
		return ctrl.Result{}, err
	}

	err = r.syncClusterConfigForNodes(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ClusterFlowConfigReconciler) syncClusterConfigForNodes(ctx context.Context, instance *flowconfigv1.ClusterFlowConfig) error {

	nodeToNodeFlowConfig := make(map[string]*flowconfigv1.NodeFlowConfig) // placeholder for node name to it's NodeFlowConfig API object

	// 1. Get Pod list from PodSelector in ClusterFlowConfig instance
	podList, err := r.getPodsForPodSelector(ctx, instance)
	if err != nil {
		return err
	}

	// 2. Loop over all Pods from podList
	if podList != nil {
		for _, pod := range podList.Items {
			// 2.2. Get nodeName from Pod
			nodeName := pod.Spec.NodeName
			if nodeName != "" {
				nodeFlowConfig, err := r.getNodeFlowConfig(nodeName, nodeToNodeFlowConfig)
				if err != nil {
					// Log error
				}

				// 2.4. Update NodeFlowConfig spec for a given pod from ClusterFlowConfig instance
				if err := r.updateNodeFlowConfigSpec(pod, nodeFlowConfig, instance); err != nil {
					// Log error
				}
				// 2.5. Add NodeFlowConfig to nodeToNodeFlowConfig map for that node
				nodeToNodeFlowConfig[nodeName] = nodeFlowConfig
			}

		}
	}
	// 3. Create/Update all NodeFlowConfig from nodeToNodeFlowConfig map

	return nil
}

func (r *ClusterFlowConfigReconciler) getNodeFlowConfig(nodeName string, nodeToNodeFlowConfig map[string]*flowconfigv1.NodeFlowConfig) (*flowconfigv1.NodeFlowConfig, error) {

	// 2.3.1. Get NodeFlowConfig from nodeToNodeFlowConfig if it exists
	nodeFlowConfig, ok := nodeToNodeFlowConfig[nodeName]
	if !ok {
		// 2.3.2. If not found Get NodeFlowConfig from K8s APIServer for that Node
		nodeFlowConfig = &flowconfigv1.NodeFlowConfig{}

		// [TO-DO]
		// The namespace is hardcoded for now. We need to get this namespace from Controller manager Pod(prob from ENV).
		// assuming all NodeFlowConfig objects will be created in that same namespace only.
		nodeFlowConfigNamespace := "intel-ethernet-operator-system"
		nameSpcedName := types.NamespacedName{
			Namespace: nodeFlowConfigNamespace,
			Name:      nodeName,
		}
		err := r.Get(context.TODO(), nameSpcedName, nodeFlowConfig)
		if err != nil {
			if errors.IsNotFound(err) {
				// 2.3.3. Not found in API server; create a new instance
				nodeFlowConfig.Name = nodeName
				nodeFlowConfig.Namespace = nodeFlowConfigNamespace
			} else {
				// Other problems getting nodeFlowConfig CR from API server. Log this and return error
				return nil, err
			}
		}

	}
	return nodeFlowConfig, nil
}

func (r *ClusterFlowConfigReconciler) getPodsForPodSelector(ctx context.Context, instance *flowconfigv1.ClusterFlowConfig) (*corev1.PodList, error) {
	selector, err := metav1.LabelSelectorAsSelector(instance.Spec.PodSelector)
	if err != nil {
		return nil, err
	}
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.Namespace),
		client.MatchingLabelsSelector{Selector: selector},
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		return nil, err
	}
	return podList, nil
}

func (r *ClusterFlowConfigReconciler) updateNodeFlowConfigSpec(pod corev1.Pod, nodeFlowConfig *flowconfigv1.NodeFlowConfig, instance *flowconfigv1.ClusterFlowConfig) error {

	if nodeFlowConfig.Spec.Rules == nil {
		nodeFlowConfig.Spec.Rules = make([]*flowconfigv1.FlowRules, 0)
	}
	for _, rule := range instance.Spec.Rules {
		newRule := &flowconfigv1.FlowRules{}
		newRule.Pattern = rule.DeepCopy().Pattern
		newRule.Attr = rule.Attr
		// newRule.PortId = portID // Cannot get portID for now; need to get this based on VF ID in the action
		newRule.PortId = 0 // Temporary hard-coded value for testing

		actions := r.getNodeActionsFromClusterActions(rule.Action)
		newRule.Action = actions

	}

	return nil // [NEEDS-UPDATE]
}

// Convert ClusterFlowAction to FlowAction for NodeFlowConfig
func (r *ClusterFlowConfigReconciler) getNodeActionsFromClusterActions(action []*flowconfigv1.ClusterFlowAction) []*flowconfigv1.FlowAction {
	nodeActions := make([]*flowconfigv1.FlowAction, 0)
	for _, act := range action {
		actType := act.Type
		_ = act.Conf

		nodeAction := &flowconfigv1.FlowAction{}

		// If Action Type is custom ClusterFlowConfigAction we convert that to NodeFlowConfigAction and associated 'Conf'
		if actType.String() == "to-pod-interface" {
			nodeAction = r.getNodeActionForPodInterface()
		}

		nodeActions = append(nodeActions, nodeAction)
	}

	// append END action at the end of the action list
	nodeActions = append(nodeActions, &flowconfigv1.FlowAction{
		Type: flowapi.RteFlowActionType_RTE_FLOW_ACTION_TYPE_END.String(),
	})

	return nodeActions
}

func (r *ClusterFlowConfigReconciler) getNodeActionForPodInterface() *flowconfigv1.FlowAction {
	flowAction := &flowconfigv1.FlowAction{
		Type: flowapi.RteFlowActionType_RTE_FLOW_ACTION_TYPE_VF.String(),
	}

	vfConf := &flowapi.RteFlowActionVf{Id: 0} // [HARD-CODED value for testing ]
	if rawBytes, err := json.Marshal(vfConf); err == nil {
		flowAction.Conf = &runtime.RawExtension{Raw: rawBytes}
	}

	return flowAction
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterFlowConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&flowconfigv1.ClusterFlowConfig{}).
		Complete(r)
}
