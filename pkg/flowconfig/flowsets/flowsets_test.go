// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2023 Intel Corporation

package flowsets

import (
	"testing"

	flowapi "github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFlowSets(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FlowSets Suite")
}

var _ = Describe("FlowSets", func() {
	var (
		flowRecs *FlowSets
	)

	BeforeEach(func() {

		// Initialize flowRecs with 2 entries
		flowRecs = NewFlowSets()
		flowRecs.Add("aa", 1, &flowapi.RequestFlowCreate{})
		flowRecs.Add("bb", 2, &flowapi.RequestFlowCreate{})
	})

	Describe("Get FlowSets size", func() {
		Context("Get initial size", func() {
			It("should have 2 items in it", func() {
				Expect(flowRecs.Size()).To(Equal(2))
			})
		})
		Context("After removing item(s)", func() {
			It("should have 1 item in it", func() {
				flowRecs.Delete("aa")
				Expect(flowRecs.Size()).To(Equal(1))
			})
			It("should have 0 item after deleting all", func() {
				flowRecs.Delete("aa")
				flowRecs.Delete("bb")
				Expect(flowRecs.Size()).To(Equal(0))
			})
		})
		Context("After removing item that does not exist", func() {
			It("should have still have initial 2 items in it", func() {
				flowRecs.Delete("cc")
				Expect(flowRecs.Size()).To(Equal(2))
			})
		})
	})

	Describe("Look up items", func() {
		Context("Look up item expected to be found", func() {
			It("should return true", func() {
				Expect(flowRecs.Has("aa")).To(Equal(true))
			})
		})
		Context("Look up item expected not to be found", func() {
			It("should return false", func() {
				Expect(flowRecs.Has("cc")).To(Equal(false))
			})
		})
	})

	Describe("Get compliments set", func() {
		Context("look up with empty string slice", func() {
			It("should return all items", func() {
				lookUpKeys := []string{}
				compliments := flowRecs.GetCompliments(lookUpKeys)
				Expect(len(compliments)).To(Equal(2))
			})
		})
		Context("look up with one matched key", func() {
			It("should return one other unmatched item", func() {
				lookUpKeys := []string{"aa", "cc"}
				compliments := flowRecs.GetCompliments(lookUpKeys)
				Expect(len(compliments)).To(Equal(1))
				item, ok := compliments["aa"]
				Expect(ok).To(Equal(false))
				Expect(item).Should(BeNil())
				// it should contain other item with key "bb"
				item, ok = compliments["bb"]
				Expect(ok).To(Equal(true))
				Expect(item).ShouldNot(BeNil())
			})
		})
		Context("look up with all matched keys", func() {
			It("should return one unmatched item", func() {
				lookUpKeys := []string{"aa", "bb"}
				compliments := flowRecs.GetCompliments(lookUpKeys)
				Expect(len(compliments)).To(Equal(0))
				item, ok := compliments["aa"]
				Expect(ok).To(Equal(false))
				Expect(item).Should(BeNil())
			})
		})
		Context("look up with none matched keys", func() {
			It("should return 2 unmatched items", func() {
				lookUpKeys := []string{"cc", "dd"}
				compliments := flowRecs.GetCompliments(lookUpKeys)
				Expect(len(compliments)).To(Equal(2))
				item, ok := compliments["aa"]
				Expect(ok).To(Equal(true))
				Expect(item).ShouldNot(BeNil())

				item, ok = compliments["bb"]
				Expect(ok).To(Equal(true))
				Expect(item).ShouldNot(BeNil())
			})
		})
	})
})
