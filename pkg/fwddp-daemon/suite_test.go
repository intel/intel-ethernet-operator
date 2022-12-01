// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

package daemon

import (
	"os"
	"path/filepath"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	config        *rest.Config
	k8sClient     client.Client
	testEnv       *envtest.Environment
	testTmpFolder string
	log           = ctrl.Log.WithName("EthernetDaemon-test")
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	var err error

	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	config, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(config).ToNot(BeNil())

	err = ethernetv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = configv1.Install(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(config, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())

	err = os.RemoveAll(testTmpFolder)
	Expect(err).ShouldNot(HaveOccurred())
})
