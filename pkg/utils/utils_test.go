// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package utils

import (
	"archive/tar"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/klog/klogr"

	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main suite")
}

var _ = Describe("Utils", func() {
	var _ = Describe("LoadSupportedDevices", func() {
		var _ = It("will fail if the file does not exist", func() {
			cfg := make(SupportedDevices)
			err := LoadSupportedDevices("notExistingFile.json", &cfg)
			Expect(err).To(HaveOccurred())
			Expect(cfg).To(Equal(SupportedDevices{}))
		})
		var _ = It("will fail if the file is not json", func() {
			cfg := make(SupportedDevices)
			err := LoadSupportedDevices("testdata/invalid.json", &cfg)
			Expect(err).To(HaveOccurred())
			Expect(cfg).To(Equal(SupportedDevices{}))
		})
		var _ = It("will load the valid config successfully", func() {
			cfg := make(SupportedDevices)
			err := LoadSupportedDevices("testdata/valid.json", &cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).To(Equal(SupportedDevices{
				"E810": {
					VendorID: "0001",
					Class:    "00",
					SubClass: "00",
					DeviceID: "123",
				},
				"E811": {
					VendorID: "0002",
					Class:    "00",
					SubClass: "00",
					DeviceID: "321",
				},
			}))
		})
	})

	var _ = Describe("verifyChecksum", func() {
		var _ = It("will return false and error if it's not able to open file", func() {
			result, err := verifyChecksum("./invalidfile", "somechecksum")
			Expect(err).To(HaveOccurred())
			Expect(result).To(Equal(false))
		})

		var _ = It("will return false and no error if the expected is empty", func() {
			result, err := verifyChecksum("./invalidfile", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(false))
		})

		var _ = It("will return false if checksum does not match", func() {
			tmpfile, err := ioutil.TempFile(".", "update")
			Expect(err).ToNot(HaveOccurred())

			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.Write([]byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"))
			Expect(err).ToNot(HaveOccurred())
			err = tmpfile.Close()
			Expect(err).ToNot(HaveOccurred())

			result, err := verifyChecksum(tmpfile.Name(), "somechecksum")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(false))
		})

		var _ = It("will return true if checksum does match", func() {
			tmpfile, err := ioutil.TempFile(".", "testfile")
			Expect(err).ToNot(HaveOccurred())

			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.Write([]byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"))
			Expect(err).ToNot(HaveOccurred())
			err = tmpfile.Close()
			Expect(err).ToNot(HaveOccurred())

			f, _ := os.Open(tmpfile.Name())

			h := md5.New()
			_, err = io.Copy(h, f)
			Expect(err).ToNot(HaveOccurred())

			result, err := verifyChecksum(tmpfile.Name(), hex.EncodeToString(h.Sum(nil)))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(true))
		})
	})

	var _ = Describe("CreateFolder", func() {
		log := klogr.New()
		somefolderName := "/tmp/somefolder"
		var _ = It("will return no error if folder does not exist", func() {
			defer os.Remove(somefolderName)
			err := CreateFolder(somefolderName, log)
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(somefolderName)
			Expect(err).ToNot(HaveOccurred())
		})

		var _ = It("will return no error if folder already exists", func() {
			tmpfile, err := os.OpenFile(somefolderName, os.O_RDWR|os.O_CREATE, 0777)
			defer os.Remove(tmpfile.Name())
			Expect(err).ToNot(HaveOccurred())

			err = CreateFolder(somefolderName, log)
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(somefolderName)
			Expect(err).ToNot(HaveOccurred())
		})

		var _ = It("will return error if folder does not exist", func() {
			err := CreateFolder("", log)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such file or directory"))
		})
	})

	var _ = Describe("Write", func() {
		log := klogr.New()
		var _ = It("will return no error and a count of writtendata", func() {
			var l LogWriter
			l.Log = log
			testString := []byte("randomdata")
			n, err := l.Write(testString)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(len(testString)))
		})
	})

	var _ = Describe("ExecCmd", func() {
		log := ctrl.Log.WithName("EthernetDaemon-test")
		var _ = It("will return no error if output", func() {
			str, err := ExecCmd([]string{"ls"}, log)
			Expect(err).ToNot(HaveOccurred())
			Expect(str).ToNot(Equal(""))
		})

		var _ = It("will return error if command is invalid", func() {
			str, err := ExecCmd([]string{"grep", "--fakeparam"}, log)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exit status 2"))
			Expect(str).To(Equal(""))
		})

		var _ = It("will return error if command is not set", func() {
			str, err := ExecCmd([]string{}, log)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cmd is empty"))
			Expect(str).To(Equal(""))
		})
	})

	var _ = Describe("DownloadFile", func() {
		log := ctrl.Log.WithName("EthernetDaemon-test")
		var _ = It("will return error if url format is invalid", func() {
			defer os.Remove("/tmp/somefileanme")
			err := DownloadFile("/tmp/somefolder", "/tmp/fake", "", log)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported protocol"))
		})

		var _ = It("will return error if file already exists, but cannot acquire file", func() {
			tmpfile, err := ioutil.TempFile("/tmp", "somefilename")
			defer os.Remove(tmpfile.Name())
			Expect(err).ToNot(HaveOccurred())

			err = DownloadFile(tmpfile.Name(), "http://0.0.0.0/tmp/fake", "check", log)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to download image from: http://0.0.0.0/tmp/fake err: 502 Bad Gateway"))
		})

		var _ = It("will return no error if file already exists and checksum matches", func() {
			err := DownloadFile("testdata/invalid.json", "/tmp/fake", "7de0bf711e9ceb9269e7315c78024a32", log)
			Expect(err).ToNot(HaveOccurred())
		})

		var _ = It("will return error if filename is invalid", func() {
			err := DownloadFile("", "/tmp/fake", "bf51ac6aceed5ca4227e640046ad9de4", log)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such file or directory"))
		})
	})

	var _ = Describe("Untar", func() {
		log := ctrl.Log.WithName("EthernetDaemon-test")
		var _ = It("will return error if it's not able to open file", func() {
			err := Untar("./somesrcfile", "./somedstfile", log)
			Expect(err).To(HaveOccurred())
		})

		var _ = It("will return error if input file is not an archive", func() {
			tmpfile, err := ioutil.TempFile(".", "testfile")
			Expect(err).ToNot(HaveOccurred())

			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.Write([]byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"))
			Expect(err).ToNot(HaveOccurred())
			err = tmpfile.Close()
			Expect(err).ToNot(HaveOccurred())

			err = Untar(tmpfile.Name(), "./somedstfile", log)
			Expect(err).To(HaveOccurred())
		})

		var _ = It("will exctract valid archive", func() {
			tarPath, filenames, err := testTar()
			rootDir := filenames[0]
			Expect(err).ToNot(HaveOccurred())
			err = Untar(tarPath, "./", log)
			Expect(err).ToNot(HaveOccurred())
			Expect(filenames).To(HaveLen(5))
			defer os.RemoveAll(rootDir)
			defer os.Remove(tarPath)

			var untaredFilenames []string
			err = filepath.Walk(rootDir,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					untaredFilenames = append(untaredFilenames, path)
					return nil
				})
			Expect(err).ToNot(HaveOccurred())

			sort.Strings(filenames)
			sort.Strings(untaredFilenames)

			Expect(filenames).To(Equal(untaredFilenames))
			// TODO: walk over extracted files and compare their content to the original ones
		})
	})
})

