// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/go-logr/logr"
	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"
	"github.com/otcshare/intel-ethernet-operator/pkg/utils"
)

const (
	nvmupdate64e  = "./nvmupdate64e"
	updateOutFile = "update.xml"
)

var (
	findFw        = findFwExec
	nvmupdateExec = utils.RunExecWithLog
)

type fwUpdater struct {
	log logr.Logger
}

func (f *fwUpdater) prepareFirmware(config ethernetv1.DeviceNodeConfig) (string, error) {
	log := f.log.WithName("prepareFirmware")

	if config.DeviceConfig.FWURL == "" {
		log.V(4).Info("Empty FWURL")
		return "", nil
	}

	targetPath := filepath.Join(artifactsFolder, config.PCIAddress)

	err := utils.CreateFolder(targetPath, log)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(targetPath, filepath.Base(config.DeviceConfig.FWURL))
	log.V(4).Info("Downloading", "url", config.DeviceConfig.FWURL, "dstPath", fullPath)
	err = downloadFile(fullPath, config.DeviceConfig.FWURL, config.DeviceConfig.FWChecksum)
	if err != nil {
		return "", err
	}

	log.V(4).Info("FW file downloaded - extracting")
	err = untarFile(fullPath, targetPath, log)
	if err != nil {
		return "", err
	}

	return findFw(targetPath)
}

func (f *fwUpdater) handleFWUpdate(pciAddr, fwPath string) (bool, error) {
	log := f.log.WithName("handleFWUpdate")
	rebootRequired := false

	if fwPath == "" {
		return false, nil
	}

	err := f.updateFirmware(pciAddr, fwPath)
	if err != nil {
		log.Error(err, "Failed to update firmware", "device", pciAddr)
		return false, err
	}

	reboot, err := isRebootRequired(updateResultPath(fwPath))
	if err != nil {
		log.Error(err, "Failed to extract reboot required flag from file")
		rebootRequired = true // failsafe
	} else if reboot {
		log.V(4).Info("Node reboot required to complete firmware update", "device", pciAddr)
		rebootRequired = true
	}
	return rebootRequired, nil
}

func (f *fwUpdater) updateFirmware(pciAddr, fwPath string) error {
	log := f.log.WithName("updateFirmware")

	rootAttr := &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: 0, Gid: 0},
	}
	// Call nvmupdate64 -i first to refresh devices
	log.V(2).Info("Refreshing nvmupdate inventory")
	cmd := exec.Command(nvmupdate64e, "-i")
	cmd.SysProcAttr = rootAttr
	cmd.Dir = fwPath
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
	cmd.Dir = fwPath
	err = nvmupdateExec(cmd, log)
	if err != nil {
		return err
	}

	return nil
}

func findFwExec(targetPath string) (string, error) {
	var fwPaths []string
	walkFunction := func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(info.Name(), "nvmupdate64e") && isExecutable(info) {
			fwPaths = append(fwPaths, strings.TrimSuffix(path, "nvmupdate64e"))
		}
		return nil
	}
	err := filepath.Walk(targetPath, walkFunction)
	if err != nil {
		return "", err
	}
	if len(fwPaths) != 1 {
		return "", fmt.Errorf("expected to find exactly 1 file starting with 'nvmupdate64e', but found %v - %v", len(fwPaths), fwPaths)
	}
	return fwPaths[0], err
}

func nvmupdate64eCfgPath(p string) string { return filepath.Join(p, "nvmupdate.cfg") }
func updateResultPath(p string) string    { return filepath.Join(p, updateOutFile) }
func isExecutable(info os.FileInfo) bool  { return info.Mode()&0100 != 0 }
