// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-logr/logr"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ctrl "sigs.k8s.io/controller-runtime"
)

func Test(t *testing.T) {
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
		log := logr.Discard()
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
		log := logr.Discard()
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
			Expect(str).To(ContainSubstring("grep: unrecognized option '--fakeparam'"))
		})

		var _ = It("will return error if command is not set", func() {
			str, err := ExecCmd([]string{}, log)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cmd is empty"))
			Expect(str).To(Equal(""))
		})
	})

	var _ = Describe("DownloadFile", func() {
		var _ = It("will return error if url format is invalid", func() {
			defer os.Remove("/tmp/somefileanme")
			err := DownloadFile("/tmp/somefolder", "/tmp/fake", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported protocol"))
		})

		var _ = It("will return error if file already exists, but cannot acquire file", func() {
			tmpfile, err := ioutil.TempFile("/tmp", "somefilename")
			defer os.Remove(tmpfile.Name())
			Expect(err).ToNot(HaveOccurred())

			err = DownloadFile(tmpfile.Name(), "http://0.0.0.0/tmp/fake", "check")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to download image"))
		})

		var _ = It("will return a download error if file already exists and checksum matches, but no file with url found", func() {
			filePath := "/tmp/updatefile_101.tar.gz"
			url := "/tmp/fake"

			err := ioutil.WriteFile(filePath, []byte("1010101"), 0666)
			Expect(err).To(BeNil())
			defer os.Remove(filePath)

			err = DownloadFile(filePath, url, "63effa2530d088a06f071bc5f016f8d4")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported protocol"))
		})

		var _ = It("will return a download error if file already exists, checksum matches and url file found, but url does not match", func() {
			filePath := "/tmp/updatefile_101.tar.gz"
			fileWithUrl := filePath + ".url"
			url := "/tmp/fake"

			err := ioutil.WriteFile(filePath, []byte("1010101"), 0666)
			Expect(err).To(BeNil())
			defer os.Remove(filePath)

			err = ioutil.WriteFile(fileWithUrl, []byte(filePath), 0666)
			Expect(err).To(BeNil())
			defer os.Remove(fileWithUrl)

			err = DownloadFile(filePath, url, "63effa2530d088a06f071bc5f016f8d4")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported protocol"))
		})

		var _ = It("will return error if filename is invalid", func() {
			err := DownloadFile("", "/tmp/fake", "bf51ac6aceed5ca4227e640046ad9de4")
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
		})
	})

	var _ = Describe("OpenNoLinks", func() {
		var _ = It("will succeed if a path is neither symlink nor hard link", func() {
			tmpFile, err := ioutil.TempFile("", "regularFile")
			defer os.Remove(tmpFile.Name())
			Expect(err).ToNot(HaveOccurred())

			err = tmpFile.Close()
			Expect(err).ToNot(HaveOccurred())

			f, err := OpenNoLinks(tmpFile.Name())
			Expect(err).ToNot(HaveOccurred())
			Expect(f).ToNot(BeNil())

			err = f.Close()
			Expect(err).ToNot(HaveOccurred())
		})

		var _ = It("will return error if a path is a symlink", func() {
			tmpFile, err := ioutil.TempFile("", "regularFile")
			defer os.Remove(tmpFile.Name())
			Expect(err).ToNot(HaveOccurred())

			err = tmpFile.Close()
			Expect(err).ToNot(HaveOccurred())

			symlinkPath := tmpFile.Name() + "-symlink"
			err = os.Symlink(tmpFile.Name(), symlinkPath)
			defer os.Remove(symlinkPath)
			Expect(err).ToNot(HaveOccurred())

			f, err := OpenNoLinks(symlinkPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("too many levels of symbolic links"))
			Expect(f).To(BeNil())
		})

		var _ = It("will return error if a path is a hard link", func() {
			tmpFile, err := ioutil.TempFile("", "regularFile")
			defer os.Remove(tmpFile.Name())
			Expect(err).ToNot(HaveOccurred())

			err = tmpFile.Close()
			Expect(err).ToNot(HaveOccurred())

			hardlinkPath := tmpFile.Name() + "-hardlink"
			err = os.Link(tmpFile.Name(), hardlinkPath)
			defer os.Remove(hardlinkPath)
			Expect(err).ToNot(HaveOccurred())

			f, err := OpenNoLinks(hardlinkPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(hardlinkPath + " is a hardlink"))
			Expect(f).To(BeNil())
		})
	})

	var _ = Describe("Unzip", func() {
		log := logr.Discard()

		var _ = It("will unpack zip archive", func() {
			filesCreate := []string{
				"readme.txt",
				"testDir/",
				"testDir/test.txt",
				"testDir/nestedDir/",
				"testDir/nestedDir/test.txt"}
			zipPath := makeZip("zip-*.zip", filesCreate, nil)
			defer os.Remove(zipPath)

			dir, err := ioutil.TempDir("", "zip-")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			err = Unzip(zipPath, dir, log)
			Expect(err).ToNot(HaveOccurred())

			var extractedFiles []string
			err = filepath.WalkDir(
				dir,
				func(path string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}

					relPath, errRel := filepath.Rel(dir, path)
					if errRel != nil {
						return errRel
					}

					if relPath == "." {
						return nil
					}

					if d.IsDir() {
						relPath = relPath + "/"
					}

					extractedFiles = append(extractedFiles, relPath)

					return nil
				})
			Expect(err).ToNot(HaveOccurred())

			sort.Strings(filesCreate)
			Expect(extractedFiles).To(Equal(filesCreate))
		})

		var _ = It("will return error if input file is not a zip archive", func() {
			tarPath, filenames, err := testTar()
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(filenames[0])
			defer os.Remove(tarPath)

			err = Unzip(tarPath, os.TempDir(), log)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(zip.ErrFormat))
		})
	})

	var _ = Describe("UnpackDDPArchive", func() {
		log := logr.Discard()

		var _ = It("will unzip DDP archive", func() {
			innerFilesCreate := []string{
				"ice_comms-1.3.30.0.pkg",
				"readme.txt",
				"Intel_800_series_market_segment_DDP_license.txt"}
			innerZipPath := makeZip("ice_*.zip", innerFilesCreate, nil)
			defer os.Remove(innerZipPath)

			outerFilesCreate := []string{"E810 DDP for Comms TechGuide_Rev2.5.pdf"}
			ddpArchive := makeZip("test-*.zip", outerFilesCreate, []string{innerZipPath})
			defer os.Remove(ddpArchive)

			archivedFiles := append(innerFilesCreate, filepath.Base(innerZipPath))
			archivedFiles = append(archivedFiles, outerFilesCreate...)
			sort.Strings(archivedFiles)

			dir, err := ioutil.TempDir("", "ddp-")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			err = UnpackDDPArchive(ddpArchive, dir, log)
			Expect(err).ToNot(HaveOccurred())

			files, err := os.ReadDir(dir)
			Expect(err).ToNot(HaveOccurred())

			var extractedFiles []string
			for _, file := range files {
				extractedFiles = append(extractedFiles, file.Name())
			}

			Expect(extractedFiles).To(Equal(archivedFiles))
		})

		var _ = It("will return error if there are 2 inner zips", func() {
			innerFilesCreate := []string{
				"ice_comms-1.3.30.0.pkg",
				"readme.txt",
				"Intel_800_series_market_segment_DDP_license.txt"}

			var innerZipPaths []string
			for i := 0; i < 2; i++ {
				innerZipPaths = append(innerZipPaths, makeZip("ice_*.zip", innerFilesCreate, nil))
				defer os.Remove(innerZipPaths[i])
			}

			outerFilesCreate := []string{"E810 DDP for Comms TechGuide_Rev2.5.pdf"}
			ddpArchive := makeZip("test-*.zip", outerFilesCreate, innerZipPaths)
			defer os.Remove(ddpArchive)

			dir, err := ioutil.TempDir("", "ddp-")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			err = UnpackDDPArchive(ddpArchive, dir, log)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unexpected number of DDP archives"))
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

func makeZip(name string, filesCreate, filesCopy []string) string {
	zipFile, err := ioutil.TempFile("", name)
	Expect(err).ToNot(HaveOccurred())
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	for i, name := range filesCreate {
		fh := zip.FileHeader{
			Name:   name,
			Method: zip.Deflate}

		mode := os.FileMode(0600)
		if name[len(name)-1:] == "/" {
			mode |= os.ModeDir
			mode |= 0100
		}
		fh.SetMode(mode)

		writer, err := w.CreateHeader(&fh)
		Expect(err).ToNot(HaveOccurred())

		if mode.IsRegular() {
			content := []byte(fmt.Sprintf("%v: %v", i, name))
			_, err = writer.Write(content)
			Expect(err).ToNot(HaveOccurred())
		}
	}

	for _, path := range filesCopy {
		file, err := os.Open(path)
		Expect(err).ToNot(HaveOccurred())
		defer file.Close()

		writer, err := w.Create(filepath.Base(path))
		Expect(err).ToNot(HaveOccurred())

		_, err = io.Copy(writer, file)
		Expect(err).ToNot(HaveOccurred())
	}

	return zipFile.Name()
}
