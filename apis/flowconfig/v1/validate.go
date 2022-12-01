// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

package v1

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
	any "google.golang.org/protobuf/types/known/anypb"
)

// RteFlowItem Validation

func validateRteFlowItemEth(itemName string, spec, item *any.Any) error {
	specItem := new(flow.RteFlowItemEth)
	if spec != nil {
		if err := spec.UnmarshalTo(specItem); err != nil {
			return fmt.Errorf("could not unmarshal %s: %s", itemName, err)
		}
	}

	ethItem := new(flow.RteFlowItemEth)
	if err := item.UnmarshalTo(ethItem); err != nil {
		return fmt.Errorf("could not unmarshal %s: %s", itemName, err)
	}
	var specDstAddrInt, specSrcAddrInt uint32 = 0, 0
	var err error
	if ethItem.Type != 0 {
		if err = validateUint32ItemField(itemName, "type", specItem.Type, ethItem.Type); err != nil {
			return fmt.Errorf("validateUint32ItemField() error: %v", err)
		}
	}
	if ethItem.Dst != nil {
		if specItem.Dst != nil {
			specDstAddrInt, err = encodeHexMacAddress(specItem.Dst.AddrBytes)
			if err != nil {
				return fmt.Errorf("invalid spec: %v, %v", specItem.Dst.AddrBytes, err)
			}
		}
		dstAddrInt, err := encodeHexMacAddress(ethItem.Dst.AddrBytes)
		if err != nil {
			return fmt.Errorf("invalid %s: %v, %v", itemName, ethItem.Dst.AddrBytes, err)
		}
		if err := validateUint32ItemField(itemName, "dst", specDstAddrInt, dstAddrInt); err != nil {
			return err
		}
	}
	if ethItem.Src != nil {
		if specItem.Src != nil {
			specSrcAddrInt, err = encodeHexMacAddress(specItem.Src.AddrBytes)
			if err != nil {
				return fmt.Errorf("invalid spec: %v, %v", specItem.Src.AddrBytes, err)
			}
		}
		srcAddrInt, err := encodeHexMacAddress(ethItem.Src.AddrBytes)
		if err != nil {
			return fmt.Errorf("invalid %s: %v, %v", itemName, ethItem.Src.AddrBytes, err)
		}
		if err := validateUint32ItemField(itemName, "src", specSrcAddrInt, srcAddrInt); err != nil {
			return err
		}
	}

	if ethItem.Type > 0xFFFF {
		return fmt.Errorf("invalid type (%d), must be equal or lower than 65,535", ethItem.Type)
	}

	return nil
}

func encodeHexMacAddress(addr []byte) (uint32, error) {
	intAddr := make([]byte, hex.EncodedLen(len(addr)))
	hex.Encode(intAddr, addr)
	parseAddr, err := strconv.ParseInt(string(intAddr[:]), 16, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing mac address: %v", err)
	}
	return uint32(parseAddr), nil
}

func validateRteFlowItemVlan(itemName string, spec, item *any.Any) error {
	specItem := new(flow.RteFlowItemVlan)
	if spec != nil {
		if err := spec.UnmarshalTo(specItem); err != nil {
			return fmt.Errorf("could not unmarshal %s: %s", itemName, err)
		}
	}

	vlanItem := new(flow.RteFlowItemVlan)
	if err := item.UnmarshalTo(vlanItem); err != nil {
		return fmt.Errorf("could not unmarshal %s: %s", itemName, err)
	}

	if err := validateUint32ItemField(itemName, "tci", specItem.Tci, vlanItem.Tci); err != nil {
		return err
	}

	if err := validateUint32ItemField(itemName, "inner_type", specItem.InnerType, vlanItem.InnerType); err != nil {
		return err
	}

	if vlanItem.Tci > 0xFFFF {
		return fmt.Errorf("invalid tci (%d), must be equal or lower than 65,535", vlanItem.Tci)
	}

	if vlanItem.InnerType > 0xFFFF {
		return fmt.Errorf("invalid inner_type (%d), must be equal or lower than 65,535", vlanItem.InnerType)
	}

	return nil
}

