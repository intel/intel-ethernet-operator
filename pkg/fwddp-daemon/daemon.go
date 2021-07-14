// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package daemon

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"syscall"

	"github.com/go-logr/logr"
	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"
	"github.com/otcshare/intel-ethernet-operator/pkg/utils"
	"go.uber.org/multierr"
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

const (
	nvmupdate64e = "./nvmupdate64e"
)

var (
	supportedDevices utils.SupportedDevices
	compatMapPath    = "./compat.json"

	fwInstallDest            = "/workdir/nvmupdate/"
	nvmupdatePackageFilename = "nvmupdate.tar.gz"
	nvmupdate64eDirSuffix    = "E810/Linux_x64/"

	getInventory  = GetInventory
	nvmupdateExec = runExecWithLog

	utilsDownloadFile = utils.DownloadFile
	utilsUntar        = utils.Untar
)

func nvmupdate64eDir(p string) string     { return path.Join(p, nvmupdate64eDirSuffix) }
func nvmupdate64eCfgPath(p string) string { return path.Join(nvmupdate64eDir(p), "nvmupdate.cfg") }

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

type deviceUpdateArtifacts struct {
	fwPath  string
	ddpPath string
}
type deviceUpdateQueue map[string](deviceUpdateArtifacts)

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

	r.updateCondition(nodeConfig, metav1.ConditionFalse, UpdateInProgress, "Update started")

	updateQueue := make(deviceUpdateQueue)
	// main loop that iterates over all configs in node CR

	inv, err := getInventory(log)
	if err != nil {
		log.Error(err, "Failed to retrieve inventory")
		r.updateCondition(nodeConfig, metav1.ConditionFalse, UpdateFailed, err.Error())
		return reconcile.Result{}, nil
	}

	var updateErr error
	for _, deviceConfig := range nodeConfig.Spec.Config {
		artifacts, err := r.prepare(deviceConfig, inv)
		if err != nil {
			log.Error(err, "Failed to prepare artifacts for", "device", deviceConfig.PCIAddress)
			updateErr = multierr.Append(updateErr, err)
			continue
		}
		updateQueue[deviceConfig.PCIAddress] = artifacts
	}

	if len(updateQueue) == 0 {
		r.updateCondition(nodeConfig, metav1.ConditionFalse, UpdateFailed, updateErr.Error())
		return reconcile.Result{}, nil
	}

	// TODO:Add leader election or multi-leader/parallel operation (including node drain, cordon, etc)
	for pciAddr, artifacts := range updateQueue {
		if artifacts.fwPath != "" {
			err := r.updateFirmware(pciAddr, artifacts.fwPath)
			if err != nil {
				log.Error(err, "Failed to to update firmware", "device", pciAddr)
				updateErr = multierr.Append(updateErr, err)
				// Skip DDP update on device
				continue
			}
		}

		if artifacts.ddpPath != "" {
			// TODO: Update DDP
			continue
		}
	}
	//TODO: Add node power-cycle and uncordon

	if updateErr != nil {
		r.updateCondition(nodeConfig, metav1.ConditionFalse, UpdateFailed, updateErr.Error())
	} else {
		r.updateCondition(nodeConfig, metav1.ConditionTrue, UpdateSucceeded, "Updated successfully")
	}

	log.V(2).Info("Reconciled")
	return reconcile.Result{}, nil
}

func (r *NodeConfigReconciler) prepare(config ethernetv1.DeviceNodeConfig, inv []ethernetv1.Device) (deviceUpdateArtifacts, error) {
	log := r.log.WithName("prepare")

	found := false
	for _, i := range inv {
		if i.PCIAddress == config.PCIAddress {
			found = true
			break
		}
	}

	if !found {
		return deviceUpdateArtifacts{}, fmt.Errorf("Device %v not found", config.PCIAddress)
	}

	fwPath, err := r.prepareFirmware(config)
	if err != nil {
		log.Error(err, "Failed to prepare firmware")
		return deviceUpdateArtifacts{}, err
	}

	ddpPath, err := r.prepareDDP(config)
	if err != nil {
		log.Error(err, "Failed to prepare DDP")
		return deviceUpdateArtifacts{}, err
	}

	err = r.verifyCompatibility(fwPath, ddpPath, config.DeviceConfig.Force)
	if err != nil {
		log.Error(err, "Failed to verify compatibility")
		return deviceUpdateArtifacts{}, err
	}

	return deviceUpdateArtifacts{fwPath, ddpPath}, nil
}

func (r *NodeConfigReconciler) prepareFirmware(config ethernetv1.DeviceNodeConfig) (string, error) {
	log := r.log.WithName("prepareFirmware")

	if config.DeviceConfig.FWURL == "" {
		log.V(4).Info("Empty FWURL")
		return "", nil
	}

	targetPath := path.Join(fwInstallDest, config.PCIAddress)

	err := utils.CreateFolder(targetPath, log)
	if err != nil {
		return "", err
	}

	log.V(4).Info("Downloading", "url", config.DeviceConfig.FWURL)
	err = utilsDownloadFile(path.Join(targetPath, nvmupdatePackageFilename), config.DeviceConfig.FWURL,
		config.DeviceConfig.FWChecksum, log)
	if err != nil {
		return "", err
	}

	log.V(4).Info("File downloaded - extracting")
	err = utilsUntar(path.Join(targetPath, nvmupdatePackageFilename), targetPath, log)
	if err != nil {
		return "", err
	}

	return targetPath, nil
}

func (r *NodeConfigReconciler) updateFirmware(pciAddr, fwPath string) error {
	log := r.log.WithName("updateFirmware")

	rootAttr := &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: 0, Gid: 0},
	}
	// Call nvmupdate64 -i first to refresh devices
	log.V(2).Info("Refreshing nvmupdate inventory")
	cmd := exec.Command(nvmupdate64e, "-i")
	cmd.SysProcAttr = rootAttr
	cmd.Dir = nvmupdate64eDir(fwPath)
	err := nvmupdateExec(cmd, log)
	if err != nil {
		return err
	}

	mac, err := getDeviceMAC(pciAddr, log)
	if err != nil {
		log.Error(err, "Failed to get MAC for", "device", pciAddr)
		return err
	}

	log.V(2).Info("Updating", "MAC", mac)
	cmd = exec.Command(nvmupdate64e, "-u", "-m", mac, "-c", nvmupdate64eCfgPath(fwPath), "-l")
	cmd.SysProcAttr = rootAttr
	cmd.Dir = nvmupdate64eDir(fwPath)
	err = nvmupdateExec(cmd, log)
	if err != nil {
		return err
	}

	return nil
}

func runExecWithLog(cmd *exec.Cmd, log logr.Logger) error {
	cmd.Stdout = &utils.LogWriter{Log: log, Stream: "stdout"}
	cmd.Stderr = &utils.LogWriter{Log: log, Stream: "stderr"}
	return cmd.Run()
}

func (r *NodeConfigReconciler) prepareDDP(config ethernetv1.DeviceNodeConfig) (string, error) {
	log := r.log.WithName("prepareDDP")
	// TODO: Download DDP profile, extract if required and return path if successful
	_ = log
	return "", nil
}

func (r *NodeConfigReconciler) verifyCompatibility(fwPath, ddpPath string, force bool) error {
	log := r.log.WithName("verifyCompatibility")
	// TODO: Get versions of proviced FW and DDP if provided (if empty, retrieve current from device);
	// compare against compatibility map; skip if force==true
	_ = log
	return nil
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
