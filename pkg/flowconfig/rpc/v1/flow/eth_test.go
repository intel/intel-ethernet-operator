// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flow

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func TestEth_ToRteFlowItemEth(t *testing.T) {
	tests := []struct {
		name        string
		testData    *Eth
		wantItemEth *Eth
		wantErr     bool
	}{
		{
			name: "Test correct data",
			testData: &Eth{
				"00:00:5f:00:53:01",
				"00:00:5e:00:53:01",
				0x8100,
			},
			wantItemEth: &Eth{
				Dst:  "00005f005301",
				Src:  "00005e005301",
				Type: 0x8100,
			},
			wantErr: false,
		},
		{
			name: "Test incorrect dst",
			testData: &Eth{
				"00:00:00:00:0000",
				"00:00:5e:00:53:01",
				0x8100,
			},
			wantItemEth: nil,
			wantErr:     true,
		},
		{
			name: "Test incorrect src",
			testData: &Eth{
				"00:00:5f:00:53:01",
				"00:00:^&:00:5%:01",
				0x8100,
			},
			wantItemEth: nil,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			gotRteFlowItemEth, err := tt.testData.ToRteFlowItemEth()
			if (err != nil) != tt.wantErr {
				t.Errorf("Eth.ToRteFlowItemEth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRteFlowItemEth == nil && tt.wantItemEth != nil {
				t.Errorf("Eth.ToRteFlowItemEth() = %v, want %v", gotRteFlowItemEth, tt.wantItemEth)
			}
			if gotRteFlowItemEth != nil {

				srcBytes := gotRteFlowItemEth.Src.AddrBytes
				encodedSrc := hex.EncodeToString(srcBytes)

				dstBytes := gotRteFlowItemEth.Dst.AddrBytes
				encodedDst := hex.EncodeToString(dstBytes)

				gotItemEth := &Eth{
					Dst:  encodedDst,
					Src:  encodedSrc,
					Type: gotRteFlowItemEth.Type,
				}
				if !reflect.DeepEqual(gotItemEth, tt.wantItemEth) {
					t.Errorf("Eth.ToRteFlowItemEth() = %v, want %v", gotItemEth, tt.wantItemEth)
				}
			}

		})
	}
}
