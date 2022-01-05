// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package daemon

import (
	"fmt"
	"github.com/go-logr/logr"
	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"
	"github.com/otcshare/intel-ethernet-operator/pkg/utils"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const (
	ddpPackageFilename = "ddp.tar.gz"
	// /host comes from mounted folder in OCP
	// /var/lib/firmware comes from modified kernel argument, which allows OS to read DDP profile from that path.
	// This is done because on RHCOS /lib/firmware/* path is read-only
	// intel/ice/ddp is default path for ICE *.pkg files
)

var (
	findDdp           = findDdpProfile
	reloadIceServiceP = reloadIceService

	ddpUpdateFolder = "/host/var/lib/firmware/intel/ice/ddp/"
)

type ddpUpdater struct {
	log logr.Logger
}

func (d *ddpUpdater) handleDDPUpdate(pciAddr string, forceReboot bool, ddpPath string) error {
	log := d.log.WithName("handleDDPUpdate")
	if ddpPath == "" {
		return nil
	}

	err := d.updateDDP(pciAddr, ddpPath)
	if err != nil {
		log.Error(err, "Failed to update DDP", "device", pciAddr)
		return err
	}

	// Recommended for clusters, on which ControlPlane is running on E810 cards.
	if forceReboot {
		return nil
	}

	err = reloadIceServiceP()
	if err != nil {
		log.Error(err, "Failed to reload ICE service")
		return err
	}

	return nil
}

func (d *ddpUpdater) determineDDPPath(ddpPath string, dev ethernetv1.Device) (string, error) {
	log := d.log.WithName("determineDDPPath")
	if ddpPath == "" {
		log.V(4).Info("DDP package not provided - retrieving version from device")
		return dev.DDP.PackageName + "-" + dev.DDP.Version, nil
	} else {
		log.V(4).Info("Retrieving version from", "path", ddpPath)
		return ddpPath, nil
	}
}

// ddpProfilePath is the path to our extracted DDP profile
// we copy it to ddpUpdateFolder
func (d *ddpUpdater) updateDDP(pciAddr, ddpProfilePath string) error {
	log := d.log.WithName("updateDDP")

	err := os.MkdirAll(ddpUpdateFolder, 0600)
	if err != nil {
		return err
	}

	devId, err := execCmd([]string{"sh", "-c", "lspci -vs " + pciAddr +
		" | awk '/Device Serial/ {print $NF}' | sed s/-//g"}, log)
	if err != nil {
		return err
	}
	devId = strings.TrimSuffix(devId, "\n")
	if devId == "" {
		return fmt.Errorf("failed to extract devId")
	}

	target := path.Join(ddpUpdateFolder, "ice-"+devId+".pkg")
	log.V(4).Info("Copying", "source", ddpProfilePath, "target", target)

	return utils.CopyFile(ddpProfilePath, target)
}

func (d *ddpUpdater) prepareDDP(config ethernetv1.DeviceNodeConfig) (string, error) {
	log := d.log.WithName("prepareDDP")

	if config.DeviceConfig.DDPURL == "" {
		log.V(4).Info("Empty DDPURL")
		return "", nil
	}

	targetPath := path.Join(artifactsFolder, config.PCIAddress)

	err := utils.CreateFolder(targetPath, log)
	if err != nil {
		return "", err
	}

	fullPath := path.Join(targetPath, ddpPackageFilename)
	log.V(4).Info("Downloading", "url", config.DeviceConfig.DDPURL, "dstPath", fullPath)
	err = downloadFile(fullPath, config.DeviceConfig.DDPURL, config.DeviceConfig.DDPChecksum)
	if err != nil {
		return "", err
	}

	log.V(4).Info("DDP file downloaded - extracting")
	// XXX so this unpacks into the same directory as the source file
	// We might add more comments here explaining the mechanics and reasoning
	err = unpackDDPArchive(fullPath, targetPath, log)
	if err != nil {
		return "", err
	}

	return findDdp(targetPath)
}

func reloadIceService() error {
	cmd := exec.Command("chroot", "/host", "systemctl", "restart", "oot-ice-driver-load")
	return cmd.Run()
}

func findDdpProfile(targetPath string) (string, error) {
	var ddpProfilesPaths []string
	walkFunction := func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(info.Name(), ".pkg") && info.Mode()&os.ModeSymlink == 0 {
			ddpProfilesPaths = append(ddpProfilesPaths, path)
		}
		return nil
	}
	err := filepath.Walk(targetPath, walkFunction)
	if err != nil {
		return "", err
	}
	if len(ddpProfilesPaths) != 1 {
		return "", fmt.Errorf("expected to find exactly 1 file ending with '.pkg', but found %v - %v", len(ddpProfilesPaths), ddpProfilesPaths)
	}
	return ddpProfilesPaths[0], err
}
