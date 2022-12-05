// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package sriovutils

import (
	"path"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSriovUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sriov utils suite")
}

func assertShouldFail(err error, shouldFail bool) {
	if shouldFail {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}
}

var _ = Describe("In the sriovutils package", func() {
	DescribeTable("getting PF names",
		func(fs *FakeFilesystem, device string, expected string, shouldFail bool) {

			tempRoot, tearDown := fs.Use()
			defer tearDown()
			su := GetSriovUtils(path.Join(tempRoot, "/sys"))
			actual, err := su.GetPfName(device)
			assertShouldFail(err, shouldFail)
			Expect(actual).To(Equal(expected))
		},
		Entry("device doesn't exist", &FakeFilesystem{}, "0000:01:10.0", "", true),
		Entry("device is a VF and interface name exists",
			&FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:01:10.0",
					"sys/bus/pci/devices/0000:01:00.0/net/fakePF",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:01:10.0/physfn/": "../0000:01:00.0",
				},
			}, "0000:01:10.0", "fakePF", false,
		),
		Entry("device is a VF and interface name does not exist",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/physfn/net/"}},
			"0000:01:10.0", "", true,
		),
		Entry("pf net is not a directory at all",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:01:10.0/physfn"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:01:10.0/physfn/net/": []byte("junk")},
			},
			"0000:01:10.0", "", true,
		),
	)

	DescribeTable("getting ID of VF",
		func(fs *FakeFilesystem, device string, expected int, shouldFail bool) {
			tempRoot, tearDown := fs.Use()
			defer tearDown()

			su := GetSriovUtils(path.Join(tempRoot, "/sys"))
			actual, err := su.GetVFID(device)
			Expect(actual).To(Equal(expected))
			assertShouldFail(err, shouldFail)
		},
		Entry("device doesn't exist",
			&FakeFilesystem{},
			"0000:01:10.0", -1, false),
		Entry("device has no link to PF",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:10.0"}},
			"0000:01:10.0", -1, false),
		Entry("PF has no VF links",
			&FakeFilesystem{
				Dirs:     []string{"sys/bus/pci/devices/0000:01:10.0/", "sys/bus/pci/devices/0000:01:00.0/"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:10.0/physfn": "../0000:01:00.0"},
			},
			"0000:01:10.0", -1, false),
		Entry("VF not found in PF",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/", "sys/bus/pci/devices/0000:01:00.0/"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:10.0/physfn": "../0000:01:00.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn0": "../0000:01:08.0",
				},
			},
			"0000:01:10.0", -1, false),
		Entry("VF found in PF",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/", "sys/bus/pci/devices/0000:01:00.0/"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:10.0/physfn": "../0000:01:00.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn0": "../0000:01:08.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn1": "../0000:01:09.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn2": "../0000:01:10.0",
				},
			},
			"0000:01:10.0", 2, false),
	)
})
