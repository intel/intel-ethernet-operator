// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flowconfig

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	nadClientTypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

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
	// one ClusterFlowConfig CR can influence on several NodeFlowConfigs, by adding one or more rules to new/existing instance of NodeFlowConfig
	// above depends on the POD selector labels defined within ClusterFlowConfig
	// This map is also usefull when ClusterFlowConfig is deleted. In normal case controller only would know the namespace and name of CR,
	// but from this map it can find all NodeFlowConfigs that were affected (have rules defined by deleted ClusterFlowConfig)
	// Will hold map[ClusterFlowConfig name] map[node where NodeFlowConfig was created][]hash to rules that were created
	Cluster2NodeRulesHashMap map[types.NamespacedName]map[types.NamespacedName][]string
}

//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=clusterflowconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=clusterflowconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=clusterflowconfigs/finalizers,verbs=update

func (r *ClusterFlowConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cfcLogger := r.Log.WithValues("Reconcile", req.NamespacedName)
	cfcLogger.Info("Reconciling ClusterFlowConfig")
	if r.Cluster2NodeRulesHashMap == nil {
		r.Cluster2NodeRulesHashMap = make(map[types.NamespacedName]map[types.NamespacedName][]string)
	}

	instance := &flowconfigv1.ClusterFlowConfig{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			err = r.cleanNodeFlowConfig(req)
			if err != nil {
				return ctrl.Result{}, err
			}

			// Return and don't requeue
			cfcLogger.Info("ClusterFlowConfig resource not found. Clean and return since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		cfcLogger.Error(err, "Failed to get ClusterFlowConfig.")
		return ctrl.Result{}, err
	}

	err = r.syncClusterConfigForNodes(ctx, instance, req)
	if err != nil {
		cfcLogger.Info("failed:", "error", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ClusterFlowConfigReconciler) cleanNodeFlowConfig(req ctrl.Request) error {
	cfcLogger := r.Log.WithName("cleanNodeFlowConfig")
	// clean NodeFlowConfig CRs that were connected with deleted ClusterFlowConfig
	if nodeConfigs, ok := r.Cluster2NodeRulesHashMap[req.NamespacedName]; ok {
		for config, hashes := range nodeConfigs {
			object := &flowconfigv1.NodeFlowConfig{}
			err := r.Client.Get(context.TODO(), client.ObjectKey{
				Namespace: config.Namespace,
				Name:      config.Name,
			}, object)

			// if object exists, let's delete removed rules
			if err == nil {
				// convert slice with hashes to map
				hashMap := map[string]bool{}
				for _, ruleHash := range hashes {
					hashMap[ruleHash] = true
				}

				index := 0
				for _, rule := range object.Spec.Rules {
					if key, err := getFlowRulesHash(rule); err == nil && !hashMap[key] {
						object.Spec.Rules[index] = rule
						index++
					}
				}

				object.Spec.Rules = object.Spec.Rules[:index]

				// send update to API
				err = r.Client.Update(context.TODO(), object)
				if err != nil {
					if strings.Contains(err.Error(), "the object has been modified; please apply your changes to the latest version and try again") {
						return err
					}

					cfcLogger.Error(err, "Unable to update NodeFlowConfig on", "object", object.Name)
				}
			}
		}
	}

	// at the end remove from map information about ClusterFlowConfig and its hashes
	delete(r.Cluster2NodeRulesHashMap, req.NamespacedName)

	return nil
}

func (r *ClusterFlowConfigReconciler) syncClusterConfigForNodes(ctx context.Context, instance *flowconfigv1.ClusterFlowConfig, req ctrl.Request) error {
	cfcLogger := r.Log.WithName("syncClusterConfigForNodes")
	nodeToNodeFlowConfig := make(map[string]*flowconfigv1.NodeFlowConfig) // placeholder for node name to it's NodeFlowConfig API object

	// 1. Get Pod list from PodSelector in ClusterFlowConfig instance
	podList, err := r.getPodsForPodSelector(ctx, instance)
	if err != nil {
		return err
	}

	if len(podList.Items) == 0 {
		cfcLogger.Info("There is no PODs with labels defined in CR labels selector")
		// there is no POD with that selector, check if there is NodeFlowConfig connected with it, if yes, remove rules from it
		return r.cleanNodeFlowConfig(req)
	}

	// 1.1 Get all ClusterFlowConfigs with same PodSelector as PODs to be able to update correctly rules within NodeFlowConfigs
	clusterFlowList, err := r.getClusterFlowConfigsForPodSelector(ctx, instance)
	if err != nil || clusterFlowList == nil {
		return err
	}

	// 2. Loop over all Pods from podList
	if podList != nil {
		for _, pod := range podList.Items {
			// 2.2. Get nodeName from Pod
			nodeName := pod.Spec.NodeName
			if nodeName != "" {
				nodeFlowConfig, err := r.getNodeFlowConfig(nodeName, instance.Namespace, nodeToNodeFlowConfig)
				if err != nil {
					cfcLogger.Info("Skipping", "instance in ns", instance.Namespace, "node", nodeName, "due to problems with getting node config:", err)
					continue
				}

				// 2.4. Update NodeFlowConfig spec for a given pod from ClusterFlowConfig instance
				if err := r.updateNodeFlowConfigSpec(&pod, nodeFlowConfig, clusterFlowList); err != nil {
					cfcLogger.Info("Skipping", "instance in ns", instance.Namespace, "node", nodeName, "due to problems with updating node config:", err)
					continue
				}

				// 2.5. Add NodeFlowConfig to nodeToNodeFlowConfig map for that node
				nodeToNodeFlowConfig[nodeName] = nodeFlowConfig
			} else {
				cfcLogger.Info("Missing information about node name", "instance in ns", instance.Namespace, "POD ", pod.ObjectMeta.Name)
			}
		}
	}

	// 3. Create/Update all NodeFlowConfig from nodeToNodeFlowConfig map
	r.createNodeFlowConfigOnNode(nodeToNodeFlowConfig)

	return nil
}

func (r *ClusterFlowConfigReconciler) createNodeFlowConfigOnNode(nodeToNodeFlowConfig map[string]*flowconfigv1.NodeFlowConfig) {
	logger := r.Log.WithName("createNodeFlowConfigOnNode")
	for nodeName, nodeConfig := range nodeToNodeFlowConfig {
		object := &flowconfigv1.NodeFlowConfig{}
		err := r.Client.Get(context.TODO(), client.ObjectKey{
			Namespace: nodeConfig.Namespace,
			Name:      nodeConfig.Name,
		}, object)

		// if object exists, we need only to update it
		if err == nil {
			err = r.Client.Update(context.TODO(), nodeConfig)
			if err != nil {
				logger.Error(err, "Unable to update NodeFlowConfig on", "node", nodeName)
			}
			continue
		}

		err = r.Client.Create(context.TODO(), nodeConfig)
		// only report that some configs cannot be created, do not break the process of creation for other nodes
		if err != nil {
			logger.Error(err, "Unable to create NodeFlowConfig on", "node", nodeName)
		}
	}
}

func (r *ClusterFlowConfigReconciler) getNodeFlowConfig(nodeName, namespace string, nodeToNodeFlowConfig map[string]*flowconfigv1.NodeFlowConfig) (*flowconfigv1.NodeFlowConfig, error) {
	// 2.3.1. Get NodeFlowConfig from nodeToNodeFlowConfig if it exists
	nodeFlowConfig, ok := nodeToNodeFlowConfig[nodeName]
	if !ok {
		// 2.3.2. If not found Get NodeFlowConfig from K8s APIServer for that Node
		nodeFlowConfig = &flowconfigv1.NodeFlowConfig{}

		// assuming all NodeFlowConfig objects will be created in that same namespace as the ClusterFlowConfig
		nameSpacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      nodeName,
		}
		err := r.Get(context.TODO(), nameSpacedName, nodeFlowConfig)
		if err != nil {
			if errors.IsNotFound(err) {
				// 2.3.3. Not found in API server; create a new instance
				nodeFlowConfig.Name = nodeName
				nodeFlowConfig.Namespace = namespace
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

// getClusterFlowConfigsForPodSelector returns all ClusterFlowConfigs that defines the same PodSelector
func (r *ClusterFlowConfigReconciler) getClusterFlowConfigsForPodSelector(ctx context.Context, instance *flowconfigv1.ClusterFlowConfig) (*flowconfigv1.ClusterFlowConfigList, error) {
	logger := r.Log.WithName("getClusterFlowConfigsForPodSelector")
	selector, err := metav1.LabelSelectorAsSelector(instance.Spec.PodSelector)
	if err != nil {
		return nil, err
	}

	configList := &flowconfigv1.ClusterFlowConfigList{}
	// TODO consider using MatchingFieldsSelector to obtain results directly from API instead of filtering it on own
	// MatchingLabelsSelector cannot be used, because CR defines labels inside SPEC section
	// MatchingFieldsSelector also involves indexer implementation
	listOpts := []client.ListOption{
		client.InNamespace(instance.Namespace),
	}
	if err = r.List(ctx, configList, listOpts...); err != nil {
		return nil, err
	}

	out := []flowconfigv1.ClusterFlowConfig{}
	for _, cr := range configList.Items {
		insideSelector, err := metav1.LabelSelectorAsSelector(cr.Spec.PodSelector)
		if err != nil {
			logger.Info("Unable to convert LabelSelectorAsSelector for CR", cr.ObjectMeta.Name)
			continue
		}

		if insideSelector.String() == selector.String() {
			out = append(out, cr)
		}
	}
	configList.Items = out

	return configList, nil
}

func (r *ClusterFlowConfigReconciler) updateNodeFlowConfigSpec(pod *corev1.Pod,
	nodeFlowConfig *flowconfigv1.NodeFlowConfig,
	clusterConfigList *flowconfigv1.ClusterFlowConfigList) error {

	if len(clusterConfigList.Items) == 0 {
		return fmt.Errorf("ClusterFlowConfig list is empty. Unable to gather rules")
	}

	// Clear existing rules - configuration is going to be recreated from scratch
	nodeFlowConfig.Spec.Rules = make([]*flowconfigv1.FlowRules, 0)

	// store all hash for rules that goes into NodeFlowConfig
	allHashes := make(map[string]bool)
	for _, clusterConfig := range clusterConfigList.Items {
		hashes := []string{}
		for _, rule := range clusterConfig.Spec.Rules {
			newRule := &flowconfigv1.FlowRules{}
			newRule.Pattern = rule.DeepCopy().Pattern
			newRule.Attr = rule.Attr
			// set PortId to invalid number, NodeFlowConfig controller based on interface name from POD selector will figure out PortId and fill it in.
			newRule.PortId = invalidPortId

			actions, err := r.getNodeActionsFromClusterActions(rule.Action, pod)
			if err != nil {
				return err
			}
			newRule.Action = actions

			if key, err := getFlowRulesHash(newRule); err == nil {
				hashes = append(hashes, key)
				if _, hashExists := allHashes[key]; hashExists {
					// avoid duplicated rules - do not append rules that are already added to NodeFlowConfig
					continue
				}

				allHashes[key] = true
			}

			nodeFlowConfig.Spec.Rules = append(nodeFlowConfig.Spec.Rules, newRule)
		}

		// add for particular ClusterFlowConfig hashes that were added to NodeFlowConfig instance
		if r.Cluster2NodeRulesHashMap[types.NamespacedName{Namespace: clusterConfig.Namespace, Name: clusterConfig.Name}] == nil {
			r.Cluster2NodeRulesHashMap[types.NamespacedName{Namespace: clusterConfig.Namespace, Name: clusterConfig.Name}] = make(map[types.NamespacedName][]string)
		}

		r.Cluster2NodeRulesHashMap[types.NamespacedName{
			Namespace: clusterConfig.Namespace,
			Name:      clusterConfig.Name,
		}][types.NamespacedName{Namespace: nodeFlowConfig.Namespace, Name: nodeFlowConfig.Name}] = hashes
	}

	return nil
}

// getClusterFlowRuleHash returns a hash value from a ClusterFlowRule object
func getFlowRulesHash(rule *flowconfigv1.FlowRules) (string, error) {
	reqBytes, err := json.Marshal(rule)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(reqBytes)

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Convert ClusterFlowAction to FlowAction for NodeFlowConfig. Returns nil when there are errors in conversion.
func (r *ClusterFlowConfigReconciler) getNodeActionsFromClusterActions(actions []*flowconfigv1.ClusterFlowAction, pod *corev1.Pod) ([]*flowconfigv1.FlowAction, error) {
	nodeActions := make([]*flowconfigv1.FlowAction, 0)

	for _, act := range actions {
		actType := act.Type

		// If Action Type is custom ClusterFlowConfigAction we convert that to NodeFlowConfigAction and associated 'Conf'
		if actType.String() == flowconfigv1.ClusterFlowActionToString(flowconfigv1.ToPodInterface) {
			var err error
			var nodeAction *flowconfigv1.FlowAction
			if nodeAction, err = r.getNodeActionForPodInterface(act.Conf, pod); err != nil {
				return nil, err
			}

			nodeActions = append(nodeActions, nodeAction)
		} else {
			// convert ClusterFlowAction to FlowAction
			nodeActions = append(nodeActions, &flowconfigv1.FlowAction{
				Type: flowapi.RteFlowActionType_name[int32(act.Type)],
				Conf: act.Conf,
			})
		}
	}

	// append END action at the end of the action list only when there is at least one action on list
	// in other case an empty list is going to be returned
	if len(nodeActions) > 0 {
		nodeActions = append(nodeActions, &flowconfigv1.FlowAction{
			Type: flowapi.RteFlowActionType_RTE_FLOW_ACTION_TYPE_END.String(),
		})
	}

	return nodeActions, nil
}

func (r *ClusterFlowConfigReconciler) getNodeActionForPodInterface(conf *runtime.RawExtension, pod *corev1.Pod) (*flowconfigv1.FlowAction, error) {
	var err error
	var pciAddr string

	interfaceName, err := getPodInterfaceNameFromRawExtension(conf)
	if err != nil {
		return nil, err
	}

	if pod == nil {
		return nil, fmt.Errorf("pod object is nil")
	}

	// verify if POD network annotations have interface name provided within Action
	podAnnotations, exists := pod.ObjectMeta.Annotations["k8s.v1.cni.cncf.io/network-status"]
	if !exists {
		return nil, fmt.Errorf("pod %s does not contains network status", pod.Name)
	}

	var networks []nadClientTypes.NetworkStatus
	err = json.Unmarshal([]byte(podAnnotations), &networks)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data for pod %s with error %s", pod.Name, err)
	}

	for _, network := range networks {
		if network.Interface == interfaceName && network.DeviceInfo.Type == "pci" {
			pciAddr = network.DeviceInfo.Pci.PciAddress

			break
		}
	}

	if pciAddr == "" {
		return nil, fmt.Errorf("pod %s network status does not contains interface %s", pod.Name, interfaceName)
	}

	// create Action for node config controller with interface PCI address
	flowAction := &flowconfigv1.FlowAction{
		Type: flowapi.RTE_FLOW_ACTION_TYPE_VFPCIADDR,
	}

	// marshal PCI address into Conf field
	vfConf := &flowapi.RteFlowActionVfPciAddr{Addr: pciAddr}
	rawBytes, err := json.Marshal(vfConf)
	if err != nil {
		return nil, err
	}

	flowAction.Conf = &runtime.RawExtension{Raw: rawBytes}

	return flowAction, nil
}

func getPodInterfaceNameFromRawExtension(conf *runtime.RawExtension) (string, error) {
	if conf == nil {
		return "", fmt.Errorf("action configuration is empty")
	}

	podInterfaceName := &flowconfigv1.ToPodInterfaceConf{}

	if err := json.Unmarshal(conf.Raw, podInterfaceName); err != nil {
		return "", fmt.Errorf("unable to unmarshal action raw data %v", err)
	}

	return podInterfaceName.NetInterfaceName, nil
}

func (r *ClusterFlowConfigReconciler) mapPodsToRequests(object client.Object) []reconcile.Request {
	logger := r.Log.WithName("mapPodsToRequests")
	reconcileRequests := make([]reconcile.Request, 0)

	pod, ok := object.(*corev1.Pod)
	if !ok {
		logger.Info("Object passed to method is not a POD type")
		return reconcileRequests
	}

	crList := &flowconfigv1.ClusterFlowConfigList{}
	if err := r.Client.List(context.Background(), crList); err != nil {
		if !errors.IsNotFound(err) {
			logger.Info("unable to fetch custom resources", err)
		}

		return reconcileRequests
	}

	// check each instance of ClusterFlowConfig PodSelector against the POD labels
	// and if labels from ClusterFlowConfig matches to labels in POD, add CR to reconcile request
	for _, instance := range crList.Items {
		var counter int
		for key, valCr := range instance.Spec.PodSelector.MatchLabels {
			valPod, ok := pod.Labels[key]
			if !ok {
				// do not check others - key does not exists
				break
			}

			if valPod != valCr {
				// do not check others - value does not match
				break
			}
			counter++
		}

		// all labels from POD matches the POD selector defined within ClusterFlowConfig -> trigger reconcile on that CR
		if len(instance.Spec.PodSelector.MatchLabels) == counter {
			reconcileRequests = append(reconcileRequests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      instance.Name,
					Namespace: instance.Namespace,
				},
			})
		}
	}

	return reconcileRequests
}

func (r *ClusterFlowConfigReconciler) getPodFilterPredicates() predicate.Predicate {
	pred := predicate.Funcs{
		// Create returns true if the Create event should be processed
		CreateFunc: func(e event.CreateEvent) bool {
			if _, ok := e.Object.(*flowconfigv1.ClusterFlowConfig); ok {
				return true
			}

			return false
		},

		// Delete returns true if the Delete event should be processed
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},

		// Update returns true if the Update event should be processed
		UpdateFunc: func(e event.UpdateEvent) bool {
			if newPod, ok := e.ObjectNew.(*corev1.Pod); ok && newPod.Status.Phase == "Running" {
				if oldPod, ok := e.ObjectOld.(*corev1.Pod); ok {
					// process event only when labels and annotations are different
					return !reflect.DeepEqual(newPod.ObjectMeta.Labels, oldPod.ObjectMeta.Labels) || !reflect.DeepEqual(newPod.ObjectMeta.Annotations, oldPod.ObjectMeta.Annotations)
				}
			}

			return false
		},

		// Generic returns true if the Generic event should be processed
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}

	return pred
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterFlowConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&flowconfigv1.ClusterFlowConfig{}).
		Watches(
			&source.Kind{Type: &corev1.Pod{}},
			handler.EnqueueRequestsFromMapFunc(r.mapPodsToRequests),
		).
		WithEventFilter(r.getPodFilterPredicates()).
		Complete(r)
}
