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

	var _ = Describe("Untar", func() {
		log := klogr.New()
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
