// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package utils

import (
	"testing"

	sriovutils "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	flowapi "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
)

func TestGetFlowActionAny(t *testing.T) {

	actionData := []struct {
		name              string
		Type              string
		Conf              []byte
		expectedErr       bool
		expectedAny       bool
		isCalledByWebhook bool
	}{
		{
			name: "tc1",
			Type: "RTE_FLOW_ACTION_TYPE_VF", Conf: []byte(`{ "id": 1 }`),
			expectedErr: false, expectedAny: true, isCalledByWebhook: false,
		},
		{
			name: "tc2",
			Type: "RTE_FLOW_ACTION_TYPE_END", Conf: []byte(`{}`),
			expectedErr: false, expectedAny: true, isCalledByWebhook: false,
		},
		{
			name: "tc3",
			Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR", Conf: []byte(`{}`),
			expectedErr: true, expectedAny: false, isCalledByWebhook: false,
		},
		{
			name: "tc4",
			Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR_OTHER", Conf: []byte(`{}`),
			expectedErr: true, expectedAny: false, isCalledByWebhook: false,
		},
		{
			name: "tc5",
			Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR", Conf: []byte(`{"addr":"0000:01:10.0"}`),
			expectedErr: false, expectedAny: true, isCalledByWebhook: false,
		},
		{
			name: "tc6",
			Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR", Conf: []byte(`{"addr":"0000:01:11.0"}`),
			expectedErr: true, expectedAny: false, isCalledByWebhook: false,
		},
		{
			name: "tc7",
			Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR", Conf: []byte(`{"ip":"0000:01:11.0"}`),
			expectedErr: true, expectedAny: false, isCalledByWebhook: false,
		},
		{
			name: "tc5_webhook",
			Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR", Conf: []byte(`{"addr":"0000:01:10.0"}`),
			expectedErr: false, expectedAny: true, isCalledByWebhook: true,
		},
		{
			name: "tc6_webhook",
			Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR", Conf: []byte(`{"addr":"0000:01:11.0"}`),
			expectedErr: true, expectedAny: false, isCalledByWebhook: true,
		},
		{
			name: "tc7_webhook",
			Type: "RTE_FLOW_ACTION_TYPE_VFPCIADDR", Conf: []byte(`{"ip":"0000:01:11.0"}`),
			expectedErr: true, expectedAny: false, isCalledByWebhook: true,
		},
	}

	fs := &sriovutils.FakeFilesystem{
		Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/", "sys/bus/pci/devices/0000:01:00.0/"},
		Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:10.0/physfn": "../0000:01:00.0",
			"sys/bus/pci/devices/0000:01:00.0/virtfn0": "../0000:01:08.0",
			"sys/bus/pci/devices/0000:01:00.0/virtfn1": "../0000:01:09.0",
			"sys/bus/pci/devices/0000:01:00.0/virtfn2": "../0000:01:10.0",
		},
	}
	defer fs.Use()()

	for _, item := range actionData {
		t.Run(item.name, func(t *testing.T) {
			any, err := GetFlowActionAny(item.Type, item.Conf, item.isCalledByWebhook)
			if err != nil && !item.expectedErr {
				t.Errorf("%v", err)
			}

			if item.expectedAny && any == nil {
				t.Errorf("Any object unexpected result: %v should be %v", any, item.expectedAny)
			}
		})
	}
}

func TestGetFlowItemAny(t *testing.T) {

	itemData := []struct {
		Type string
		Spec []byte
	}{
		{
			Type: "RTE_FLOW_ITEM_TYPE_IPV4",
			Spec: []byte(`{
				"hdr": {
					"dst_addr": "1.1.1.1"
				}
			}`),
		},
		{
			Type: "RTE_FLOW_ITEM_TYPE_END",
			Spec: []byte(`{}`),
		},
	}

	for _, item := range itemData {
		any, err := GetFlowItemAny(item.Type, item.Spec)
		if err != nil {
			t.Errorf("%v", err)
		}
		_ = any
	}
}

func TestItemAnyObjects(t *testing.T) {
	item := struct {
		Type string
		Spec []byte
	}{
		Type: "RTE_FLOW_ITEM_TYPE_IPV4",
		Spec: []byte(`{
						"hdr": {
							"dst_addr": "1.1.1.1"
						}
				}`),
	}

	anyObj, err := GetFlowItemAny(item.Type, item.Spec)
	if err != nil {
		t.Errorf("%v", err)
	}
	ipv4 := &flowapi.RteFlowItemIpv4{}

	if err := anyObj.UnmarshalTo(ipv4); err != nil {
		t.Errorf("%v", err)
	}
}
