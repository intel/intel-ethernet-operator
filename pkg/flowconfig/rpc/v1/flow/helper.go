// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flow

import (
	"google.golang.org/protobuf/types/known/emptypb"
)

func GetFlowItemObj(itemType RteFlowItemType) interface{} {
	switch itemType {
	// Types with no associated struct
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_END,
		RteFlowItemType_RTE_FLOW_ITEM_TYPE_VOID,
		RteFlowItemType_RTE_FLOW_ITEM_TYPE_INVERT,
		RteFlowItemType_RTE_FLOW_ITEM_TYPE_PF:
		return new(emptypb.Empty)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_ANY:
		return new(RteFlowItemAny)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_VF:
		return new(RteFlowItemVf)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_PHY_PORT:
		return new(RteFlowItemPhyPort)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_PORT_ID:
		return new(RteFlowItemPortId)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_RAW:
		return new(RteFlowItemRaw)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_ETH:
		return new(RteFlowItemEth)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_VLAN:
		return new(RteFlowItemVlan)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_IPV4:
		return new(RteFlowItemIpv4)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_IPV6:
		return new(RteFlowItemIpv6)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_ICMP:
		return new(RteFlowItemIcmp)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_UDP:
		return new(RteFlowItemUdp)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_TCP:
		return new(RteFlowItemTcp)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_SCTP:
		return new(RteFlowItemSctp)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_VXLAN:
		return new(RteFlowItemVxlan)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_E_TAG:
		return new(RteFlowItemETag)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_NVGRE:
		return new(RteFlowItemNvgre)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_MPLS:
		return new(RteFlowItemMpls)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_GRE:
		return new(RteFlowItemGre)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_FUZZY:
		return new(RteFlowItemFuzzy)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_GTP,
		RteFlowItemType_RTE_FLOW_ITEM_TYPE_GTPC,
		RteFlowItemType_RTE_FLOW_ITEM_TYPE_GTPU:
		return new(RteFlowItemGtp)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_ESP:
		return new(RteFlowItemEsp)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_GENEVE:
		return new(RteFlowItemGeneve)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_VXLAN_GPE:
		return new(RteFlowItemVxlanGpe)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_ARP_ETH_IPV4:
		return new(RteFlowItemArpEthIpv4)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_IPV6_EXT:
		return new(RteFlowItemIpv6Ext)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_ICMP6:
		return new(RteFlowItemIcmp6)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_ICMP6_ND_NS:
		return new(RteFlowItemIcmp6NdNs)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_ICMP6_ND_NA:
		return new(RteFlowItemIcmp6NdNa)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_ICMP6_ND_OPT:
		return new(RteFlowItemIcmp6NdOpt)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_ICMP6_ND_OPT_SLA_ETH:
		return new(RteFlowItemIcmp6NdOptSlaEth)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_ICMP6_ND_OPT_TLA_ETH:
		return new(RteFlowItemIcmp6NdOptStaEth)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_MARK:
		return new(RteFlowItemMark)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_META:
		return new(RteFlowItemMeta)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_GRE_KEY:
		return new(RteFlowItemGre)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_GTP_PSC:
		return new(RteFlowItemGtpPsc)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_PPPOES,
		RteFlowItemType_RTE_FLOW_ITEM_TYPE_PPPOED:
		return new(RteFlowItemPppoe)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_PPPOE_PROTO_ID:
		return new(RteFlowItemPppoeProtoId)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_NSH:
		return new(RteFlowItemNsh)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_IGMP:
		return new(RteFlowItemIgmp)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_AH:
		return new(RteFlowItemAh)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_HIGIG2:
		return new(RteFlowItemHigig2Hdr)
	case RteFlowItemType_RTE_FLOW_ITEM_TYPE_TAG:
		return new(RteFlowItemTag)
	default:
		return nil
	}
}

func GetFlowActionObj(actionType RteFlowActionType) interface{} {
	switch actionType {
	// Types with no associated struct
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_END,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_VOID,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_PASSTHRU,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_FLAG,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_DROP,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_PF,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_DEC_MPLS_TTL,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_DEC_NW_TTL,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_COPY_TTL_OUT,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_COPY_TTL_IN,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_POP_VLAN,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_VXLAN_DECAP,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_NVGRE_DECAP,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_MAC_SWAP,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_DEC_TTL,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_INC_TCP_SEQ,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_DEC_TCP_SEQ,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_INC_TCP_ACK,
		RteFlowActionType_RTE_FLOW_ACTION_TYPE_DEC_TCP_ACK:
		return new(emptypb.Empty)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_JUMP:
		return new(RteFlowActionJump)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_MARK:
		return new(RteFlowActionMark)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_QUEUE:
		return new(RteFlowActionQueue)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_COUNT:
		return new(RteFlowActionCount)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_RSS:
		return new(RteFlowActionRss)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_VF:
		return new(RteFlowActionVf)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_PHY_PORT:
		return new(RteFlowActionPhyPort)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_PORT_ID:
		return new(RteFlowActionPortId)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_METER:
		return new(RteFlowActionMeter)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SECURITY:
		return new(RteFlowActionSecurity)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_SET_MPLS_TTL:
		return new(RteFlowActionOfSetMplsTtl)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_SET_NW_TTL:
		return new(RteFlowActionOfSetNwTtl)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_PUSH_VLAN:
		return new(RteFlowActionOfPushVlan)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_SET_VLAN_VID:
		return new(RteFlowActionOfSetVlanVid)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_SET_VLAN_PCP:
		return new(RteFlowActionOfSetVlanPcp)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_POP_MPLS:
		return new(RteFlowActionOfPopMpls)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_OF_PUSH_MPLS:
		return new(RteFlowActionOfPushMpls)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_VXLAN_ENCAP:
		return new(RteFlowActionVxlanEncap)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_NVGRE_ENCAP:
		return new(RteFlowActionNvgreEncap)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_RAW_ENCAP:
		return new(RteFlowActionRawEncap)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_RAW_DECAP:
		return new(RteFlowActionRawDecap)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_IPV4_SRC:
		return new(RteFlowActionSetIpv4)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_IPV4_DST:
		return new(RteFlowActionSetIpv4)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_IPV6_SRC:
		return new(RteFlowActionSetIpv6)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_IPV6_DST:
		return new(RteFlowActionSetIpv6)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_TP_SRC:
		return new(RteFlowActionSetTp)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_TP_DST:
		return new(RteFlowActionSetTp)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_TTL:
		return new(RteFlowActionSetTtl)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_MAC_SRC:
		return new(RteFlowActionSetMac)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_MAC_DST:
		return new(RteFlowActionSetMac)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_TAG:
		return new(RteFlowActionSetTag)
	case RteFlowActionType_RTE_FLOW_ACTION_TYPE_SET_META:
		return new(RteFlowActionSetMeta)
	default:
		return nil
	}
}
