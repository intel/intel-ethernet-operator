// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package daemon

import (
	"context"

	"github.com/go-logr/logr"
	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"
	"github.com/otcshare/intel-ethernet-operator/pkg/utils"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	supportedDevices utils.SupportedDevices
	compatMapPath    = "./compat.json"
	getInventory     = GetInventory
)

type UpdateConditionReason string

const (
	UpdateCondition    string                = "Updated"
	UpdateUnknown      UpdateConditionReason = "Unknown"
	UpdateInProgress   UpdateConditionReason = "InProgress"
	UpdateFailed       UpdateConditionReason = "Failed"
	UpdateNotRequested UpdateConditionReason = "NotRequested"
	UpdateSucceeded    UpdateConditionReason = "Succeeded"
)

func (r *NodeConfigReconciler) updateCondition(nc *ethernetv1.EthernetNodeConfig, status metav1.ConditionStatus,
	reason UpdateConditionReason, msg string) {
	log := r.log.WithName("updateCondition")
	c := metav1.Condition{
		Type:               UpdateCondition,
		Status:             status,
		Reason:             string(reason),
		Message:            msg,
		ObservedGeneration: nc.GetGeneration(),
	}
	if err := r.updateStatus(nc, []metav1.Condition{c}); err != nil {
		log.Error(err, "failed to update EthernetNodeConfig condition")
	}
}

func (r *NodeConfigReconciler) updateStatus(nc *ethernetv1.EthernetNodeConfig, c []metav1.Condition) error {
	log := r.log.WithName("updateStatus")

	inv, err := getInventory(log)
	if err != nil {
		log.Error(err, "failed to obtain inventory for the node")
		return err
	}
	nodeStatus := ethernetv1.EthernetNodeConfigStatus{Devices: inv}

	for _, condition := range c {
		meta.SetStatusCondition(&nodeStatus.Conditions, condition)
	}

	nc.Status = nodeStatus
	if err := r.Status().Update(context.Background(), nc); err != nil {
		log.Error(err, "failed to update EthernetFecNode status")
		return err
	}

	return nil
}

type NodeConfigReconciler struct {
	client.Client
	log       logr.Logger
	nodeName  string
	namespace string
}

func NewNodeConfigReconciler(c client.Client, clientSet *clientset.Clientset, log logr.Logger,
	nodeName, ns string) (*NodeConfigReconciler, error) {

	var err error
	supportedDevices, err = utils.LoadSupportedDevices(compatMapPath)
	if err != nil {
		return nil, err
	}

	return &NodeConfigReconciler{
		Client:    c,
		log:       log,
		nodeName:  nodeName,
		namespace: ns,
	}, nil
}

func (r *NodeConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ethernetv1.EthernetNodeConfig{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				_, ok := e.Object.(*ethernetv1.EthernetNodeConfig)
				if !ok {
					r.log.V(2).Info("Failed to convert e.Object to ethernetv1.EthernetNodeConfig", "e.Object", e.Object)
					return false
				}
				return true

			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				if e.ObjectOld.GetGeneration() == e.ObjectNew.GetGeneration() {
					r.log.V(4).Info("Update ignored, generation unchanged")
					return false
				}
				return true
			},
		}).
		Complete(r)
}

func (r *NodeConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithName("Reconcile").WithValues("namespace", req.Namespace, "name", req.Name)
	if req.Namespace != r.namespace {
		log.V(4).Info("unexpected namespace - ignoring", "expected namespace", r.namespace)
		return reconcile.Result{}, nil
	}

	if req.Name != r.nodeName {
		log.V(4).Info("CR intended for another node - ignoring", "expected name", r.nodeName)
		return reconcile.Result{}, nil
	}

	nodeConfig := &ethernetv1.EthernetNodeConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, nodeConfig); err != nil {
		if k8serrors.IsNotFound(err) {
			log.V(4).Info("not found - creating")
			return reconcile.Result{}, r.CreateEmptyNodeConfigIfNeeded(r.Client)
		}
		log.Error(err, "Get() failed")
		return reconcile.Result{}, err
	}

	if len(nodeConfig.Spec.Config) == 0 {
		log.V(4).Info("Nothing to do")
		r.updateCondition(nodeConfig, metav1.ConditionFalse, UpdateNotRequested, "Inventory up to date")
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

func (r *NodeConfigReconciler) CreateEmptyNodeConfigIfNeeded(c client.Client) error {
	log := r.log.WithName("CreateEmptyNodeConfigIfNeeded").WithValues("name", r.nodeName, "namespace", r.namespace)

	nodeConfig := &ethernetv1.EthernetNodeConfig{}
	err := c.Get(context.Background(),
		client.ObjectKey{
			Name:      r.nodeName,
			Namespace: r.namespace,
		},
		nodeConfig)

	if err == nil {
		log.V(4).Info("already exists")
		return nil
	}

	if !k8serrors.IsNotFound(err) {
		return err
	}

	log.V(2).Info("not found - creating")

	nodeConfig = &ethernetv1.EthernetNodeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.nodeName,
			Namespace: r.namespace,
		},
		Spec: ethernetv1.EthernetNodeConfigSpec{
			Config: []ethernetv1.DeviceNodeConfig{},
		},
	}

	if createErr := c.Create(context.Background(), nodeConfig); createErr != nil {
		log.Error(createErr, "failed to create")
		return createErr
	}

	updateErr := c.Status().Update(context.Background(), nodeConfig)
	if updateErr != nil {
		log.Error(updateErr, "failed to update status")
	}
	return updateErr
}
