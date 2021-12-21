// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package daemon

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"
	"github.com/otcshare/intel-ethernet-operator/pkg/utils"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dh "github.com/otcshare/intel-ethernet-operator/pkg/drainhelper"
)

const (
	requeueAfter = 15 * time.Minute
)

var (
	getInventory = GetInventory
	getIDs       = getDeviceIDs
	execCmd      = utils.ExecCmd

	downloadFile = utils.DownloadFile
	untarFile    = utils.Untar

	artifactsFolder = "/host/tmp/fwddp_artifacts/nvmupdate/"
)

type UpdateConditionReason string

const (
	UpdateCondition        string                = "Updated"
	UpdateInProgress       UpdateConditionReason = "InProgress"
	UpdatePostUpdateReboot UpdateConditionReason = "PostUpdateReboot"
	UpdateFailed           UpdateConditionReason = "Failed"
	UpdateNotRequested     UpdateConditionReason = "NotRequested"
	UpdateSucceeded        UpdateConditionReason = "Succeeded"
)

type deviceUpdateArtifacts struct {
	fwPath  string
	ddpPath string
}
type deviceUpdateQueue map[string]deviceUpdateArtifacts

type NodeConfigReconciler struct {
	client.Client
	log         logr.Logger
	drainHelper *dh.DrainHelper
	nodeNameRef types.NamespacedName
	ddpUpdater  *ddpUpdater
	fwUpdater   *fwUpdater
}

func LoadConfig() error {
	cmpMap := make(CompatibilityMap)
	err := utils.LoadSupportedDevices(compatMapPath, &cmpMap)
	if err != nil {
		return err
	}
	compatibilityMap = &cmpMap

	return nil
}

type ResourceNamePredicate struct {
	predicate.Funcs
	requiredName string
	log          logr.Logger
}

func (r ResourceNamePredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectNew.GetName() != r.requiredName {
		r.log.Info("CR intended for another node - ignoring", "expected name", r.requiredName)
		return false
	}
	return true
}

func (r ResourceNamePredicate) Create(e event.CreateEvent) bool {
	if e.Object.GetName() != r.requiredName {
		r.log.Info("CR intended for another node - ignoring", "expected name", r.requiredName)
		return false
	}
	return true
}

//returns result indicating necessity of re-queuing Reconcile after configured resyncPeriod
func requeueLater() (reconcile.Result, error) {
	return reconcile.Result{RequeueAfter: requeueAfter}, nil
}

//returns result indicating necessity of re-queuing Reconcile(...) immediately; non-nil err will be logged by controller
func requeueNowWithError(e error) (reconcile.Result, error) {
	return reconcile.Result{Requeue: true}, e
}

//returns result indicating that there is no need to Reconcile because everything is configured as expected
func doNotRequeue() (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func NewNodeConfigReconciler(c client.Client, clientSet *clientset.Clientset, log logr.Logger,
	nodeName, ns string) (*NodeConfigReconciler, error) {

	err := LoadConfig()
	if err != nil {
		return nil, err
	}

	return &NodeConfigReconciler{
		Client:      c,
		log:         log,
		drainHelper: dh.NewDrainHelper(log, clientSet, nodeName, ns),
		nodeNameRef: types.NamespacedName{
			Namespace: ns,
			Name:      nodeName,
		},
		ddpUpdater: &ddpUpdater{
			log: log,
		},
		fwUpdater: &fwUpdater{
			log: log,
		},
	}, nil
}

func (r *NodeConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ethernetv1.EthernetNodeConfig{}).
		WithEventFilter(
			predicate.And(
				ResourceNamePredicate{
					requiredName: r.nodeNameRef.Name,
					log:          r.log,
				},
				predicate.GenerationChangedPredicate{},
			),
		).
		Complete(r)
}
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
		log.Error(err, "failed to update EthernetNodeConfig status")
		return err
	}

	return nil
}

func (r *NodeConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithName("Reconcile").WithValues("namespace", req.Namespace, "name", req.Name)
	nodeConfig := &ethernetv1.EthernetNodeConfig{}

	syscall.Umask(0077)
	if err := r.Client.Get(ctx, req.NamespacedName, nodeConfig); err != nil {
		if k8serrors.IsNotFound(err) {
			log.V(4).Info("not found - creating")
			return requeueNowWithError(r.CreateEmptyNodeConfigIfNeeded(r.Client))
		}
		log.Error(err, "Get() failed")
		return requeueNowWithError(err)
	}

	condition := meta.FindStatusCondition(nodeConfig.Status.Conditions, UpdateCondition)
	if condition != nil && condition.Reason == string(UpdatePostUpdateReboot) {
		log.V(4).Info("Post-update node reboot completed, finishing update...")
		r.updateCondition(nodeConfig, metav1.ConditionTrue, UpdateSucceeded, "Updated successfully")
		log.V(2).Info("Reconciled")
		return doNotRequeue()
	}

	if len(nodeConfig.Spec.Config) == 0 {
		log.V(4).Info("Nothing to do")
		r.updateCondition(nodeConfig, metav1.ConditionTrue, UpdateNotRequested, "Inventory up to date")
		return doNotRequeue()
	}

	r.updateCondition(nodeConfig, metav1.ConditionFalse, UpdateInProgress, "Update started")

	updateQueue, err := r.prepareUpdateQueue(nodeConfig)
	if err != nil {
		r.updateCondition(nodeConfig, metav1.ConditionFalse, UpdateFailed, err.Error())
		return requeueLater()
	}

	err = r.configureNode(updateQueue, nodeConfig)
	if err != nil {
		r.updateCondition(nodeConfig, metav1.ConditionFalse, UpdateFailed, err.Error())
		return requeueLater()
	}

	r.updateCondition(nodeConfig, metav1.ConditionTrue, UpdateSucceeded, "Updated successfully")
	log.V(2).Info("Reconciled")
	return doNotRequeue()
}

