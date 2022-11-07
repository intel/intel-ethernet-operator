// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package v1

import (
	"os"
	"strconv"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("ClusterFlowConfig Webhook tests", func() {
	var (
		validClusterFlowRulesWithoutAction = []*ClusterFlowRule{
			{
				Pattern: []*FlowItem{
					{
						Type: "RTE_FLOW_ITEM_TYPE_IPV6",
						Spec: &runtime.RawExtension{
							Raw: []byte(`{ "hdr": { "vtc_flow": 12 } }`),
						},
					},
					{
						Type: "RTE_FLOW_ITEM_TYPE_END",
					},
				},
				Attr: &FlowAttr{
					Ingress: 1,
				},
			},
		}

		validClusterFlowRulesAction = []*ClusterFlowRule{
			{
				Action: []*ClusterFlowAction{
					{
						Type: ToPodInterface,
						Conf: &runtime.RawExtension{
							Raw: []byte(`{ "podInterface": "net0" }`),
						},
					},
				},
				Attr: &FlowAttr{
					Ingress: 1,
				},
			},
		}

		invalidClusterFlowRulesAction = []*ClusterFlowRule{
			{
				Action: []*ClusterFlowAction{
					{
						Type: ToPodInterface,
						Conf: &runtime.RawExtension{
							Raw: []byte(`{ "some": "net0" }`),
						},
					},
				},
				Attr: &FlowAttr{
					Ingress: 1,
				},
			},
		}

		invalidClusterFlowRulesActionParse = []*ClusterFlowRule{
			{
				Action: []*ClusterFlowAction{
					{
						Type: ToPodInterface,
						Conf: &runtime.RawExtension{
							Raw: []byte(`{ "some": net0" }`),
						},
					},
				},
				Attr: &FlowAttr{
					Ingress: 1,
				},
			},
		}

		invalidClusterFlowRulesActionNilConf = []*ClusterFlowRule{
			{
				Action: []*ClusterFlowAction{
					{
						Type: ToPodInterface,
					},
				},
				Attr: &FlowAttr{
					Ingress: 1,
				},
			},
		}

		invalidClusterFlowRulesActionType = []*ClusterFlowRule{
			{
				Action: []*ClusterFlowAction{
					{
						Type: 3435,
						Conf: &runtime.RawExtension{
							Raw: []byte(`{ "podInterface": "net0" }`),
						},
					},
				},
				Attr: &FlowAttr{
					Ingress: 1,
				},
			},
		}
	)

	getClusterFlowConfig := func(configurers ...func(flowConfig *ClusterFlowConfig)) *ClusterFlowConfig {
		obj := &ClusterFlowConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "sriov.intel.com/v1",
				Kind:       "ClusterFlowConfig",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "",
			},
			Spec: ClusterFlowConfigSpec{},
		}

		for _, config := range configurers {
			config(obj)
		}

		return obj
	}

	Context("verify ValidateCreate()", func() {
		DescribeTable("verify ValidateCreate()", func(clusterFlowConfigObject *ClusterFlowConfig, expectError bool, message string) {
			err := clusterFlowConfigObject.ValidateCreate()
			if expectError {
				Expect(err).ShouldNot(BeNil())
				Expect(err.Error()).Should(ContainSubstring(message))
			} else {
				Expect(err).Should(BeNil())
			}
		},
			Entry("empty CR", getClusterFlowConfig(), true, "PodSelector is undefined, please add it."),
			Entry("empty rules, valid PodSelector", getClusterFlowConfig(func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.PodSelector = &metav1.LabelSelector{}
			}), false, ""),
			Entry("valid rules, missing action, valid PodSelector", getClusterFlowConfig(func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.PodSelector = &metav1.LabelSelector{}
			}, func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.Rules = validClusterFlowRulesWithoutAction
			}), false, ""),
			Entry("missing rules, valid action, valid PodSelector", getClusterFlowConfig(func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.PodSelector = &metav1.LabelSelector{}
			}, func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.Rules = validClusterFlowRulesAction
			}), false, ""),
			Entry("missing rules, invalid action - invalid interface, valid PodSelector", getClusterFlowConfig(func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.PodSelector = &metav1.LabelSelector{}
			}, func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.Rules = invalidClusterFlowRulesAction
			}), true, "network interface name cannot be empty action to-pod-interface"),
			Entry("missing rules, invalid action - unmarshal error, valid PodSelector", getClusterFlowConfig(func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.PodSelector = &metav1.LabelSelector{}
			}, func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.Rules = invalidClusterFlowRulesActionParse
			}), true, "unable to unmarshal action"),
			Entry("missing rules, invalid action - nil config, valid PodSelector", getClusterFlowConfig(func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.PodSelector = &metav1.LabelSelector{}
			}, func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.Rules = invalidClusterFlowRulesActionNilConf
			}), true, "have empty configuration"),
			Entry("missing rules, invalid action - type, valid PodSelector", getClusterFlowConfig(func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.PodSelector = &metav1.LabelSelector{}
			}, func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.Rules = invalidClusterFlowRulesActionType
			}), true, "invalid action type"),
		)
	})

	Context("verify ValidateUpdate()", func() {
		DescribeTable("verify ValidateUpdate()", func(clusterFlowConfigObject *ClusterFlowConfig, expectError bool, message string) {
			oldObj := &runtime.Unknown{}
			err := clusterFlowConfigObject.ValidateUpdate(oldObj)
			if expectError {
				Expect(err).ShouldNot(BeNil())
				Expect(err.Error()).Should(ContainSubstring(message))
			} else {
				Expect(err).Should(BeNil())
			}
		},
			Entry("empty CR", getClusterFlowConfig(), true, "PodSelector is undefined, please add it."),
			Entry("empty rules, valid PodSelector", getClusterFlowConfig(func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.PodSelector = &metav1.LabelSelector{}
			}), false, ""),
			Entry("valid rules, missing action, valid PodSelector", getClusterFlowConfig(func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.PodSelector = &metav1.LabelSelector{}
			}, func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.Rules = validClusterFlowRulesWithoutAction
			}), false, ""),
			Entry("missing rules, valid action, valid PodSelector", getClusterFlowConfig(func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.PodSelector = &metav1.LabelSelector{}
			}, func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.Rules = validClusterFlowRulesAction
			}), false, ""),
			Entry("missing rules, invalid action, valid PodSelector", getClusterFlowConfig(func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.PodSelector = &metav1.LabelSelector{}
			}, func(flowConfig *ClusterFlowConfig) {
				flowConfig.Spec.Rules = invalidClusterFlowRulesAction
			}), true, "network interface name cannot be empty action to-pod-interface"),
		)
	})

	Context("validate delete", func() {
		It("empty CR", func() {
			obj := getClusterFlowConfig()
			Expect(obj.ValidateDelete()).To(BeNil())
		})
	})
})

