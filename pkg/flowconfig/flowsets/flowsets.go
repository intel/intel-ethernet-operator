// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

package flowsets

import (
	flowapi "github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
)

type FlowCreateRecord struct {
	FlowID   uint32
	FlowRule *flowapi.RequestFlowCreate
}

func newFlowCreateRecord(id uint32, rule *flowapi.RequestFlowCreate) *FlowCreateRecord {
	return &FlowCreateRecord{
		FlowID:   id,
		FlowRule: rule,
	}
}

type FlowSets struct {
	data map[string]*FlowCreateRecord
}

func (s *FlowSets) Add(key string, id uint32, rule *flowapi.RequestFlowCreate) *FlowSets {
	if s.data == nil {
		s.data = make(map[string]*FlowCreateRecord)
	}
	if _, ok := s.data[key]; !ok {
		s.data[key] = newFlowCreateRecord(id, rule)
	}

	return s
}

func (s *FlowSets) Delete(key string) {
	delete(s.data, key)
}

func (s *FlowSets) Has(key string) bool {
	_, ok := s.data[key]
	return ok
}

func (s *FlowSets) Size() int {
	return len(s.data)
}

// GetCompliments returns a list of flowRecords with keys that are not in the list of keys given in the parameter
// Calling GetCompliments() with empty key slice will return all items in the flowSets
func (s *FlowSets) GetCompliments(keys []string) map[string]*FlowCreateRecord {
	flowRecs := make(map[string]*FlowCreateRecord)
	for setKey := range s.data {
		found := false
		for _, k := range keys {
			if setKey == k {
				found = true
				break
			}
		}
		if !found {
			flowRecs[setKey] = s.data[setKey]
		}
	}
	return flowRecs
}

// newFlowSets return an instacne of FlowSets
func NewFlowSets() *FlowSets {
	return &FlowSets{
		data: map[string]*FlowCreateRecord{},
	}
}
