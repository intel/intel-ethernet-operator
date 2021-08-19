// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package daemon

import (
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	nvmupdateOutput = `<?xml version="1.0" encoding="UTF-8"?>
<DeviceUpdate lang="en">
        <Instance vendor="8086" device="1592" subdevice="2" subvendor="8086" bus="6" dev="0" func="1" PBA="K91258-006" port_id="Port 2 of 2" display="Intel(R) Ethernet Network Adapter E810-C-Q2">
                <Module type="PXE" version="2.5.0" previous_version="2.0.0">
                        <Status result="Success" id="0">All operations completed successfully.</Status>
                </Module>
                <Module type="EFI" version="2.5.12" previous_version="2.1.2">
                        <Status result="Success" id="0">All operations completed successfully.</Status>
                </Module>
                <Module type="Netlist" version="2.80.29.0" previous_version="2.1.19.0">
                        <Status result="Success" id="0">All operations completed successfully.</Status>
                </Module>
                <Module type="NVM" version="800077A6" previous_version="800049C3">
                        <Status result="Success" id="0">All operations completed successfully.</Status>
                </Module>
                <MACAddresses>
                        <MAC address="B49691AA9E19">
                        </MAC>
                </MACAddresses>
        </Instance>
        <NextUpdateAvailable> 0 </NextUpdateAvailable>
        <RebootRequired> 1 </RebootRequired>
        <PowerCycleRequired> 0 </PowerCycleRequired>
</DeviceUpdate>`

	nvmupdateOutputMissingRebootRequiredClause = `<?xml version="1.0" encoding="UTF-8"?>
<DeviceUpdate lang="en">
        <Instance vendor="8086" device="1592" subdevice="2" subvendor="8086" bus="6" dev="0" func="1" PBA="K91258-006" port_id="Port 2 of 2" display="Intel(R) Ethernet Network Adapter E810-C-Q2">
                <Module type="PXE" version="2.5.0" previous_version="2.0.0">
                        <Status result="Success" id="0">All operations completed successfully.</Status>
                </Module>
                <Module type="EFI" version="2.5.12" previous_version="2.1.2">
                        <Status result="Success" id="0">All operations completed successfully.</Status>
                </Module>
                <Module type="Netlist" version="2.80.29.0" previous_version="2.1.19.0">
                        <Status result="Success" id="0">All operations completed successfully.</Status>
                </Module>
                <Module type="NVM" version="800077A6" previous_version="800049C3">
                        <Status result="Success" id="0">All operations completed successfully.</Status>
                </Module>
                <MACAddresses>
                        <MAC address="B49691AA9E19">
                        </MAC>
                </MACAddresses>
        </Instance>
        <NextUpdateAvailable> 0 </NextUpdateAvailable>
        <RebootRequired>`
)

var _ = Describe("isRebootRequired", func() {
	var _ = It("will request reboot as specified in the XML", func() {
		tmpfile, err := ioutil.TempFile(".", "update")
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.Write([]byte(nvmupdateOutput))
		Expect(err).ToNot(HaveOccurred())

		result, err := isRebootRequired(tmpfile.Name())

		Expect(result).To(BeTrue())
		Expect(err).ToNot(HaveOccurred())
	})

	var _ = It("will return error if too large file is provided", func() {
		tmpfile, err := ioutil.TempFile(".", "update")
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(tmpfile.Name())

		blob := make([]byte, 50000)
		_, err = tmpfile.Write([]byte(blob))

		Expect(err).ToNot(HaveOccurred())

		_, err = isRebootRequired(tmpfile.Name())

		Expect(err).To(HaveOccurred())
	})

	var _ = It("will return error if parsing xml exceeds timeout value", func() {
		tmpfile, err := ioutil.TempFile(".", "update")
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.Write([]byte(nvmupdateOutput))
		Expect(err).ToNot(HaveOccurred())

		updateXMLParseTimeout = 1 * time.Nanosecond
		_, err = isRebootRequired(tmpfile.Name())
		updateXMLParseTimeout = 100 * time.Millisecond

		Expect(err).To(HaveOccurred())
	})

	var _ = It("will return error if unable to open file", func() {
		_, err := isRebootRequired("/dev/null/fake")
		Expect(err).To(HaveOccurred())
	})

	var _ = It("will return error if missing RebootRequired closing tag", func() {
		tmpfile, err := ioutil.TempFile(".", "update")
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.Write([]byte(nvmupdateOutputMissingRebootRequiredClause))
		Expect(err).ToNot(HaveOccurred())

		_, err = isRebootRequired(tmpfile.Name())
		Expect(err).To(HaveOccurred())
	})
})
