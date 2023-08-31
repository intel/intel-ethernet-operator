// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2023 Intel Corporation

package v1

import (
	"bytes"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterFlowAction defines flow actions
type ClusterFlowAction struct {
	Type ClusterFlowActionType `json:"type,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Conf *runtime.RawExtension `json:"conf,omitempty"`
}

// ToPodInterfaceConf is configuration for type ToPodInterface
type ToPodInterfaceConf struct {
	NetInterfaceName string `json:"podInterface,omitempty"`
}

// +kubebuilder:validation:Type=string
type ClusterFlowActionType int

const (
	ToPodInterface ClusterFlowActionType = 20000 // number has to be greater than last RteFlowItemType_RTE_FLOW_ITEM_TYPE_... defined in flow.pb.go
)

var clusterFlowActionTypeName = map[ClusterFlowActionType]string{
	ToPodInterface: "to-pod-interface",
}

var clusterFlowActionTypeValue = map[string]ClusterFlowActionType{
	"to-pod-interface": ToPodInterface,
}

func (ct ClusterFlowActionType) String() string {
	return clusterFlowActionTypeName[ct]
}

// MarshalJSON marshals the enum as a quoted json string as kept in clusterFlowActionTypeName map
func (s ClusterFlowActionType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(clusterFlowActionTypeName[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmarshals a quoted json string to the enum value as kept in clusterFlowActionTypeValue map
func (s *ClusterFlowActionType) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}

	if j == "" {
		return fmt.Errorf("json unmarshalling produced an empty string")
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	v, ok := clusterFlowActionTypeValue[j]
	if !ok {
		return fmt.Errorf("unsupported type string %s", j)
	}
	*s = v
	return nil
}

// ClusterFlowActionToString returns string represention of ClusterFlowActionType
// It returns empty string for unknown/unsupported type. Caller must handle empty string.
func ClusterFlowActionToString(actionType ClusterFlowActionType) string {
	if s, ok := clusterFlowActionTypeName[actionType]; ok {
		return s
	}
	return ""
}
