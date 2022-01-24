// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package main

import (
	"context"
	"flag"
	"os"
	"time"

	fwddp_manager "github.com/otcshare/intel-ethernet-operator/pkg/fwddp-manager"
	"github.com/otcshare/intel-ethernet-operator/pkg/utils/assets"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"
	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"
	flowconfigcontrollers "github.com/otcshare/intel-ethernet-operator/controllers/flowconfig"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(ethernetv1.AddToScheme(scheme))
	utilruntime.Must(flowconfigv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	restConfig := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "8ee6d2ed.intel.com",
		Namespace:              fwddp_manager.NAMESPACE,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&fwddp_manager.EthernetClusterConfigReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ethernet").WithName("EthernetClusterConfig"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EthernetClusterConfig")
		os.Exit(1)
	}

	// to disable webhook(e.g. when testing locally) run it as 'make run ENABLE_WEBHOOKS=false'
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&flowconfigv1.NodeFlowConfig{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NodeFlowConfig")
			os.Exit(1)
		}

		if err = (&flowconfigv1.ClusterFlowConfig{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ClusterFlowConfig")
			os.Exit(1)
		}
	}

	if err = (&flowconfigcontrollers.FlowConfigNodeAgentDeploymentReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("flowconfig").WithName("FlowConfigNodeAgentDeployment"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FlowConfigNodeAgentDeployment")
		os.Exit(1)
	}

	if err = (&flowconfigcontrollers.ClusterFlowConfigReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("flowconfig").WithName("ClusterFlowConfig"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterFlowConfig")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	var adHocClient client.Client
	if adHocClient, err = client.New(restConfig, client.Options{Scheme: scheme}); err != nil {
		setupLog.Error(err, "failed to create client")
		os.Exit(1)
	}

	owner := new(appsv1.Deployment)
	if err := adHocClient.Get(
		context.Background(),
		client.ObjectKey{Name: "intel-ethernet-operator-controller-manager", Namespace: fwddp_manager.NAMESPACE},
		owner,
	); err != nil {
		setupLog.Error(err, "unable to get operator deployment")
		os.Exit(1)
	}

	if err := (&assets.Manager{
		Client:    adHocClient,
		Log:       ctrl.Log.WithName("manager"),
		EnvPrefix: "ETHERNET_",
		Scheme:    scheme,
		Owner:     owner,
		Assets: []assets.Asset{
			{Path: "assets/100-labeler.yaml"},
			{Path: "assets/200-daemon.yaml", BlockingReadiness: assets.ReadinessPollConfig{Retries: 30, Delay: 20 * time.Second}},
			{Path: "assets/300-machine-config.yaml"},
		},
	}).LoadAndDeploy(context.Background()); err != nil {
		setupLog.Error(err, "failed to deploy the assets")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