// TestClusterValidateCreateFuzz runs fuzz test on ValidateCreate() method by fuzzing ClusterFlowConfig API object.
// Set FUZZITER env variable to define how many times the test will fuzz. Default count is 10.
func TestClusterValidateCreateFuzz(t *testing.T) {
	fuzzIterationStr := os.Getenv("FUZZITER")
	var fuzzIteration int
	fuzzIteration, err := strconv.Atoi(fuzzIterationStr)

	if err != nil || fuzzIteration <= 0 {
		t.Logf("the FUZZITER env variable is not set or contains invalid value: %+s\n", fuzzIterationStr)
		t.Logf("skipping TestClusterValidateCreateFuzz\n")
		t.Skip()
	}

	clusterFlowConfig := &ClusterFlowConfig{}
	f := fuzz.New().NilChance(0).Funcs(
		func(e *runtime.RawExtension, c fuzz.Continue) {
			b := []byte{}
			c.Fuzz(&b)
			e.Object = nil
			e.Raw = b

		},
		func(e *ClusterFlowRule, c fuzz.Continue) {
			act := []ClusterFlowAction{}
			c.Fuzz(&act)

			it := []FlowItem{}
			c.Fuzz(&it)

			atr := FlowAttr{}
			c.Fuzz(&atr)

			e.Pattern = itemPtr(it)
			e.Action = clusterActionPtr(act)
			e.Attr = &atr
		},
		func(e *ClusterFlowConfigSpec, c fuzz.Continue) {
			x := []ClusterFlowRule{}
			c.Fuzz(&x)
			e.Rules = clusterRulePtr(x)
		},
	)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("the code resulted in panic. flowconfig:\n %+v\n", clusterFlowConfig)
		}
	}()

	t.Logf("fuzzing ClusterFlowConfig for: %d\n", fuzzIteration)
	for i := 0; i < fuzzIteration; i++ {
		f.Fuzz(&clusterFlowConfig)
		_ = clusterFlowConfig.ValidateCreate()
	}
}

func clusterRulePtr(cfr []ClusterFlowRule) []*ClusterFlowRule {
	out := []*ClusterFlowRule{}
	for _, r := range cfr {
		out = append(out, &r)
	}
	return out
}

func clusterActionPtr(x []ClusterFlowAction) []*ClusterFlowAction {
	out := []*ClusterFlowAction{}
	for _, xx := range x {
		out = append(out, &xx)
	}
	return out
}
