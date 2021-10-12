// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flow

import (
	"fmt"
	"net"
)

// Eth is the intermediary struct to unmarshall Eth Mac addresses
type Eth struct {
	Dst  string `json:"dst,omitempty"`
	Src  string `json:"src,omitempty"`
	Type uint32 `json:"type,omitempty"`
}

// ToRteFlowItemEth converts Eth item from Json to correct rteFlowItemEth type object
func (eth *Eth) ToRteFlowItemEth() (rteFlowItemEth *RteFlowItemEth, err error) {
	rteFlowItemEth = &RteFlowItemEth{Type: eth.Type}

	// Convert Src and Dst MAC address from string to []byte
	ethDst := &RteEtherAddr{}
	var dstBytes []byte
	if eth.Dst != "" {
		dstBytes, err = net.ParseMAC(eth.Dst)
		if err != nil {
			return nil, fmt.Errorf("could not parse MAC address (%s)", eth.Dst)
		}
		ethDst.AddrBytes = dstBytes
		rteFlowItemEth.Dst = ethDst
	}

	ethSrc := &RteEtherAddr{}
	var srcBytes []byte
	if eth.Src != "" {
		srcBytes, err = net.ParseMAC(eth.Src)
		if err != nil {
			return nil, fmt.Errorf("could not parse MAC address (%s)", eth.Src)
		}
		ethSrc.AddrBytes = srcBytes
		rteFlowItemEth.Src = ethSrc
	}

	return rteFlowItemEth, nil
}