func validateRteFlowItemIpv4(itemName string, spec, item *any.Any) error {
	specItem := new(flow.RteFlowItemIpv4)
	if spec != nil {
		if err := spec.UnmarshalTo(specItem); err != nil {
			return fmt.Errorf("could not unmarshal %s: %s", itemName, err)
		}
	}

	ipv4Item := new(flow.RteFlowItemIpv4)
	if err := item.UnmarshalTo(ipv4Item); err != nil {
		return fmt.Errorf("could not unmarshal %s: %s", itemName, err)
	}

	if specItem.Hdr != nil {
		if err := validateUint32ItemField(itemName, "version_ihl", specItem.Hdr.VersionIhl, ipv4Item.Hdr.VersionIhl); err != nil {
			return err
		}
		if err := validateUint32ItemField(itemName, "type_of_service", specItem.Hdr.TypeOfService, ipv4Item.Hdr.TypeOfService); err != nil {
			return err
		}
		if err := validateUint32ItemField(itemName, "total_length", specItem.Hdr.TotalLength, ipv4Item.Hdr.TotalLength); err != nil {
			return err
		}
		if err := validateUint32ItemField(itemName, "packet_id", specItem.Hdr.PacketId, ipv4Item.Hdr.PacketId); err != nil {
			return err
		}
		if err := validateUint32ItemField(itemName, "fragment_offset", specItem.Hdr.FragmentOffset, ipv4Item.Hdr.FragmentOffset); err != nil {
			return err
		}
		if err := validateUint32ItemField(itemName, "time_to_live", specItem.Hdr.TimeToLive, ipv4Item.Hdr.TimeToLive); err != nil {
			return err
		}
		if err := validateUint32ItemField(itemName, "next_proto_id", specItem.Hdr.NextProtoId, ipv4Item.Hdr.NextProtoId); err != nil {
			return err
		}
		if err := validateUint32ItemField(itemName, "hdr_checksum", specItem.Hdr.HdrChecksum, ipv4Item.Hdr.HdrChecksum); err != nil {
			return err
		}
		if err := validateUint32ItemField(itemName, "src_addr", specItem.Hdr.SrcAddr, ipv4Item.Hdr.SrcAddr); err != nil {
			return err
		}
		if err := validateUint32ItemField(itemName, "dst_addr", specItem.Hdr.DstAddr, ipv4Item.Hdr.DstAddr); err != nil {
			return err
		}
	}

	if ipv4Item.Hdr.VersionIhl > 0xFF {
		return fmt.Errorf("invalid version_ihl (%d), must be equal or lower than 255", ipv4Item.Hdr.VersionIhl)
	}
	if ipv4Item.Hdr.TypeOfService > 0xFF {
		return fmt.Errorf("invalid type_of_service (%d), must be equal or lower than 255", ipv4Item.Hdr.TypeOfService)
	}
	if ipv4Item.Hdr.TotalLength > 0xFFFF {
		return fmt.Errorf("invalid total_length (%d), must be equal or lower than 65,535", ipv4Item.Hdr.TotalLength)
	}
	if ipv4Item.Hdr.PacketId > 0xFFFF {
		return fmt.Errorf("invalid packet_id (%d), must be equal or lower than 65,535", ipv4Item.Hdr.PacketId)
	}
	if ipv4Item.Hdr.FragmentOffset > 0xFFFF {
		return fmt.Errorf("invalid fragment_offset (%d), must be equal or lower than 65,535", ipv4Item.Hdr.FragmentOffset)
	}
	if ipv4Item.Hdr.TimeToLive > 0xFF {
		return fmt.Errorf("invalid time_to_live (%d), must be equal or lower than 255", ipv4Item.Hdr.TimeToLive)
	}
	if ipv4Item.Hdr.NextProtoId > 0xFF {
		return fmt.Errorf("invalid next_proto_id (%d), must be equal or lower than 255", ipv4Item.Hdr.NextProtoId)
	}
	if ipv4Item.Hdr.HdrChecksum > 0xFF {
		return fmt.Errorf("invalid hdr_checksum (%d), must be equal or lower than 255", ipv4Item.Hdr.HdrChecksum)
	}

	return nil
}

func validateRteFlowItemUdp(itemName string, spec, item *any.Any) error {
	specItem := new(flow.RteFlowItemUdp)
	if spec != nil {
		if err := spec.UnmarshalTo(specItem); err != nil {
			return fmt.Errorf("could not unmarshal %s: %s", itemName, err)
		}
	}

	udpItem := new(flow.RteFlowItemUdp)
	if err := item.UnmarshalTo(udpItem); err != nil {
		return fmt.Errorf("could not unmarshal %s: %s", itemName, err)
	}

	if specItem.Hdr != nil {
		if udpItem.Hdr.SrcPort != 0 {
			if err := validateUint32ItemField(itemName, "src_port", specItem.Hdr.SrcPort, udpItem.Hdr.SrcPort); err != nil {
				return err
			}
		}
		if udpItem.Hdr.DstPort != 0 {
			if err := validateUint32ItemField(itemName, "dst_port", specItem.Hdr.DstPort, udpItem.Hdr.DstPort); err != nil {
				return err
			}
		}
		if udpItem.Hdr.DgramLen != 0 {
			if err := validateUint32ItemField(itemName, "dgram_len", specItem.Hdr.DgramLen, udpItem.Hdr.DgramLen); err != nil {
				return err
			}
		}
		if udpItem.Hdr.DgramCksum != 0 {
			if err := validateUint32ItemField(itemName, "dgram_cksum", specItem.Hdr.DgramCksum, udpItem.Hdr.DgramCksum); err != nil {
				return err
			}
		}
	}

	if udpItem.Hdr != nil {
		if udpItem.Hdr.SrcPort > 0xFFFF {
			return fmt.Errorf("invalid src_port (%d), must be equal or lower than 65,535", udpItem.Hdr.SrcPort)
		}
		if udpItem.Hdr.DstPort > 0xFFFF {
			return fmt.Errorf("invalid dst_port (%d), must be equal or lower than 65,535", udpItem.Hdr.DstPort)
		}
		if udpItem.Hdr.DgramLen > 0xFFFF {
			return fmt.Errorf("invalid dgram_len (%d), must be equal or lower than 65,535", udpItem.Hdr.DgramLen)
		}
		if udpItem.Hdr.DgramCksum > 0xFFFF {
			return fmt.Errorf("invalid dgram_cksum (%d), must be equal or lower than 65,535", udpItem.Hdr.DgramCksum)
		}
	}

	return nil
}

