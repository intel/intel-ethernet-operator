// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flowconfig

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeTypes "k8s.io/apimachinery/pkg/types"
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
	clusterFlowConfigRc   *ClusterFlowConfigReconciler
	managerMutex          = sync.Mutex{}
	nodePrototype         = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-dummy",
		},
	}
	defaultSysFs = "/sys"
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

func createNode(name string, configurers ...func(n *corev1.Node)) *corev1.Node {
	node := nodePrototype.DeepCopy()
	node.Name = name
	for _, configure := range configurers {
		configure(node)
	}

	Expect(k8sClient.Create(context.TODO(), node)).ToNot(HaveOccurred())

	return node
}

func deleteNode(node *corev1.Node) {
	err := k8sClient.Delete(context.Background(), node)

	Expect(err).Should(BeNil())
}

func createPod(name, ns string, configurers ...func(pod *corev1.Pod)) *corev1.Pod {
	var graceTime int64 = 0
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &graceTime,
			Containers: []corev1.Container{
				{
					Name:    "uft",
					Image:   "docker.io/alpine",
					Command: []string{"/bin/sh", "-c", "sleep INF"},
				},
			},
		},
	}

	for _, configure := range configurers {
		configure(pod)
	}

	return pod
}

// Deploys pod and sets its phase to the desired value.
// This function waits until the pod is created before updating it. The timeout and checking interval can be configured (in seconds).
func deployPodAndUpdatePhase(pod *corev1.Pod, podPhase corev1.PodPhase, checkTimeout time.Duration, checkInterval time.Duration) error {
	err := k8sClient.Create(context.TODO(), pod)
	if err != nil {
		return err
	}

	err = WaitForObjectCreation(k8sClient, pod.Name, pod.Namespace, checkTimeout*time.Second, checkInterval*time.Second, pod)
	if err != nil {
		return err
	}

	pod.Status.Phase = podPhase
	err = k8sClient.Status().Update(context.Background(), pod)
	if err != nil {
		return err
	}

	return nil
}

func deletePod(name, ns string) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}

	err := k8sClient.Delete(context.Background(), pod)
	Expect(err).Should(BeNil())
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
		defaultSysFs,
	)

	err = nodeFlowConfigRc.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// Point test to the correct "assets" directory
	podTemplateFile = "../../assets/flowconfig-daemon/daemon.yaml"

	nodeAgentDeploymentRc = &FlowConfigNodeAgentDeploymentReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NodeAgentDeployment"),
		Scheme: k8sManager.GetScheme(),
	}
	err = nodeAgentDeploymentRc.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	clusterFlowConfigRc = &ClusterFlowConfigReconciler{
		Client:                   k8sManager.GetClient(),
		Log:                      ctrl.Log.WithName("controllers").WithName("ClusterFlowConfig"),
		Scheme:                   k8sManager.GetScheme(),
		Cluster2NodeRulesHashMap: make(map[kubeTypes.NamespacedName]map[kubeTypes.NamespacedName][]string),
	}
	err = clusterFlowConfigRc.SetupWithManager(k8sManager)
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
