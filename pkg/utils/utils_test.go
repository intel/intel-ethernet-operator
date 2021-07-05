// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package utils

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main suite")
}

var _ = Describe("Utils", func() {
	var _ = Describe("LoadSupportedDevices", func() {
		var _ = It("will fail if the file does not exist", func() {
			cfg, err := LoadSupportedDevices("notExistingFile.json")
			Expect(err).To(HaveOccurred())
			Expect(cfg).To(Equal(SupportedDevices{}))
		})
		var _ = It("will fail if the file is not json", func() {
			cfg, err := LoadSupportedDevices("testdata/invalid.json")
			Expect(err).To(HaveOccurred())
			Expect(cfg).To(Equal(SupportedDevices{}))
		})
		var _ = It("will load the valid config successfully", func() {
			cfg, err := LoadSupportedDevices("testdata/valid.json")
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
})
