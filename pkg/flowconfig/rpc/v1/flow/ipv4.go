// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flow

import (
	"encoding/binary"
	"fmt"
	"net"
)

// Ipv4 is the intermediary struct to unmarshall IPv4 headers with doted-decimal IP and Mask values from yaml/json
type Ipv4 struct {
	Hdr *Ipv4Hdr `protobuf:"bytes,1,opt,name=hdr" json:"hdr,omitempty"`
}

type Ipv4Hdr struct {
	VersionIhl     uint32 `json:"version_ihl,omitempty"`
	TypeOfService  uint32 `json:"type_of_service,omitempty"`
	TotalLength    uint32 `json:"total_length,omitempty"`
	PacketId       uint32 `json:"packet_id,omitempty"`
	FragmentOffset uint32 `json:"fragment_offset,omitempty"`
	TimeToLive     uint32 `json:"time_to_live,omitempty"`
	NextProtoId    uint32 `json:"next_proto_id,omitempty"`
	HdrChecksum    uint32 `json:"hdr_checksum,omitempty"`
	SrcAddr        string `json:"src_addr,omitempty"`
	DstAddr        string `json:"dst_addr,omitempty"`
}

func (ipv4 *Ipv4) ToRteFlowItemIpv4() (*RteFlowItemIpv4, error) {
	rteFlowItemIpv4 := &RteFlowItemIpv4{}
	if ipv4.Hdr != nil {
		hdr := &RteIpv4Hdr{}

		// Copy same fields
		hdr.VersionIhl = ipv4.Hdr.VersionIhl
		hdr.TypeOfService = ipv4.Hdr.TypeOfService
		hdr.TotalLength = ipv4.Hdr.TotalLength
		hdr.PacketId = ipv4.Hdr.PacketId
		hdr.FragmentOffset = ipv4.Hdr.FragmentOffset
		hdr.TimeToLive = ipv4.Hdr.TimeToLive
		hdr.NextProtoId = ipv4.Hdr.NextProtoId
		hdr.HdrChecksum = ipv4.Hdr.HdrChecksum

		// Convert Src and Dst IP address from string dotted-decimal to uint32
		if ipv4.Hdr.SrcAddr != "" {
			ip := net.ParseIP(ipv4.Hdr.SrcAddr)
			if ip == nil {
				return nil, fmt.Errorf("could not parse IP address")
			}
			hdr.SrcAddr = ipToUint32(ip)
		}
		if ipv4.Hdr.DstAddr != "" {
			ip := net.ParseIP(ipv4.Hdr.DstAddr)
			if ip == nil {
				return nil, fmt.Errorf("could not parse IP address")
			}
			hdr.DstAddr = ipToUint32(ip)
		}

		rteFlowItemIpv4.Hdr = hdr
	}

	return rteFlowItemIpv4, nil
}

func ipToUint32(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}

	// smaller number of bytes passed to converter will leads to runtime error
	if len(ip) < 3 {
		return 0
	}

	return binary.BigEndian.Uint32(ip)
}

func Uint32ToIP(val uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, val)
	return ip
}
