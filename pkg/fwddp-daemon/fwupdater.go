// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/go-logr/logr"
	ethernetv1 "github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/apis/ethernet/v1"
	"github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/pkg/utils"
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

func (f *fwUpdater) handleFWUpdate(pciAddr, fwPath, fwUpdateParam string) (bool, error) {
	log := f.log.WithName("handleFWUpdate")
	rebootRequired := false

	if fwPath == "" {
		return false, nil
	}

	returnCode, err := f.updateFirmware(pciAddr, fwPath, fwUpdateParam)
	if err != nil {
		log.Error(err, "Failed to update firmware", "device", pciAddr)
		return false, err
	}

	var reboot bool
	// update successful despite exit code being >0, reboot needed to finish process
	if returnCode == 50 || returnCode == 51 {
		reboot = true
	} else {
		reboot, err = isRebootRequired(updateResultPath(fwPath))
	}
	if err != nil {
		log.Error(err, "Failed to extract reboot required flag from file")
		rebootRequired = true // failsafe
	} else if reboot {
		log.V(4).Info("Node reboot required to complete firmware update", "device", pciAddr)
		rebootRequired = true
	}
	return rebootRequired, nil
}

func (f *fwUpdater) updateFirmware(pciAddr, fwPath, fwUpdateParam string) (int, error) {
	log := f.log.WithName("updateFirmware")

	rootAttr := &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: 0, Gid: 0},
	}

	// read alternative fw search path, path might get modified on
	// NVM Update runtime so it needs to be restored after tool execution
	altFwPathBytes, err := os.ReadFile("/sys/module/firmware_class/parameters/path")
	if err != nil {
		log.V(2).Info("Error reading /sys/module/firmware_class/parameters/path", "error", err)
	}

	altFwPath := strings.TrimSuffix(string(altFwPathBytes), "\n")
	if altFwPath != "" {
		log.V(2).Info("Alternative firmware search path found", "path", altFwPath)
	}

	log.V(2).Info("Splitting PCI addr and converting to decimal", "pciAddr", pciAddr)
	domain, bus, _, _, err := splitPCIAddr(pciAddr, log)
	if err != nil {
		log.V(2).Info("Error spitting PCI Addr", "error", err)
		return -1, err
	}

	bus_dec, err := strconv.ParseInt(bus, 16, 32)
	if err != nil {
		log.V(2).Info("Error converting bus PCI to decimal", "error", err)
		return -1, err
	}
	domain_dec, err := strconv.ParseInt(domain, 16, 32)
	if err != nil {
		log.V(2).Info("Error converting PCI domain to decimal", "error", err)
		return -1, err
	}
	log.V(2).Info("PCI Addr splitted and converted successfully", "domain",
		domain_dec, "bus", bus_dec)

	configPath := nvmupdate64eCfgPath(fwPath)
	resultPath := updateResultPath(fwPath)
	pciLocation := fmt.Sprintf("%02d:%03d", domain_dec, bus_dec)

	log.V(2).Info("Starting Firmware Update", "pciLocation", pciLocation,
		"configPath", configPath, "resultPath", resultPath)

	var cmd *exec.Cmd
	if fwUpdateParam != "" {
		cmd = exec.Command(nvmupdate64e, "-u", "-location", pciLocation, "-c",
			configPath, "-o", resultPath, "-l", fwUpdateParam)
	} else {
		cmd = exec.Command(nvmupdate64e, "-u", "-location", pciLocation, "-c",
			configPath, "-o", resultPath, "-l")
	}
	cmd.SysProcAttr = rootAttr
	cmd.Dir = fwPath
	err = nvmupdateExec(cmd, log)

	// restore alternative firmware search path after update if necessary
	if altFwPath != "" {
		if err := os.WriteFile(
			"/sys/module/firmware_class/parameters/path",
			[]byte(altFwPath),
			0644,
		); err != nil {
			log.V(2).Info("Failed to restore alternative firmware search path")
		}
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			if code != 3 && code != 30 && code != 50 && code != 51 {
				return -1, exitErr
			} else {
				return code, nil
			}
		} else {
			return -1, err
		}
	}

	return 0, nil
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
