// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flowconfig

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	resourceUtils "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"
	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/flowsets"
	flowapi "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/utils"
)

// NodeFlowConfigReconciler reconciles a NodeFlowConfig object
type NodeFlowConfigReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	nodeName   string
	flowSets   *flowsets.FlowSets
	flowClient flowapi.FlowServiceClient
}

//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=nodeflowconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=nodeflowconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=nodeflowconfigs/finalizers,verbs=update

// GetNodeFlowConfigReconciler returns an instance of NodeFlowConfigReconciler
func GetNodeFlowConfigReconciler(k8sClient client.Client, logger logr.Logger, scheme *runtime.Scheme, fs *flowsets.FlowSets,
	fc flowapi.FlowServiceClient, nodeName string) *NodeFlowConfigReconciler {

	return &NodeFlowConfigReconciler{
		Client:     k8sClient,
		Log:        logger,
		Scheme:     scheme,
		nodeName:   nodeName,
		flowSets:   fs,
		flowClient: fc,
	}
}

func (r *NodeFlowConfigReconciler) getNodeFilterPredicate() predicate.Predicate {
	// Create predicates for watching node specific object only, using nodeName
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetName() == r.nodeName
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() &&
				(e.ObjectOld.GetName() == r.nodeName || e.ObjectNew.GetName() == r.nodeName)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown && e.Object.GetName() == r.nodeName
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return e.Object.GetName() == r.nodeName
		},
	}

	return pred
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeFlowConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&flowconfigv1.NodeFlowConfig{}).
		WithEventFilter(r.getNodeFilterPredicate()).
		Complete(r)
}

// Reconcile reads that state of the cluster for a NodeFlowConfig object and makes changes based on the state read
// and what is in the NodeFlowConfig.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *NodeFlowConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling NodeFlowConfig")

	// Fetch the NodeFlowConfig instance
	instance := &flowconfigv1.NodeFlowConfig{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("NodeFlowConfig object is deleted")

			// Do reset to default config here
			if err := r.deleteAllRules(); err != nil {
				reqLogger.Info("error deleting all rules from cache")
			}
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	err = r.syncFlowConfig(instance)
	if err != nil {
		reqLogger.Info("syncPolicy returned error", "error message", err)
		// Even though we have encountered syncPolicy error we are returning error nil to avoid requeuing
		// TO-DO: log such error in object Status
	}
	return ctrl.Result{}, nil
}

func (r *NodeFlowConfigReconciler) updateStatus(instance *flowconfigv1.NodeFlowConfig) {
	statusLogger := r.Log.WithName("updateStatus()")
	err := r.Client.Status().Update(context.TODO(), instance)
	if err != nil {
		statusLogger.Error(err, "couldn't update NodeFlowConfig status")
	}
}

// SyncFlowConfig is new method to apply Policy Spec.
func (r *NodeFlowConfigReconciler) syncFlowConfig(newPolicy *flowconfigv1.NodeFlowConfig) error {
	syncLogger := r.Log.WithName("SyncFlowConfig")
	syncLogger.Info("syncing NodeFlowConfig")

	// Get DCF port list
	portList, err := r.listDCFPorts()
	if err != nil {
		syncLogger.Error(err, "unable to get DCF port info")
		return err
	}

	if !reflect.DeepEqual(newPolicy.Status.PortInfo, portList) {
		newPolicy.Status.PortInfo = portList
		r.updateStatus(newPolicy)
	}

	return r.syncRules(newPolicy)
}

func (r *NodeFlowConfigReconciler) syncRules(policyInstance *flowconfigv1.NodeFlowConfig) error {

	flowReqs := []*flowapi.RequestFlowCreate{}
	// Create FlowCreateRequests from rules in Specs
	if policyInstance.Spec.Rules != nil {
		for _, fr := range policyInstance.Spec.Rules {
			rteFlowCreateRequests, err := r.getFlowCreateRequests(fr)
			if err != nil {
				return err
			}
			flowReqs = append(flowReqs, rteFlowCreateRequests)
		}
	}
	toAdd, toDelete := r.getToAddAndDelete(flowReqs)
	return r.createAndDeleteRules(toAdd, toDelete)
}

