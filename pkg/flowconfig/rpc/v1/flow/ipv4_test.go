// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flow

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("IPv4", func() {
	const (
		ip192_168_0_1     = uint32(0xC0A80001)
		ip192_168_0_2     = uint32(0xC0A80002)
		ip255_255_255_255 = uint32(0xFFFFFFFF)
	)

	createIpv4 := func(srcAddr, dstAddr string) Ipv4 {
		return Ipv4{
			Hdr: &Ipv4Hdr{
				VersionIhl:     10,
				TypeOfService:  11,
				TotalLength:    12,
				PacketId:       13,
				FragmentOffset: 14,
				TimeToLive:     15,
				NextProtoId:    16,
				HdrChecksum:    17,
				SrcAddr:        srcAddr,
				DstAddr:        dstAddr,
			},
		}
	}

	DescribeTable("IPv4 to Decimal conversion",
		func(ipv4str string, expectedVal uint32, valid bool, wantIPtoBeNil bool) {
			ip := net.ParseIP(ipv4str)
			if !wantIPtoBeNil {
				Expect(ip).NotTo(BeNil())
				ret := ipToUint32(ip)
				if valid {
					Expect(ret).Should(Equal(expectedVal))
				} else {
					Expect(ret).ShouldNot(Equal(expectedVal))
				}
			} else {
				Expect(ip).To(BeNil())
			}
		},
		// online IP to Int value conversion : https://www.vultr.com/resources/ipv4-converter/
		Entry("with valid IP 0.0.0.0", "0.0.0.0", uint32(0), true, false),
		Entry("with valid IP 255.255.255.255", "255.255.255.255", ip255_255_255_255, true, false),
		Entry("with valid IP ", "192.168.0.1", ip192_168_0_1, true, false),
		Entry("with invalid IP abc.def.ghi.jkl", "abc.def.ghi.jkl", uint32(0), false, true),
		Entry("Valid IPv6", "2001:db8::68", uint32(104), true, false),
		Entry("Valid IPv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", uint32(0x03707334), true, false),
	)

	Context("IPv4 to Decimal conversion - direct function call", func() {
		It("Too small number of bytes", func() {
			Expect(ipToUint32(nil)).Should(Equal(uint32(0)))
			Expect(ipToUint32([]byte{})).Should(Equal(uint32(0)))
		})

		It("Correct number of bytes", func() {
			Expect(ipToUint32([]byte("AAAA"))).Should(Equal(uint32(1094795585)))
		})
	})

	DescribeTable("Decimal to IPv4 conversion",
		func(decimalIP uint32, expectedIP string, valid bool) {
			ip := Uint32ToIP(decimalIP)
			if valid {
				Expect(ip.String()).Should(Equal(expectedIP))
			} else {
				Expect(ip).Should(BeNil())
			}
		},
		// online Int to IP value conversion : https://www.vultr.com/resources/ipv4-converter/
		Entry("with valid IP 0.0.0.0", uint32(0), "0.0.0.0", true),
		Entry("with valid IP 255.255.255.255", ip255_255_255_255, "255.255.255.255", true),
		Entry("with valid IP", ip192_168_0_1, "192.168.0.1", true),
	)

	DescribeTable("ToRteFlowItemIpv4 function",
		func(ipv4 Ipv4, expectError bool, src, dst int) {
			flowItem, err := ipv4.ToRteFlowItemIpv4()
			if expectError {
				Expect(err).ShouldNot(BeNil())
				Expect(flowItem).Should(BeNil())
			} else {
				Expect(err).Should(BeNil())
				Expect(flowItem).ShouldNot(BeNil())
				if ipv4.Hdr == nil {
					Expect(flowItem.Hdr).Should(BeNil())
				} else {
					Expect(flowItem.Hdr.VersionIhl).Should(Equal(uint32(10)))
					Expect(flowItem.Hdr.TypeOfService).Should(Equal(uint32(11)))
					Expect(flowItem.Hdr.TotalLength).Should(Equal(uint32(12)))
					Expect(flowItem.Hdr.PacketId).Should(Equal(uint32(13)))
					Expect(flowItem.Hdr.FragmentOffset).Should(Equal(uint32(14)))
					Expect(flowItem.Hdr.TimeToLive).Should(Equal(uint32(15)))
					Expect(flowItem.Hdr.NextProtoId).Should(Equal(uint32(16)))
					Expect(flowItem.Hdr.HdrChecksum).Should(Equal(uint32(17)))
					Expect(flowItem.Hdr.SrcAddr).Should(Equal(uint32(src)))
					Expect(flowItem.Hdr.DstAddr).Should(Equal(uint32(dst)))
				}
			}
		},
		Entry("Hdr is nil", Ipv4{}, false, 0, 0),
		Entry("Empty object", createIpv4("", ""), false, 0, 0),
		Entry("Missing hdr src addr", createIpv4("", "192.168.0.1"), false, 0, int(ip192_168_0_1)),
		Entry("Missing hdr dst addr", createIpv4("192.168.0.1", ""), false, int(ip192_168_0_1), 0),
		Entry("Valid hdr dst and src addr", createIpv4("192.168.0.1", "192.168.0.2"), false, int(ip192_168_0_1), int(ip192_168_0_2)),
		Entry("Invalid hdr src addr, valid dst addr, conversion error", createIpv4("abc.dfg.hij.klm", "192.168.0.1"), true, 0, 0),
		Entry("Invalid hdr dst addr, valid src addr, conversion error", createIpv4("192.168.0.1", "abc.dfg.hij.klm"), true, 0, 0),
	)
})
