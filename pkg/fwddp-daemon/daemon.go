// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package daemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
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
	nvmupdate64e             = "./nvmupdate64e"
	nvmupdateVersionFilesize = 10
)

var (
	compatibilityMap     *CompatibilityMap
	compatMapPath        = "./devices.json"
	compatibilityWildard = "*"

	fwInstallDest            = "/workdir/nvmupdate/"
	nvmupdatePackageFilename = "nvmupdate.tar.gz"
	nvmupdate64eDirSuffix    = "E810/Linux_x64/"
	updateOutFile            = "update.xml"
	nvmupdateVersionFilename = "version.txt"

	getInventory  = GetInventory
	getIDs        = getDeviceIDs
	nvmupdateExec = runExecWithLog
	execCmd       = utils.ExecCmd

	utilsDownloadFile = utils.DownloadFile
	utilsUntar        = utils.Untar
)

type CompatibilityMap map[string]Compatibility
type Compatibility struct {
	utils.SupportedDevice
	Driver   string
	Firmware string
	DDP      []string
}

func nvmupdate64eDir(p string) string     { return path.Join(p, nvmupdate64eDirSuffix) }
func nvmupdate64eCfgPath(p string) string { return path.Join(nvmupdate64eDir(p), "nvmupdate.cfg") }
func updateResultPath(p string) string    { return path.Join(nvmupdate64eDir(p), updateOutFile) }

type UpdateConditionReason string

const (
	UpdateCondition        string                = "Updated"
	UpdateUnknown          UpdateConditionReason = "Unknown"
	UpdateInProgress       UpdateConditionReason = "InProgress"
	UpdatePostUpdateReboot UpdateConditionReason = "PostUpdateReboot"
	UpdateFailed           UpdateConditionReason = "Failed"
	UpdateNotRequested     UpdateConditionReason = "NotRequested"
	UpdateSucceeded        UpdateConditionReason = "Succeeded"
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
		log.Error(err, "failed to update EthernetNodeConfig status")
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

func LoadConfig(log logr.Logger) error {
	cmpMap := make(CompatibilityMap)
	err := utils.LoadSupportedDevices(compatMapPath, &cmpMap)
	if err != nil {
		return err
	}
	compatibilityMap = &cmpMap

	return nil
}

func NewNodeConfigReconciler(c client.Client, clientSet *clientset.Clientset, log logr.Logger,
	nodeName, ns string) (*NodeConfigReconciler, error) {

	err := LoadConfig(log)
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

	nodeConfig := ethernetv1.EthernetNodeConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, &nodeConfig); err != nil {
		if k8serrors.IsNotFound(err) {
			log.V(4).Info("not found - creating")
			return reconcile.Result{}, r.CreateEmptyNodeConfigIfNeeded(r.Client)
		}
		log.Error(err, "Get() failed")
		return reconcile.Result{}, err
	}

	postUpdateReboot := false
	condition := meta.FindStatusCondition(nodeConfig.Status.Conditions, UpdateCondition)
	if condition != nil {
		if condition.Reason == string(UpdatePostUpdateReboot) {
			// State where daemon is up again after post-firmware-update node reboot.
			log.V(4).Info("Post-update node reboot completed, finishing update...")
			postUpdateReboot = true
		} else if condition.ObservedGeneration == nodeConfig.GetGeneration() {
			log.V(4).Info("Created object was handled previously, ignoring")
			return reconcile.Result{}, nil
		}
	}

	if len(nodeConfig.Spec.Config) == 0 {
		log.V(4).Info("Nothing to do")
		r.updateCondition(&nodeConfig, metav1.ConditionFalse, UpdateNotRequested, "Inventory up to date")
		return reconcile.Result{}, nil
	}

	var updateErr error
	if !postUpdateReboot {
		r.updateCondition(&nodeConfig, metav1.ConditionFalse, UpdateInProgress, "Update started")

		inv, err := getInventory(log)
		if err != nil {
			log.Error(err, "Failed to retrieve inventory")
			r.updateCondition(&nodeConfig, metav1.ConditionFalse, UpdateFailed, err.Error())
			return reconcile.Result{}, nil
		}

		updateQueue := make(deviceUpdateQueue)
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
			r.updateCondition(&nodeConfig, metav1.ConditionFalse, UpdateFailed, updateErr.Error())
			return reconcile.Result{}, nil
		}

		// TODO:Add leader election or multi-leader/parallel operation (including node drain, cordon, etc)
		rebootRequired := false
		for pciAddr, artifacts := range updateQueue {
			if artifacts.fwPath != "" {
				err := r.updateFirmware(pciAddr, artifacts.fwPath)
				if err != nil {
					log.Error(err, "Failed to update firmware", "device", pciAddr)
					updateErr = multierr.Append(updateErr, err)
					// Skip DDP update on device
					continue
				}

				r, err := isRebootRequired(updateResultPath(artifacts.fwPath))
				if err != nil {
					log.Error(err, "Failed to extract reboot required flag from file")
					continue
				}

				if r {
					log.V(4).Info("Node reboot required to complete firmware update", "device", pciAddr)
					rebootRequired = true
				}
			}

			if artifacts.ddpPath != "" {
				// TODO: Update DDP
				continue
			}
		}

		if rebootRequired {
			log.V(2).Info("Rebooting the node...")
			r.updateCondition(&nodeConfig, metav1.ConditionFalse, UpdatePostUpdateReboot, "Post-update node reboot")
			err := r.rebootNode()
			if err != nil {
				log.Error(err, "Failed to reboot the node")
				updateErr = multierr.Append(updateErr, err)
			}
		}
	}

	//TODO: Add uncordon and release leader lock

	if updateErr != nil {
		r.updateCondition(&nodeConfig, metav1.ConditionFalse, UpdateFailed, updateErr.Error())
	} else {
		r.updateCondition(&nodeConfig, metav1.ConditionTrue, UpdateSucceeded, "Updated successfully")
	}

	log.V(2).Info("Reconciled")
	return reconcile.Result{}, nil
}