func (r *NodeFlowConfigReconciler) createAndDeleteRules(toAdd map[string]*flowapi.RequestFlowCreate, toDelete map[string]*flowsets.FlowCreateRecord) error {

	// First, delete from delete lists, then create new rules
	if err := r.deleteRules(toDelete); err != nil {
		return err
	}
	if err := r.createRules(toAdd); err != nil {
		return err
	}

	return nil
}

func (r *NodeFlowConfigReconciler) deleteAllRules() error {

	toDelete := r.flowSets.GetCompliments([]string{})
	delAllLogger := r.Log.WithName("deleteAllRules()")
	delAllLogger.Info("deleting all existing rules from cache")

	if err := r.deleteRules(toDelete); err != nil {
		delAllLogger.Info("DCF returned error while deleting rules")
	}
	return nil
}

func (r *NodeFlowConfigReconciler) deleteRules(toDelete map[string]*flowsets.FlowCreateRecord) error {

	logger := r.Log.WithName("deleteRules()")
	for k, fr := range toDelete {
		delReq := &flowapi.RequestFlowofPort{PortId: 0, FlowId: fr.FlowID}
		logger.Info("deleting rule", "flow ID:", fr.FlowID)
		res, err := r.flowClient.Destroy(context.TODO(), delReq)
		if err != nil {
			logger.Info("DCF returned error while deleting rules", "flow ID:", fr.FlowID, "ErrorInfo:", res.ErrorInfo)
		}
		// Delete from flowSets
		r.flowSets.Delete(k)
	}

	return nil
}

func (r *NodeFlowConfigReconciler) createRules(toAdd map[string]*flowapi.RequestFlowCreate) error {

	logger := r.Log.WithName("createRules()")

	for k, req := range toAdd {
		// Validate all rules with DCF
		logger.Info("validating CreateFlowRequests", "flow request", req)
		res, err := r.flowClient.Validate(context.TODO(), req)
		if err != nil {
			logger.Info("DCF error while validating rules")
			return NewDCFError(fmt.Sprintf("error validating flow create request: %v", err))
		}

		if res.ErrorInfo != nil && res.ErrorInfo.Type != flowapi.RteFlowErrorType_RTE_FLOW_ERROR_TYPE_NONE {
			logger.Info("RTE flow error while validating rules", "ErrorInfo:", res.ErrorInfo)
			return NewRteFlowError(fmt.Sprintf("received validation error: %s", res.ErrorInfo.Mesg))
		}

		logger.Info("CreateFlowRequest is validated")
		createRes, err := r.flowClient.Create(context.TODO(), req)
		if err != nil {
			logger.Error(err, "error calling DCF Create")
			return NewDCFError(fmt.Sprintf("error creating flow rules: %v", err))
		}

		if createRes.ErrorInfo != nil && createRes.ErrorInfo.Type != flowapi.RteFlowErrorType_RTE_FLOW_ERROR_TYPE_NONE {
			logger.Info("received error from DCF response on creating rule",
				"request", req,
				"response", createRes.ErrorInfo.Mesg)
			return NewRteFlowError(fmt.Sprintf("received flow create error: %s", createRes.ErrorInfo.Mesg))
		}

		logger.Info("flow request is created")

		// Update flowSets
		r.flowSets.Add(k, createRes.FlowId, req)

	}

	return nil
}

