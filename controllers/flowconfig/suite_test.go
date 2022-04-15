// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flowconfig

import (
	"fmt"
	configv1 "github.com/openshift/api/config/v1"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"
	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/flowsets"
	mock "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var (
	cfg                   *rest.Config
	k8sClient             client.Client
	testEnv               *envtest.Environment
	nodeName              = "testk8snode"
	mockDCF               *mock.FlowServiceClient
	nodeFlowConfigRc      *NodeFlowConfigReconciler
	nodeAgentDeploymentRc *FlowConfigNodeAgentDeploymentReconciler
	managerMutex          = sync.Mutex{}
)

func MatchQuantityObject(expected interface{}) types.GomegaMatcher {
	return &representQuantityMatcher{
		expected: expected,
	}
}

type representQuantityMatcher struct {
	expected interface{}
}

func (matcher *representQuantityMatcher) Match(actual interface{}) (success bool, err error) {
	currentQuantity, ok := actual.(resource.Quantity)
	if !ok {
		return false, fmt.Errorf("MatchQuantityObject matcher expects current as resource.Quantity")
	}

	expectedQuantity, ok := matcher.expected.(resource.Quantity)
	if !ok {
		return false, fmt.Errorf("MatchQuantityObject matcher expects expected as resource.Quantity")
	}

	return currentQuantity.Equal(expectedQuantity), nil
}

func (matcher *representQuantityMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto contain\n\t%#v", actual, matcher.expected)
}

func (matcher *representQuantityMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to contain\n\t%#v", actual, matcher.expected)
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = flowconfigv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = configv1.Install(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	r1 := rand.New(rand.NewSource(time.Now().UnixNano()))
	var metricsAddr = fmt.Sprintf(":%d", (r1.Intn(100) + 38080))

	managerMutex.Lock()

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: metricsAddr,
	})
	Expect(err).ToNot(HaveOccurred())

	// Set NodeAclReconciler
	fs := flowsets.NewFlowSets()
	mockDCF = &mock.FlowServiceClient{}

	nodeFlowConfigRc = GetNodeFlowConfigReconciler(
		k8sManager.GetClient(),
		ctrl.Log.WithName("controllers").WithName("NodeFlowConfig"),
		k8sManager.GetScheme(),
		fs,
		mockDCF,
		nodeName,
	)

	err = nodeFlowConfigRc.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	nodeAgentDeploymentRc = &FlowConfigNodeAgentDeploymentReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NodeAgentDeployment"),
		Scheme: k8sManager.GetScheme(),
	}

	err = nodeAgentDeploymentRc.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: k8sManager.GetScheme()})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	managerMutex.Unlock()

	// Start manager
	go func() {
		defer GinkgoRecover()

		managerMutex.Lock()
		defer managerMutex.Unlock()

		err := k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	_ = testEnv.Stop()

	targetDir, err := filepath.Abs(".")
	Expect(err).Should(BeNil())
	err = os.RemoveAll(targetDir + "/assets/")
	Expect(err).Should(BeNil())
})
