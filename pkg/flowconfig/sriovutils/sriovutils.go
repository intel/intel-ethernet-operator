// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package sriovutils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	sysBusPci = "/bus/pci/devices"
)

type SriovUtils interface {
	// GetPfName returns SRIOV PF name for the given VF pci address(BDF) as string format
	GetPfName(string) (string, error)
	// GetVFID returns VF ID index (within specific PF) based on PCI address
	GetVFID(string) (int, error)
}

// GetSriovUtils returns an instance of SriovUtils
func GetSriovUtils(sysFs string) SriovUtils {
	return &sriovutils{
		SysFs: sysFs,
	}
}

// Assumption: caller will provide the SysFs root path prefix. e.g., /sys, /tmp/sys, /host/sys etc.
type sriovutils struct {
	SysFs string
}

// GetPfName returns SRIOV PF name for the given VF pci address
func (s *sriovutils) GetPfName(pciAddr string) (string, error) {

	path := filepath.Join(s.SysFs, sysBusPci, pciAddr, "physfn", "net")
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("error getting PF name for device %s: %v", pciAddr, err)
	} else if len(files) > 0 {
		return files[0].Name(), nil
	}
	return "", fmt.Errorf("the PF name is not found for device %s", pciAddr)
}

// GetVFID returns VF ID index (within specific PF) based on PCI address
func (s *sriovutils) GetVFID(pciAddr string) (vfID int, err error) {
	pfDir := filepath.Join(s.SysFs, sysBusPci, pciAddr, "physfn")
	vfID = -1
	_, err = os.Lstat(pfDir)
	if os.IsNotExist(err) {
		return vfID, nil
	}
	if err != nil {
		err = fmt.Errorf("could not get PF directory information for VF device: %s, Err: %v", pciAddr, err)
		return vfID, err
	}

	vfDirs, err := filepath.Glob(filepath.Join(pfDir, "virtfn*"))
	if err != nil {
		err = fmt.Errorf("error reading VF directories %v", err)
		return vfID, err
	}

	// Read all VF directory and get VF ID
	for vfID := range vfDirs {
		dirN := fmt.Sprintf("%s/virtfn%d", pfDir, vfID)
		dirInfo, err := os.Lstat(dirN)
		if err == nil && (dirInfo.Mode()&os.ModeSymlink != 0) {
			linkName, err := filepath.EvalSymlinks(dirN)
			if err == nil && strings.Contains(linkName, pciAddr) {
				return vfID, err
			}
		}
	}
	// The requested VF not found
	vfID = -1
	return vfID, nil
}