// getToAddAndDelete returns a map of RequestFlowCreate to add in toAdd and a map of FlowCreateRecord in toDelete
func (r *NodeFlowConfigReconciler) getToAddAndDelete(flowReqs []*flowapi.RequestFlowCreate) (toAdd map[string]*flowapi.RequestFlowCreate,
	toDelete map[string]*flowsets.FlowCreateRecord) {
	toAdd = make(map[string]*flowapi.RequestFlowCreate)

	logger := r.Log.WithName("getToAddAndDelete")
	// newKeys is a placeholder for hash values from all flowRequest objects.
	// These keys will be used for look-up which older flowRequests needs to be deleted.
	newKeys := []string{}
	for _, req := range flowReqs {
		key, err := getFlowCreateHash(req)
		newKeys = append(newKeys, key)
		if err != nil {
			logger.Info("error getting flowCreateHash", "error", err)
		}
		if key != "" {
			if r.flowSets.Has(key) {
				continue
			}
			toAdd[key] = req
		}
	}

	toDelete = r.flowSets.GetCompliments(newKeys)

	return toAdd, toDelete
}

func (r *NodeFlowConfigReconciler) getFlowCreateRequests(fr *flowconfigv1.FlowRules) (*flowapi.RequestFlowCreate, error) {
	logger := r.Log.WithName("getFlowCreateRequests()")

	// TODO: consider refactoring
	rteFlowCreateRequests := new(flowapi.RequestFlowCreate)
	// 1 - Get flow patterns
	for _, item := range fr.Pattern {
		rteFlowItem := new(flowapi.RteFlowItem)

		val, ok := flowapi.RteFlowItemType_value[item.Type]
		if !ok {
			return nil, fmt.Errorf("invalid flow item type %s", item.Type)
		}
		flowType := flowapi.RteFlowItemType(val)
		rteFlowItem.Type = flowType

		if item.Spec != nil {
			// 1.1 - Get any.Any object for Spec pattern
			specAny, err := utils.GetFlowItemAny(item.Type, item.Spec.Raw)

			if err != nil {
				return nil, fmt.Errorf("error getting Spec pattern for flowtype %s : %v", flowType, err)
			}
			rteFlowItem.Spec = specAny
		}
		if item.Last != nil {
			// 1.2 - Get any.Any object for Last pattern
			lastAny, err := utils.GetFlowItemAny(item.Type, item.Last.Raw)
			if err != nil {
				return nil, fmt.Errorf("error getting Last pattern for flowtype %s : %v", flowType, err)
			}
			rteFlowItem.Spec = lastAny
		}

		if item.Mask != nil {
			// 1.3 - Get any.Any object for Mask pattern
			maskAny, err := utils.GetFlowItemAny(item.Type, item.Mask.Raw)
			if err != nil {
				return nil, fmt.Errorf("error getting Mask pattern for flowtype %s : %v", flowType, err)
			}
			rteFlowItem.Mask = maskAny
		}

		rteFlowCreateRequests.Pattern = append(rteFlowCreateRequests.Pattern, rteFlowItem)
	}

	// 2 - Get Flow actions
	var podPciAddress string
	for _, action := range fr.Action {
		rteFlowAction := new(flowapi.RteFlowAction)

		val, ok := flowapi.GetFlowActionType(action.Type)
		if !ok {
			return nil, fmt.Errorf("invalid action type %s", action.Type)
		}

		rteFlowAction.Type = flowapi.RteFlowActionType(val)
		if action.Conf != nil {
			actionAny, err := utils.GetFlowActionAny(action.Type, action.Conf.Raw)
			if err != nil {
				return nil, fmt.Errorf("error getting Spec pattern for flowtype %s : %v", actionAny, err)
			}

			// when action has PCI address we need to store it to be able to process it later, to get from it portId
			if action.Type == flowapi.RTE_FLOW_ACTION_TYPE_VFPCIADDR {
				actionObj := flowapi.RteFlowActionVfPciAddr{}

				if err := json.Unmarshal(action.Conf.Raw, &actionObj); err != nil {
					return nil, fmt.Errorf("error unmarshalling bytes %s to ptypes.Any: %v", string(action.Conf.Raw), err)
				}

				if podPciAddress == "" {
					podPciAddress = actionObj.Addr
				} else {
					logger.Info("please check CR - duplicated RTE_FLOW_ACTION_TYPE_VFPCIADDR")
				}
			}

			rteFlowAction.Conf = actionAny
		} else {
			rteFlowAction.Conf = nil
		}

		rteFlowCreateRequests.Action = append(rteFlowCreateRequests.Action, rteFlowAction)
	}

	// 3 - Get Flow attribute
	if fr.Attr != nil {
		// Copy from flowItem.Attr fields to FlowCreateRequest.Attr
		fAttr := &flowapi.RteFlowAttr{
			Group:    fr.Attr.Group,
			Priority: fr.Attr.Priority,
			Ingress:  fr.Attr.Ingress,
			Egress:   fr.Attr.Egress,
			Transfer: fr.Attr.Transfer,
			Reserved: fr.Attr.Reserved,
		}

		rteFlowCreateRequests.Attr = fAttr
	}

	// 4 - Get port information - ClusterFlowConfig controller should assing defined value to portId if not specified by user in CR
	if fr.PortId != invalidPortId {
		rteFlowCreateRequests.PortId = fr.PortId
	} else {
		portId, err := r.getPortIdFromDCFPort(podPciAddress)
		if err != nil {
			return nil, err
		}

		rteFlowCreateRequests.PortId = portId
	}

	return rteFlowCreateRequests, nil
}