func (r *NodeConfigReconciler) prepare(config ethernetv1.DeviceNodeConfig, inv []ethernetv1.Device) (deviceUpdateArtifacts, error) {
	log := r.log.WithName("prepare")

	found := false
	dev := ethernetv1.Device{}
	for _, i := range inv {
		if i.PCIAddress == config.PCIAddress {
			found = true
			dev = i
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

	err = r.verifyCompatibility(fwPath, ddpPath, dev, config.DeviceConfig.Force)
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

func (r *NodeConfigReconciler) getFWVersion(fwPath string, dev ethernetv1.Device) (string, error) {
	log := r.log.WithName("getFWVersion")
	if fwPath == "" {
		log.V(4).Info("Firmware package not provided - retrieving version from device")
		v := strings.Split(dev.Firmware.Version, " ")
		if len(v) != 3 {
			return "", fmt.Errorf("Invalid firmware package version: %v", dev.Firmware.Version)
		}
		// Pick first element from eg: 2.40 0x80007064 1.2898.0 which is the NVM Version
		return v[0], nil

	} else {
		log.V(4).Info("Retrieving version from", "path", fwPath)
		path := filepath.Join(fwPath, nvmupdate64eDirSuffix, nvmupdateVersionFilename)
		file, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("Failed to open version file: %v", err)
		}
		defer file.Close()

		ver := make([]byte, nvmupdateVersionFilesize)
		n, err := file.Read(ver)
		if err != nil {
			return "", fmt.Errorf("Unable to read: %v", path)
		}
		// Example version.txt content: v2.40
		return strings.ReplaceAll(strings.TrimSpace(string(ver[:n])), "v", ""), nil
	}
}

func (r *NodeConfigReconciler) getDDPVersion(ddpPath string, dev ethernetv1.Device) (string, error) {
	log := r.log.WithName("getDDPVersion")
	if ddpPath == "" {
		log.V(4).Info("DDP package not provided - retrieving version from device")
		return dev.DDP.PackageName + "-" + dev.DDP.Version, nil
	} else {
		// TODO: DDP Tool currently does not allow to get package version from file
		// return ddpPath instead
		log.V(4).Info("Retrieving version from", "path", ddpPath)
		return ddpPath, nil
	}
}

func getDriverVersion(dev ethernetv1.Device) string {
	return dev.Driver + "-" + dev.DriverVersion
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
	cmd = exec.Command(nvmupdate64e, "-u", "-m", mac, "-c", nvmupdate64eCfgPath(fwPath), "-o", updateResultPath(fwPath), "-l")

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
	// TODO: Download DDP profile, extract if required and return path if successful.
	// For now, return the URL instead
	_ = log
	return config.DeviceConfig.DDPURL, nil
}

func (r *NodeConfigReconciler) verifyCompatibility(fwPath, ddpPath string, dev ethernetv1.Device, force bool) error {
	log := r.log.WithName("verifyCompatibility")

	if force {
		log.V(2).Info("Force flag provided - skipping compatibility check for", "device", dev.PCIAddress)
		return nil
	}

	fwVer, err := r.getFWVersion(fwPath, dev)
	if err != nil {
		log.Error(err, "Failed to retrieve firmware version")
		return err
	}

	ddpVer, err := r.getDDPVersion(ddpPath, dev)
	if err != nil {
		log.Error(err, "Failed to retrieve DDP version")
		return err
	}

	driverVer := getDriverVersion(dev)
	deviceIDs, err := getIDs(dev.PCIAddress, log)
	if err != nil {
		log.Error(err, "Failed to retrieve device IDs")
		return err
	}

	for _, c := range *compatibilityMap {
		if deviceMatcher(deviceIDs,
			fwVer,
			ddpVer,
			driverVer,
			c) {
			log.V(2).Info("Matching compatibility entry found", "entry", c)
			return nil
		}
	}

	return fmt.Errorf("No matching compatibility entry for %v (Ven:%v Cl:%v SubCl:%v DevID:%v Drv:%v FW:%v DDP:%v)",
		dev.PCIAddress, deviceIDs.VendorID, deviceIDs.Class, deviceIDs.SubClass, deviceIDs.DeviceID, driverVer,
		fwVer, ddpVer)
}

var deviceMatcher = func(ids DeviceIDs, fwVer, ddpVer, driverVer string, entry Compatibility) bool {
	if ids.VendorID == entry.VendorID &&
		ids.Class == entry.Class &&
		ids.SubClass == entry.SubClass &&
		ids.DeviceID == entry.DeviceID &&
		(entry.Firmware == compatibilityWildard || fwVer == entry.Firmware) &&
		(entry.Driver == compatibilityWildard || driverVer == entry.Driver) &&
		ddpVersionMatcher(ddpVer, entry.DDP) {
		return true
	}
	return false
}

var ddpVersionMatcher = func(ddpVer string, ddp []string) bool {
	if len(ddp) == 1 && ddp[0] == compatibilityWildard {
		return true
	}

	for _, d := range ddp {
		if ddpVer == d {
			return true
		}
	}
	return false
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

func (r *NodeConfigReconciler) rebootNode() error {
	log := r.log.WithName("rebootNode")
	// systemd-run command borrowed from openshift/sriov-network-operator
	_, err := execCmd([]string{"chroot", "--userspec", "0", "/",
		"systemd-run",
		"--unit", "ethernet-daemon-reboot",
		"--description", "ethernet-daemon reboot",
		"/bin/sh", "-c", "systemctl stop kubelet.service; reboot"}, log)
	return err
}
