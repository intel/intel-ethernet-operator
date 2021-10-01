package flow

import (
	"reflect"
	"testing"
)

func TestGetFlowItemObj(t *testing.T) {

	fItemTcpHeader := GetFlowItemObj(RteFlowItemType(RteFlowItemType_value["RTE_FLOW_ITEM_TYPE_TCP"])).(*RteFlowItemTcp)

	if hdrField := reflect.ValueOf(*fItemTcpHeader).FieldByName("Hdr"); !hdrField.IsValid() {
		t.Fail()
	}
}