// getPortIdFromDCFPort find a portId of VF that handles traffic based on such steps:
// - getting the pfName of the VF on which traffic is handled
// - getting the pfName of the trusted VF (DCF)
// - and compare those names. If matches, we can get portId from DCF.
func (r *NodeFlowConfigReconciler) getPortIdFromDCFPort(otherVfPciAddress string) (uint32, error) {
	pfName, err := resourceUtils.GetPfName(otherVfPciAddress)
	if err != nil {
		return invalidPortId, fmt.Errorf("unable to get pfName of VF that handles traffic. Err %v", err)
	}

	dcfPorts, err := r.listDCFPorts()
	if err != nil {
		return invalidPortId, fmt.Errorf("unable to get list of DCF ports. Err %v", err)
	}

	for _, dcfPort := range dcfPorts {
		dcfPfName, err := resourceUtils.GetPfName(dcfPort.PortPci)
		if err != nil {
			return invalidPortId, fmt.Errorf("unable to get pfName of VF that handles DCF. Err %v", err)
		}

		if pfName == dcfPfName {
			return dcfPort.PortId, nil
		}
	}

	return invalidPortId, fmt.Errorf("unable to find DCF port that matches to traffic. Err %v", err)
}

func (r *NodeFlowConfigReconciler) listDCFPorts() ([]flowconfigv1.PortsInformation, error) {
	dcfPortList, err := r.flowClient.ListPorts(context.TODO(), &flowapi.RequestListPorts{})
	if err != nil {
		return nil, err
	}

	portList := make([]flowconfigv1.PortsInformation, len(dcfPortList.Ports))
	for i, p := range dcfPortList.Ports {
		portList[i].PortId = p.PortId
		portList[i].PortMode = p.PortMode
		portList[i].PortPci = p.PortPci
	}
	return portList, nil
}

// RteFlowError is custom error struct for Rte flow related errors
type RteFlowError struct{ s string }

func (re *RteFlowError) Error() string {
	return re.s
}

// NewRteFlowError retuns a new instance of RteFlowError
func NewRteFlowError(msg string) error {
	return &RteFlowError{s: msg}
}

// DCFError is custom error struct for DCF gRPC related errors
type DCFError struct{ s string }

func (re *DCFError) Error() string {
	return re.s
}

// NewDCFError returns a new instance of DCFError
func NewDCFError(msg string) error {
	return &DCFError{s: msg}
}

// getFlowCreateHash returns a hash value from a RequestFlowCreate object
func getFlowCreateHash(req *flowapi.RequestFlowCreate) (string, error) {

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(reqBytes)

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
