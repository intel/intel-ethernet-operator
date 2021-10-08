// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package v1

import (
	"encoding/json"
	"testing"

	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/utils"

	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name: "invalid eth mask config",
			data: `
---
apiVersion: sriov.intel.com/v1
kind: NodeFlowConfig
metadata:
    name: silpixa00398610d
spec:
  rules:
    - pattern:
        - type: RTE_FLOW_ITEM_TYPE_ETH
          spec:
            dst: "00:00:5f:00:53:01"
            src: "00:00:5e:00:53:01"
            type: 0x8100
          mask:
            dst: "ff:ff:ffff:ff:00"
            src: "ff:ff:ff:ff:ff:00"
            type: 0xFFFF
        - type: RTE_FLOW_ITEM_TYPE_IPV4
          spec:
            hdr:
              src_addr: 192.168.11.10
              dst_addr: 192.168.10.0
        - type: RTE_FLOW_ITEM_TYPE_END
      action:
        - type: RTE_FLOW_ACTION_TYPE_VF
          conf:
            id: 1
        - type: RTE_FLOW_ACTION_TYPE_END
      attr:
        ingress: 1
`,
			wantErr: true,
		},
		{
			name: "invalid eth last config",
			data: `
---
apiVersion: sriov.intel.com/v1
kind: NodeFlowConfig
metadata:
    name: silpixa00398610d
spec:
  rules:
    - pattern:
        - type: RTE_FLOW_ITEM_TYPE_ETH
          spec:
            dst: "00:00:5f:00:53:01"
            src: "00:00:5e:00:53:01"
            type: 0x8100
          last:
            dst: "00:00:5f:00:53:01"
            src: "00:00:5e:00:33:01"
            type: 0xFFFF
        - type: RTE_FLOW_ITEM_TYPE_IPV4
          spec:
            hdr:
              src_addr: 192.168.11.10
              dst_addr: 192.168.10.0
        - type: RTE_FLOW_ITEM_TYPE_END
      action:
        - type: RTE_FLOW_ACTION_TYPE_VF
          conf:
            id: 1
        - type: RTE_FLOW_ACTION_TYPE_END
      attr:
        ingress: 1
`,
			wantErr: true,
		},
		{
			name: "valid eth config",
			data: `
---
apiVersion: sriov.intel.com/v1
kind: NodeFlowConfig
metadata:
  name: silpixa00398610d
spec:
  rules:
    - pattern:
        - type: RTE_FLOW_ITEM_TYPE_ETH
          spec:
            dst: "00:00:5f:00:53:01"
            src: "00:00:5e:00:53:01"
            type: 0x8100
          mask:
            dst: "ff:ff:ff:ff:ff:00"
            src: "ff:ff:ff:ff:ff:00"
            type: 0xFFFF
          last:
            dst: "00:00:5f:00:63:10"
        - type: RTE_FLOW_ITEM_TYPE_IPV4
          spec:
            hdr:
              src_addr: 192.168.11.10
              dst_addr: 192.168.10.0
        - type: RTE_FLOW_ITEM_TYPE_END
      action:
        - type: RTE_FLOW_ACTION_TYPE_VF
          conf:
            id: 1
        - type: RTE_FLOW_ACTION_TYPE_END
      attr:
        ingress: 1
`,
			wantErr: false,
		},
		{
			name: "invalid config",
			data: `
---
apiVersion: sriov.intel.com/v1
kind: NodeFlowConfig
metadata:
  name: silpixa00398610d
spec:
  rules:
    - pattern:
        - type: RTE_FLOW_ITEM_TYPE_ETH
          spec:
            dst: "00:00:5f:00:53:01"
            src: "00:00:5e00:53:01"
            type: 0x8100
          mask:
            dst: "ff:ff:ff:ff:ff:00"
            src: "ff:ff:ff:ff:ff:00"
            type: 0xFFFF
        - type: RTE_FLOW_ITEM_TYPE_IPV4
          spec:
            hdr:
              src_addr: 192.168.11.10
              dst_addr: 192.168.10.0
        - type: RTE_FLOW_ITEM_TYPE_END
      action:
        - type: RTE_FLOW_ACTION_TYPE_VF
          conf:
            id: 1
        - type: RTE_FLOW_ACTION_TYPE_END
      attr:
        ingress: 1
`,
			wantErr: true,
		},
		{
			name: "empty pattern",
			data: `
---
apiVersion: sriov.intel.com/v1
kind: NodeFlowConfig
metadata:
  name: silpixa00398610d
spec:
  rules:
    - pattern:
      action:
        - type: RTE_FLOW_ACTION_TYPE_VF
          conf:
            id: 1
        - type: RTE_FLOW_ACTION_TYPE_END
      attr:
        ingress: 1
`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &NodeFlowConfig{}

			jObj, _ := yaml.ToJSON([]byte(tt.data))

			if err := json.Unmarshal(jObj, config); err != nil {
				t.Errorf("error decoding yaml into NodeFlowConfig object: %v", err)
			}

			spec := config.Spec
			for _, rule := range spec.Rules {
				if err := validate(rule); (err != nil) != tt.wantErr {
					t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}

}

func TestValidateRteFlowActionVf(t *testing.T) {
	tests := []struct {
		name    string
		conf    []byte
		wantErr bool
	}{
		{name: "valid config", conf: []byte(`{"Id":255,"Original":0,"Reserved":0}`), wantErr: false},
		{name: "invalid config: Id", conf: []byte(`{"Id":256,"Original":1,"Reserved":0}`), wantErr: true},
		{name: "invalid config: Original", conf: []byte(`{"Id":255,"Original":2,"Reserved":0}`), wantErr: true},
		{name: "invalid config: Reserved", conf: []byte(`{"Id":255,"Original":0,"Reserved":1}`), wantErr: true},
		// {name: "empty config", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &FlowAction{Type: "RTE_FLOW_ACTION_TYPE_VF"}
			rteFlowAction := new(flow.RteFlowAction)

			if tt.conf != nil {
				action.Conf = &runtime.RawExtension{Raw: tt.conf}
			} else {
				action.Conf = nil
			}

			if action.Conf != nil {
				var err error
				rteFlowAction.Conf, err = utils.GetFlowActionAny(action.Type, action.Conf.Raw)
				if err != nil {
					t.Errorf("error: %s", err)
				}
			} else {
				rteFlowAction.Conf = nil
			}

			if err := validateRteFlowActionVf(rteFlowAction.Conf); (err != nil) != tt.wantErr {
				t.Errorf("validateRteFlowActionVf() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRteFlowItemEth(t *testing.T) {
	tests := []struct {
		name    string
		item    *FlowItem
		wantErr bool
	}{
		{
			name: "valid item",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_ETH",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "dst": "00:00:5f:00:53:01", "src": "00:00:5e:00:53:01", "type": 8100 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "dst": "00:00:5f:00:53:01", "src": "00:00:5e:00:53:01", "type": 8200 }`),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid last item type",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_ETH",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "dst": "00:00:5f:00:53:01", "src": "00:00:5e:00:53:01", "type": 8100 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "dst": "00:00:5f:00:53:01", "src": "00:00:5e:00:53:01", "type": 8000 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid last item dst",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_ETH",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "dst": "00:00:5f:00:53:01", "src": "00:00:5e:00:53:01", "type": 8100 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "dst": "00:00:5f:00:43:01", "src": "00:00:5e:00:53:01", "type": 8200 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid last item src",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_ETH",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "dst": "00:00:5f:00:53:01", "src": "00:00:5e:00:53:01", "type": 8100 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "dst": "00:00:5f:00:73:01", "src": "00:00:5e:00:43:01", "type": 8200 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "mask.src with empty spec.src",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_ETH",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "type": 8100 }`),
				},
				Mask: &runtime.RawExtension{
					Raw: []byte(`{ "src": "00:00:5f:00:73:01", "type": 8200 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid eth type",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_ETH",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "src": "00:00:5f:00:73:01", "type": 87654 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "valid item spec",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_ETH",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "dst": "00:00:5f:00:73:01", "src": "00:00:5e:00:43:01", "type": 8200 }`),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := validateRteFlowItem(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRteFlowItem() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRteFlowItemVlan(t *testing.T) {
	tests := []struct {
		name    string
		item    *FlowItem
		wantErr bool
	}{
		{
			name: "valid item",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_VLAN",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "tci": 8100 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "tci": 8200 }`),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid last item inner_type",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_VLAN",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "inner_type": 100 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "inner_type": 50 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid last item tci",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_VLAN",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "tci": 1100 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "tci": 550 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "valid mask",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_VLAN",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "tci": 1100 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "tci": 65535 }`),
				},
			},
			wantErr: false,
		},
		{
			name: "mask.tci with empty spec.tci",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_VLAN",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "inner_type": 1100 }`),
				},
				Mask: &runtime.RawExtension{
					Raw: []byte(`{ "tci": 65535 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid inner_type",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_VLAN",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "inner_type": 75535 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid tci",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_VLAN",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "tci": 75535 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid vlan data",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_VLAN",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "tci": 12-5 }`),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := validateRteFlowItem(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRteFlowItem() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRteFlowItemIPv4(t *testing.T) {
	tests := []struct {
		name    string
		item    *FlowItem
		wantErr bool
	}{
		{
			name: "invalid version_ihl",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "version_ihl": 256 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid type_of_service",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "type_of_service": 256 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid total_length",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "total_length": 65536 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid packet_id",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "packet_id": 65536 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid fragment_offset",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "fragment_offset": 65536 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid time_to_live",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "time_to_live": 256 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid next_proto_id",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "next_proto_id": 256 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid hdr_checksum",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "hdr_checksum": 256 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid version_ihl last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "version_ihl": 120 } }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "version_ihl": 50} }`),
				},
			},
			wantErr: true,
		},
		{
			name: "valid type_of_service mask",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "type_of_service": 100 } }`),
				},
				Mask: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "type_of_service": 255 } }`),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid total_length last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "total_length": 65100 } }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "total_length": 6500 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid packet_id last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "packet_id": 65100 } }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "packet_id": 60100 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid fragment_offset last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{"hdr": { "fragment_offset": 12345 } }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{"hdr": { "fragment_offset": 3456 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid time_to_live last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "time_to_live": 125} }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "time_to_live": 105 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid next_proto_id last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "next_proto_id": 25 } }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "next_proto_id": 5 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid hdr_checksum last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_IPV4",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "hdr_checksum": 123 } }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "hdr_checksum": 5 } }`),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateRteFlowItem(tt.item); (err != nil) != tt.wantErr {
				t.Errorf("validateRteFlowItem() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRteFlowItemUdp(t *testing.T) {
	tests := []struct {
		name    string
		item    *FlowItem
		wantErr bool
	}{
		{
			name: "invalid src_port",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_UDP",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "src_port": 75535 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid dst_port",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_UDP",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "dst_port": 75535 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid dgram_len",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_UDP",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "dgram_len": 75535 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid dgram_cksum",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_UDP",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "dgram_cksum": 75535 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid src_port last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_UDP",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "src_port": 120 } }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": {"src_port": 50} }`),
				},
			},
			wantErr: true,
		},
		{
			name: "valid dst_port mask",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_UDP",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "dst_port": 100 } }`),
				},
				Mask: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "dst_port": 255 } }`),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid dgram_len last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_UDP",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "dgram_len": 123 } }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "dgram_len": 121 } }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid dgram_cksum last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_UDP",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "dgram_cksum": 432 } }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "hdr": { "dgram_cksum": 123 } }`),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateRteFlowItem(tt.item); (err != nil) != tt.wantErr {
				t.Errorf("validateRteFlowItem() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRteFlowItemPppoe(t *testing.T) {
	tests := []struct {
		name    string
		item    *FlowItem
		wantErr bool
	}{
		{
			name: "invalid version_type",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOES",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "version_type": 256 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid code",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOES",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "code": 256 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid session_id",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOES",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "session_id": 65537 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid length",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOES",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "length": 65537 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid version_type last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOES",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "version_type": 120 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "version_type": 50 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "valid code mask",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOES",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "code": 100 }`),
				},
				Mask: &runtime.RawExtension{
					Raw: []byte(`{ "code": 255 }`),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid session_id last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOES",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "session_id": 65530 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "session_id": 55531 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid length last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOES",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "length": 65530 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "length": 55531 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid length data",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOES",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "length": "invalid" }`),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateRteFlowItem(tt.item); (err != nil) != tt.wantErr {
				t.Errorf("validateRteFlowItem() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRteFlowItemPppoeProtoId(t *testing.T) {
	tests := []struct {
		name    string
		item    *FlowItem
		wantErr bool
	}{
		{
			name: "invalid proto_id",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOE_PROTO_ID",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "proto_id": 95535 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "valid proto_id mask",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOE_PROTO_ID",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "proto_id": 100 }`),
				},
				Mask: &runtime.RawExtension{
					Raw: []byte(`{ "proto_id": 65535 }`),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid proto_id last",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOE_PROTO_ID",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "proto_id": 65530 }`),
				},
				Last: &runtime.RawExtension{
					Raw: []byte(`{ "proto_id": 55531 }`),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid proto_id data",
			item: &FlowItem{
				Type: "RTE_FLOW_ITEM_TYPE_PPPOE_PROTO_ID",
				Spec: &runtime.RawExtension{
					Raw: []byte(`{ "proto_id": sdwet52 }`),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateRteFlowItem(tt.item); (err != nil) != tt.wantErr {
				t.Errorf("validateRteFlowItem() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRteFlowAction(t *testing.T) {
	tests := []struct {
		name       string
		actionType string
		spec       []byte
		wantErr    bool
	}{
		{name: "non-empty config", actionType: "RTE_FLOW_ACTION_TYPE_DROP", spec: []byte(`{"field":"value"}`), wantErr: true},
		{name: "empty spec", actionType: "RTE_FLOW_ACTION_TYPE_DROP", wantErr: false},
		{name: "unsupported action type", actionType: "RTE_FLOW_ACTION_TYPE_SET_META", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &FlowAction{Type: tt.actionType}
			rteFlowAction := new(flow.RteFlowAction)
			aType := flow.RteFlowActionType_value[action.Type]
			rteFlowAction.Type = flow.RteFlowActionType(aType)

			if tt.spec != nil {
				action.Conf = &runtime.RawExtension{Raw: tt.spec}
			} else {
				action.Conf = nil
			}
			if action.Conf != nil {
				var err error
				rteFlowAction.Conf, err = utils.GetFlowActionAny(action.Type, action.Conf.Raw)
				if err != nil {
					t.Errorf("error: %s", err)
				}
			} else {
				rteFlowAction.Conf = nil
			}

			if err := validateRteFlowAction(rteFlowAction); (err != nil) != tt.wantErr {
				t.Errorf("validateRteFlowAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

var (
	policy1 = &NodeFlowConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "sriov.intel.com/v1",
			Kind:       "NodeFlowConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testk8snode",
			Namespace: NodeFlowConfigNamespace,
		},
		Spec: NodeFlowConfigSpec{},
	}
	old = &runtime.Unknown{}
)

func TestValidateCreate(t *testing.T) {
	tests := []struct {
		name    string
		rules   []*FlowRules
		wantErr bool
	}{
		{
			name:    "Create NodeFlowConfigSpec with invalid Eth Field",
			rules:   invalidEthFieldName,
			wantErr: true,
		},
		{
			name:    "Create NodeFlowConfigSpec with invalid Vlan Item Field",
			rules:   invalidVlanFieldOutOfRange,
			wantErr: true,
		},
		{
			name:    "Create NodeFlowConfigSpec with invalid IPv4 Last",
			rules:   invalidIpv4LastLowerThanSpec,
			wantErr: true,
		},
		{
			name:    "Create NodeFlowConfigSpec with invalid IPv4 Last - value out of range",
			rules:   invalidIpv4LastFieldOutOfRange,
			wantErr: true,
		},
		{
			name:    "Create NodeFlowConfigSpec with invalid Udp spec",
			rules:   invalidUdpSpecFieldOutOfRange,
			wantErr: true,
		},
		{
			name:    "Create NodeFlowConfigSpec with invalid Pppoe Field",
			rules:   invalidPppoeFieldOutOfRange,
			wantErr: true,
		},
		{
			name:    "Create NodeFlowConfigSpec with invalid Pppoe Proto ID Field",
			rules:   invalidPppoeProtoIdFieldOutOfRange,
			wantErr: true,
		},
		{
			name:    "Create NodeFlowConfigSpec with valid but not supported Item",
			rules:   validUnsupportedItem,
			wantErr: false,
		},
		{
			name:    "Create NodeFlowConfigSpec with last with empty spec",
			rules:   lastWithEmptySpec,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy1.Spec.Rules = tt.rules

			if err := policy1.ValidateCreate(); (err != nil) != tt.wantErr {
				t.Errorf("NodeFlowConfig.ValidateCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	policy1.Spec.Rules = validUnsupportedItem
	t.Run("Updating NodeFlowConfigSpec with valid but not supported Item", func(t *testing.T) {
		if err := policy1.ValidateUpdate(old); (err != nil) != false {
			t.Errorf("NodeFlowConfig.ValidateUpdate() error = %v, wantErr %v", err, false)
		}
	})
}

func TestValidateDelete(t *testing.T) {
	t.Run("Deleting NodeFlowConfigSpec with valid but not supported Item", func(t *testing.T) {
		if err := policy1.ValidateDelete(); (err != nil) != false {
			t.Errorf("NodeFlowConfig.ValidateUpdate() error = %v, wantErr %v", err, false)
		}
	})
}
