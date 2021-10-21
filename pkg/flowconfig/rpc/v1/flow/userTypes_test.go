// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flow

import "testing"

func TestGetFlowActionType(t *testing.T) {
	items := []struct {
		name        string
		actionType  string
		expectedVal int32
		expectedOk  bool
	}{
		{name: "valid vfPciAddr", actionType: "RTE_FLOW_ACTION_TYPE_VFPCIADDR", expectedVal: 11, expectedOk: true},
		{name: "valid vf", actionType: "RTE_FLOW_ACTION_TYPE_VF", expectedVal: 11, expectedOk: true},
		{name: "invalid type", actionType: "RTE_FLOW_ACTION_TYPE_UNKNOWN_GO", expectedVal: 0, expectedOk: false},
	}

	for _, item := range items {
		t.Run(item.name, func(t *testing.T) {
			val, ok := GetFlowActionType(item.actionType)
			if ok != item.expectedOk || val != item.expectedVal {
				t.Errorf("Expected val %v vs %v, Expected OK %v vs %v", item.expectedVal, val, item.expectedOk, ok)
			}
		})
	}
}