func validateRteFlowItemPppoe(itemName string, spec, item *any.Any) error {
	specItem := new(flow.RteFlowItemPppoe)
	if spec != nil {
		if err := spec.UnmarshalTo(specItem); err != nil {
			return fmt.Errorf("could not unmarshal %s: %s", itemName, err)
		}
	}

	pppoeItem := new(flow.RteFlowItemPppoe)
	if err := item.UnmarshalTo(pppoeItem); err != nil {
		return fmt.Errorf("could not unmarshal %s: %s", itemName, err)
	}

	if specItem.VersionType != 0 {
		if err := validateUint32ItemField(itemName, "version_type", specItem.VersionType, pppoeItem.VersionType); err != nil {
			return err
		}
	}
	if specItem.Code != 0 {
		if err := validateUint32ItemField(itemName, "code", specItem.Code, pppoeItem.Code); err != nil {
			return err
		}
	}
	if specItem.SessionId != 0 {
		if err := validateUint32ItemField(itemName, "session_id", specItem.SessionId, pppoeItem.SessionId); err != nil {
			return err
		}
	}
	if specItem.Length != 0 {
		if err := validateUint32ItemField(itemName, "length", specItem.Length, pppoeItem.Length); err != nil {
			return err
		}
	}

	if pppoeItem.VersionType > 0xFF {
		return fmt.Errorf("invalid version_type (%d), must be equal or lower than 255", pppoeItem.VersionType)
	}
	if pppoeItem.Code > 0xFF {
		return fmt.Errorf("invalid code (%d), must be equal or lower than 255", pppoeItem.Code)
	}
	if pppoeItem.SessionId > 0xFFFF {
		return fmt.Errorf("invalid session_id (%d), must be equal or lower than 65,535", pppoeItem.SessionId)
	}
	if pppoeItem.Length > 0xFFFF {
		return fmt.Errorf("invalid length (%d), must be equal or lower than 65,535", pppoeItem.Length)
	}

	return nil
}

func validateRteFlowItemPppoeProtoId(itemName string, spec, item *any.Any) error {
	specItem := new(flow.RteFlowItemPppoeProtoId)
	if spec != nil {
		if err := spec.UnmarshalTo(specItem); err != nil {
			return fmt.Errorf("could not unmarshal spec (%v) %v", spec, err)
		}
	}

	pppoeProtoIdItem := new(flow.RteFlowItemPppoeProtoId)
	if err := item.UnmarshalTo(pppoeProtoIdItem); err != nil {
		return fmt.Errorf("could not unmarshal %s: %v", itemName, err)
	}

	if err := validateUint32ItemField(itemName, "proto_id", specItem.ProtoId, pppoeProtoIdItem.ProtoId); err != nil {
		return err
	}

	if pppoeProtoIdItem.ProtoId > 0xFFFF {
		return fmt.Errorf("invalid proto_id (%d), must be equal or lower than 65,535", pppoeProtoIdItem.ProtoId)
	}

	return nil
}

func validateUint32ItemField(itemName, field string, specValue, fieldValue uint32) error {
	if itemName == "spec" {
		return nil
	}
	if specValue == 0 && fieldValue != 0 {
		return fmt.Errorf("spec.%s must be specified", field)
	}
	if itemName == "last" {
		if fieldValue < specValue {
			return fmt.Errorf("last.%s (%d) must be higher or equal to spec: %s (%d)", field, fieldValue, field, specValue)
		}
	}

	return nil
}

// RteFlowAction Validation

func validateRteFlowActionConfigEmpty(spec *any.Any) error {
	if spec != nil {
		return fmt.Errorf("unexpected key 'conf': action is not configurable")
	}
	return nil
}

func validateRteFlowActionVf(spec *any.Any) error {
	conf := new(flow.RteFlowActionVf)

	if err := spec.UnmarshalTo(conf); err != nil {
		return fmt.Errorf("could not unmarshal RTE_FLOW_ACTION_TYPE_VF configuration: %s", err)
	}

	// CVL supports up to 256 VFs
	if conf.Id > 255 {
		return fmt.Errorf("'id' must be in 0-255 range")
	}
	// whether to use original VF, uses 1 bit so it can be zero or one
	if conf.Original > 1 {
		return fmt.Errorf("'original' must be 0 or 1")
	}
	if conf.Reserved > 0 {
		return fmt.Errorf("'reserved' field can't be non-zero")
	}

	return nil
}