func (r *NodeConfigReconciler) configureNode(updateQueue deviceUpdateQueue, nodeConfig *ethernetv1.EthernetNodeConfig) error {
	//func start
	var nodeActionErr error

	drainFunc := func(ctx context.Context) bool {
		fwReboot := false
		rebootRequired := nodeConfig.Spec.ForceReboot

		for pciAddr, artifacts := range updateQueue {
			fwReboot, nodeActionErr = r.fwUpdater.handleFWUpdate(pciAddr, artifacts.fwPath)
			if nodeActionErr != nil {
				return true
			}

			rebootRequired = rebootRequired || fwReboot

			nodeActionErr = r.ddpUpdater.handleDDPUpdate(pciAddr, nodeConfig.Spec.ForceReboot, artifacts.ddpPath)
			if nodeActionErr != nil {
				return true
			}
		}

		if rebootRequired {
			r.updateCondition(nodeConfig, metav1.ConditionFalse, UpdatePostUpdateReboot, "Post-update node reboot")
			nodeActionErr = r.rebootNode()
			return false
		}

		return true
	}
	//func end
	drainErr := r.drainHelper.Run(drainFunc, !nodeConfig.Spec.DrainSkip)

	if drainErr != nil {
		r.log.Error(drainErr, "Error during node draining")
		return drainErr
	}
	if nodeActionErr != nil {
		r.log.Error(nodeActionErr, "Error during node FW/DDP update")
		return nodeActionErr
	}

	return nil
}

func (r *NodeConfigReconciler) prepareUpdateQueue(nodeConfig *ethernetv1.EthernetNodeConfig) (deviceUpdateQueue, error) {
	inv, err := getInventory(r.log)
	if err != nil {
		return deviceUpdateQueue{}, err
	}

	err = os.RemoveAll(artifactsFolder)
	if err != nil {
		r.log.Error(err, "Failed to prepare firmware")
		return deviceUpdateQueue{}, err
	}

	updateQueue := make(deviceUpdateQueue)
	for _, deviceConfig := range nodeConfig.Spec.Config {
		artifacts, err := r.prepareArtifacts(deviceConfig, inv)
		if err != nil {
			r.log.Error(err, "Failed to prepare artifacts for", "device", deviceConfig.PCIAddress)
			return deviceUpdateQueue{}, err
		}
		updateQueue[deviceConfig.PCIAddress] = artifacts
	}
	return updateQueue, nil
}

func (r *NodeConfigReconciler) prepareArtifacts(config ethernetv1.DeviceNodeConfig, inv []ethernetv1.Device) (deviceUpdateArtifacts, error) {
	log := r.log.WithName("prepare")

	dev, err := r.findCard(config, inv)
	if err != nil {
		return deviceUpdateArtifacts{}, err
	}

	fwPath, err := r.fwUpdater.prepareFirmware(config)
	if err != nil {
		log.Error(err, "Failed to prepare firmware")
		return deviceUpdateArtifacts{}, err
	}

	ddpPath, err := r.ddpUpdater.prepareDDP(config)
	if err != nil {
		log.Error(err, "Failed to prepare DDP")
		return deviceUpdateArtifacts{}, err
	}

	err = r.verifyCompatibility(fwPath, ddpPath, dev, config.DeviceConfig.Force)
	if err != nil {
		log.Error(err, "Failed to verify compatibility")
		return deviceUpdateArtifacts{}, err
	}

	return deviceUpdateArtifacts{fwPath, ddpPath}, nil
}

func (r *NodeConfigReconciler) findCard(config ethernetv1.DeviceNodeConfig, inv []ethernetv1.Device) (ethernetv1.Device, error) {
	for _, i := range inv {
		if i.PCIAddress == config.PCIAddress {
			return i, nil
		}
	}

	return ethernetv1.Device{}, fmt.Errorf("device %v not found", config.PCIAddress)
}

func (r *NodeConfigReconciler) CreateEmptyNodeConfigIfNeeded(c client.Client) error {
	log := r.log.WithName("CreateEmptyNodeConfigIfNeeded").WithValues("name", r.nodeNameRef.Name, "namespace", r.nodeNameRef.Namespace)
	nodeConfig := &ethernetv1.EthernetNodeConfig{}
	err := c.Get(context.Background(),
		r.nodeNameRef,
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
			Name:      r.nodeNameRef.Name,
			Namespace: r.nodeNameRef.Namespace,
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

func (r *NodeConfigReconciler) rebootNode() error {
	log := r.log.WithName("rebootNode")
	// systemd-run command borrowed from openshift/sriov-network-operator
	_, err := execCmd([]string{"chroot", "--userspec", "0", "/host",
		"systemd-run",
		"--unit", "ethernet-daemon-reboot",
		"--description", "ethernet-daemon reboot",
		"/bin/sh", "-c", "systemctl stop kubelet.service; reboot"}, log)
	return err
}
