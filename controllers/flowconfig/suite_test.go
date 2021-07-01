// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flowconfig

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/flowsets"
	mock "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow/mocks"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var (
	cfg              *rest.Config
	k8sClient        client.Client
	testEnv          *envtest.Environment
	nodeName         = "testk8snode"
	mockDCF          *mock.FlowServiceClient
	nodeFlowConfigRc *NodeFlowConfigReconciler
	metricsAddr      = ":38080"
)

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

	// +kubebuilder:scaffold:scheme

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

	// Start manager
	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	// Set k8sClient from Manager
	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
