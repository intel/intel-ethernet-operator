// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"syscall"

	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
)

type SupportedDevices map[string]SupportedDevice
type SupportedDevice struct {
	VendorID string
	Class    string
	SubClass string
	DeviceID string
}

const (
	configFilesizeLimitInBytes = 10485760 //10 MB
)
const IeoPrefix = "ETHERNET_"

func GetDrainSkip(nodes *corev1.NodeList, client client.Client, log logr.Logger) (bool, error) {
	if IsK8sDeployment() {
		if len(nodes.Items) <= 1 {
			log.Info("found only 0 or 1 node(s) with CLV label in cluster - operator is running on SNO")
			return true, nil
		}
		log.Info("found several nodes with CLV label in cluster - operator is running on Cluster deployment", "nodes", len(nodes.Items))
		return false, nil
	}
	return IsOpenshiftSno(client, log)
}

func IsK8sDeployment() bool {
	value := os.Getenv(IeoPrefix + "GENERIC_K8S")
	return strings.ToLower(value) == "true"
}

func IsOpenshiftSno(c client.Client, log logr.Logger) (bool, error) {
	infra := &configv1.Infrastructure{}

	defaultInfraName := "cluster"
	err := c.Get(context.TODO(), types.NamespacedName{Name: defaultInfraName}, infra)
	if err != nil {
		return false, err
	}
	if infra == nil {
		return false, fmt.Errorf("getting resource Infrastructure (name: %s) succeeded but object was nil", defaultInfraName)
	}
	log.Info("OCP cluster infrastructure", "infra", infra.Status.ControlPlaneTopology)
	return infra.Status.ControlPlaneTopology == configv1.SingleReplicaTopologyMode, nil
}

func LoadSupportedDevices(cfgPath string, inStruct interface{}) error {
	file, err := OpenNoLinks(filepath.Clean(cfgPath))
	if err != nil {
		return fmt.Errorf("Failed to open config: %v", err)
	}
	defer file.Close()

	// get file stat
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Failed to get file stat: %v", err)
	}

	// check file size
	if stat.Size() > configFilesizeLimitInBytes {
		return fmt.Errorf("Config file size %d, exceeds limit %d bytes",
			stat.Size(), configFilesizeLimitInBytes)
	}

	cfgData := make([]byte, stat.Size())
	bytesRead, err := file.Read(cfgData)
	if err != nil || int64(bytesRead) != stat.Size() {
		return fmt.Errorf("Unable to read config: %s", filepath.Clean(cfgPath))
	}

	if err = json.Unmarshal(cfgData, inStruct); err != nil {
		return fmt.Errorf("Failed to unmarshal config: %v", err)
	}
	return nil
}

func (l *LogWriter) Write(p []byte) (n int, err error) {
	o := strings.TrimSpace(string(p))
	// Split the input string to avoid clumping of multiple lines
	for _, s := range strings.FieldsFunc(o, func(r rune) bool { return r == '\n' || r == '\r' }) {
		l.Log.V(2).Info(strings.TrimSpace(s), "stream", l.Stream)
	}
	return len(p), nil
}

func verifyChecksum(path, expected string) (bool, error) {
	if expected == "" {
		return false, nil
	}
	f, err := OpenNoLinks(path)
	if err != nil {
		return false, errors.New("Failed to open file to calculate md5")
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, errors.New("Failed to copy file to calculate md5")
	}
	if hex.EncodeToString(h.Sum(nil)) != expected {
		return false, nil
	}

	return true, nil
}

func DownloadFile(path, url, checksum string) error {
	f, err := CreateNoLinks(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("unable to download image from: %s err: %s", url, err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to download image from: %s err: %s",
			url, r.Status)
	}

	_, err = io.Copy(f, r.Body)
	if err != nil {
		return err
	}

	if checksum != "" {
		match, err := verifyChecksum(path, checksum)
		if err != nil {
			return err
		}
		if !match {
			return fmt.Errorf("Checksum mismatch in downloaded file: %s", url)
		}
	}
	return nil
}

func CreateFolder(path string, log logr.Logger) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	err = os.MkdirAll(path, 0777)
	if err != nil {
		log.V(4).Info("Unable to create", "path", path)
		return err
	}
	return nil
}

type LogWriter struct {
	Log    logr.Logger
	Stream string
}

func Untar(srcPath string, dstPath string, log logr.Logger) error {
	log.V(4).Info("Untar file", "srcPath", srcPath, "dstPath", dstPath)

	f, err := OpenNoLinks(srcPath)
	if err != nil {
		log.Error(err, "Unable to open file")
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		fh, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err, "Error when reading tar")
			return err
		}
		if fh == nil {
			err = fmt.Errorf("invalid header in file %s", fh.Name)
			log.Error(err, "Invalid tar header")
			return err
		}

		nfDst := filepath.Join(dstPath, fh.Name)

		switch fh.Typeflag {
		case tar.TypeReg:
			nf, err := OpenFileNoLinks(nfDst, os.O_CREATE|os.O_RDWR, os.FileMode(fh.Mode))
			if err != nil {
				return err
			}
			defer nf.Close()

			_, err = io.Copy(nf, tr)
			if err != nil {
				return err
			}
		case tar.TypeDir:
			err := os.MkdirAll(nfDst, fh.FileInfo().Mode())
			if err != nil {
				return err
			}
		case tar.TypeSymlink, tar.TypeLink:
			log.Info("Skipping (sym)link", "filename", fh.FileInfo().Name())
		default:
			err = fmt.Errorf("unable to untar type: %c in file %s", fh.Typeflag, fh.Name)
			log.Error(err, "Invalid untar type")
			return err
		}
	}

	return nil
}

