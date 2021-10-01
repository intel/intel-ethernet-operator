// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package utils

import (
	"archive/tar"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
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

func LoadSupportedDevices(cfgPath string, inStruct interface{}) error {
	file, err := os.Open(filepath.Clean(cfgPath))
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
	f, err := os.Open(path)
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

// TODO: [ESS-2843] Add cert validation support
func downloadFile(path, url, checksum string) error {
	f, err := os.Create(path)
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

// DownloadFile downloads file from provided URL to provided path. If checksum value is
// not empty, it first checks if file already exists in path and skips downloading
// if calculated MD5 value matches provided one.
func DownloadFile(path, url, checksum string, log logr.Logger) error {
	_, err := os.Stat(path)
	if err == nil {
		ret, err := verifyChecksum(path, checksum)
		if err != nil {
			return err
		}
		if ret {
			log.V(4).Info("File already downloaded", "path", path)
			return nil
		}
		err = os.Remove(path)
		if err != nil {
			return fmt.Errorf("Unable to remove old file: %s",
				path)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	log.V(4).Info("Downloading file", "url", url)
	if err := downloadFile(path, url, checksum); err != nil {
		log.Error(err, "Unable to download file")
		return err
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
	log.V(4).Info("Extracting file", "srcPath", srcPath, "dstPath", dstPath)

	f, err := os.Open(srcPath)
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

		nfDst := filepath.Join(dstPath, fh.Name)

		switch fh.Typeflag {
		case tar.TypeReg:
			nf, err := os.OpenFile(nfDst, os.O_CREATE|os.O_RDWR, os.FileMode(fh.Mode))
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
		}
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

	out, err := cmd.Output()
	if err != nil {
		log.Error(err, "failed to execute command", "cmd", args, "output", string(out))
		return "", err
	}

	output := string(out)
	log.V(4).Info("commands output", "output", output)
	return output, nil
}