func testTar() (string, []string, error) {
	// Generated test directory:
	// 	testdir
	// 	|-- testfile1
	// 	|-- testfile2
	// 	|-- testdir2
	// 	    |-- testfile3
	var filenames []string
	tarpath := "./test.tar.gz"

	tmpdir, err := ioutil.TempDir(".", "testdir")
	Expect(err).ToNot(HaveOccurred())
	defer os.RemoveAll(tmpdir)
	filenames = append(filenames, tmpdir)

	tmpfile1, err := ioutil.TempFile("./"+tmpdir, "testfile")
	Expect(err).ToNot(HaveOccurred())
	_, err = tmpfile1.Write([]byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"))
	Expect(err).ToNot(HaveOccurred())
	err = tmpfile1.Close()
	Expect(err).ToNot(HaveOccurred())
	filenames = append(filenames, tmpfile1.Name())

	tmpfile2, err := ioutil.TempFile("./"+tmpdir, "testfile")
	Expect(err).ToNot(HaveOccurred())
	_, err = tmpfile2.Write([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"))
	Expect(err).ToNot(HaveOccurred())
	err = tmpfile2.Close()
	Expect(err).ToNot(HaveOccurred())
	filenames = append(filenames, tmpfile2.Name())

	tmpdir2, err := ioutil.TempDir("./"+tmpdir, "testdir")
	Expect(err).ToNot(HaveOccurred())
	filenames = append(filenames, tmpdir2)

	tmpfile3, err := ioutil.TempFile("./"+tmpdir2, "testfile")
	Expect(err).ToNot(HaveOccurred())
	_, err = tmpfile3.Write([]byte("lmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk"))
	Expect(err).ToNot(HaveOccurred())
	err = tmpfile3.Close()
	Expect(err).ToNot(HaveOccurred())
	filenames = append(filenames, tmpfile3.Name())

	tarfile, err := os.Create(tarpath)
	Expect(err).ToNot(HaveOccurred())
	defer tarfile.Close()

	var fw io.Writer = tarfile

	gzw := gzip.NewWriter(fw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	err = filepath.Walk(tmpdir,
		func(path string, info os.FileInfo, err error) error {
			Expect(err).ToNot(HaveOccurred())
			header, err := tar.FileInfoHeader(info, info.Name())
			Expect(err).ToNot(HaveOccurred())

			header.Name = filepath.Join(tmpdir, strings.TrimPrefix(path, tmpdir))
			err = tw.WriteHeader(header)
			Expect(err).ToNot(HaveOccurred())

			if info.Mode().IsDir() {
				return nil
			}

			f, err := os.Open(path)
			Expect(err).ToNot(HaveOccurred())

			_, err = io.Copy(tw, f)
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			return nil
		})

	return tarpath, filenames, err
}
