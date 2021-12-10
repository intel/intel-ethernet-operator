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
	"syscall"
)

const (
	nvmupdate64e             = "./nvmupdate64e"
	nvmupdateVersionFilesize = 10
	nvmupdatePackageFilename = "nvmupdate.tar.gz"
	nvmupdate64eDirSuffix    = "E810/Linux_x64/"
	updateOutFile            = "update.xml"
	nvmupdateVersionFilename = "version.txt"
)

var (
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

	targetPath := path.Join(artifactsFolder, config.PCIAddress)

	err := utils.CreateFolder(targetPath, log)
	if err != nil {
		return "", err
	}

	log.V(4).Info("Downloading", "url", config.DeviceConfig.FWURL)
	err = downloadFile(path.Join(targetPath, nvmupdatePackageFilename), config.DeviceConfig.FWURL, config.DeviceConfig.FWChecksum)
	if err != nil {
		return "", err
	}

	log.V(4).Info("FW file downloaded - extracting")
	err = untarFile(path.Join(targetPath, nvmupdatePackageFilename), targetPath, log)
	if err != nil {
		return "", err
	}

	return targetPath, nil
}

func (f *fwUpdater) getFWVersion(fwPath string, dev ethernetv1.Device) (string, error) {
	log := f.log.WithName("getFWVersion")
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
			return "", fmt.Errorf("failed to open version file: %v", err)
		}
		defer file.Close()

		ver := make([]byte, nvmupdateVersionFilesize)
		n, err := file.Read(ver)
		if err != nil {
			return "", fmt.Errorf("unable to read: %v", path)
		}
		// Example version.txt content: v2.40
		return strings.ReplaceAll(strings.TrimSpace(string(ver[:n])), "v", ""), nil
	}
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

func nvmupdate64eDir(p string) string     { return path.Join(p, nvmupdate64eDirSuffix) }
func nvmupdate64eCfgPath(p string) string { return path.Join(nvmupdate64eDir(p), "nvmupdate.cfg") }
func updateResultPath(p string) string    { return path.Join(nvmupdate64eDir(p), updateOutFile) }
