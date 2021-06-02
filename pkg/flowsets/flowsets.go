/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package flowsets

import (
	flowapi "github.com/otcshare/intel-ethernet-operator/pkg/rpc/v1/flow"
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
	if _, ok := s.data[key]; ok {
		delete(s.data, key)
	}
}

func (s *FlowSets) Has(key string) bool {
	_, ok := s.data[key]
	return ok
}

// This method probably is not needed!
// func (s *FlowSets) GetFlowID(key string) (uint32, error) {
// 	rec, ok := s.data[key]
// 	if !ok {
// 		return 0, fmt.Errorf("FlowCreateRecord is not found")
// 	}
// 	return rec.flowID, nil
// }

func (s *FlowSets) Size() int {
	return len(s.data)
}

// GetCompliments returns a list of flowRecords with keys that are not in the list of keys given in the parameter
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