func unzipFile(zipFile *zip.File, path string, mode os.FileMode) error {
	file, err := OpenFileNoLinks(path, os.O_CREATE|os.O_RDWR, mode)
	if err != nil {
		return err
	}
	defer file.Close()

	reader, err := zipFile.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return err
	}

	return nil
}

func Unzip(srcPath, dstPath string, log logr.Logger) error {
	log.V(4).Info("Unzip file", "srcPath", srcPath, "dstPath", dstPath)

	f, err := OpenNoLinks(srcPath)
	if err != nil {
		log.Error(err, "Unable to open", "file", srcPath)
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		log.Error(err, "Can't stat", "file", f.Name())
		return err
	}

	r, err := zip.NewReader(f, stat.Size())
	if err != nil {
		return err
	}

	for _, zipFile := range r.File {
		fi := zipFile.FileInfo()
		mode := fi.Mode()
		nfDst := filepath.Join(dstPath, zipFile.Name)

		switch {
		case mode.IsRegular():
			err = unzipFile(zipFile, nfDst, mode)
			if err != nil {
				return err
			}

		case mode.IsDir():
			err = os.MkdirAll(nfDst, mode)
			if err != nil {
				return err
			}

		case mode&os.ModeSymlink == os.ModeSymlink:
			log.Info("Skipping symlink", "filename", zipFile.Name)

		default:
			err = fmt.Errorf("unable to unzip file %v", zipFile.Name)
			log.Error(err, "invalid unzip type")
			return err
		}
	}

	return nil
}

func UnpackDDPArchive(srcPath, dstPath string, log logr.Logger) error {
	log.V(4).Info("Unpack DDP archive", "srcPath", srcPath, "dstPath", dstPath)
	err := Unzip(srcPath, dstPath, log)

	switch {
	case err == nil:
		namePattern := "ice_*.zip"
		ddpArchive := filepath.Join(dstPath, namePattern)
		matches, err := filepath.Glob(ddpArchive)
		if err != nil {
			return err
		}

		if len(matches) != 1 {
			err = fmt.Errorf("unexpected number of DDP archives")
			log.Error(
				err, "expected 1 match", "name pattern", namePattern, "matches found", len(matches))
			return err
		}

		err = Unzip(matches[0], dstPath, log)
		if err != nil {
			return err
		}

	case errors.Is(err, zip.ErrFormat):
		err = Untar(srcPath, dstPath, log)
		if err != nil {
			return err
		}

	default:
		return err
	}

	return nil
}

func ExecCmd(args []string, log logr.Logger) (string, error) {
	var cmd *exec.Cmd
	if len(args) == 0 {
		log.Error(nil, "provided cmd is empty")
		return "", errors.New("cmd is empty")
	} else if len(args) == 1 {
		cmd = exec.Command(args[0])
	} else {
		cmd = exec.Command(args[0], args[1:]...)
	}

	log.V(4).Info("executing command", "cmd", cmd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Error(err, "failed to execute command", "cmd", args, "output", string(out))
		return string(out), err
	}

	output := string(out)
	log.V(4).Info("commands output", "output", output)
	return output, nil
}

func CopyFile(src string, dst string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, data, 0644)
}

func RunExecWithLog(cmd *exec.Cmd, log logr.Logger) error {
	cmd.Stdout = &LogWriter{Log: log, Stream: "stdout"}
	cmd.Stderr = &LogWriter{Log: log, Stream: "stderr"}
	return cmd.Run()
}

func GetDriverVersion(dev ethernetv1.Device) string {
	return dev.Driver + "-" + dev.DriverVersion
}

func isHardLink(path string) (bool, error) {
	var stat syscall.Stat_t

	err := syscall.Stat(path, &stat)
	if err != nil {
		return false, err
	}

	if stat.Nlink > 1 {
		return true, nil
	}

	return false, nil
}

func OpenNoLinks(path string) (*os.File, error) {
	return OpenFileNoLinks(path, os.O_RDONLY, 0)
}

func CreateNoLinks(path string) (*os.File, error) {
	return OpenFileNoLinks(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func OpenFileNoLinks(path string, flag int, perm os.FileMode) (*os.File, error) {
	// O_NOFOLLOW - If the trailing component (i.e., basename) of pathname is a symbolic link,
	// then the open fails, with the error ELOOP.
	file, err := os.OpenFile(path, flag|syscall.O_NOFOLLOW, perm)
	if err != nil {
		return nil, err
	}

	hardLink, err := isHardLink(path)
	if err != nil {
		file.Close()
		return nil, err
	}

	if hardLink {
		file.Close()
		return nil, fmt.Errorf("%v is a hardlink", path)
	}

	return file, nil
}
