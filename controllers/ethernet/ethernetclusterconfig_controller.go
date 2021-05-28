// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package ethernet

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ethernetv1 "github.com/otcshare/intel-ethernet-operator/apis/ethernet/v1"
)

// EthernetClusterConfigReconciler reconciles a EthernetClusterConfig object
type EthernetClusterConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ethernet.intel.com,resources=ethernetclusterconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ethernet.intel.com,resources=ethernetclusterconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ethernet.intel.com,resources=ethernetclusterconfigs/finalizers,verbs=update
func (r *EthernetClusterConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("ethernetclusterconfig", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EthernetClusterConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ethernetv1.EthernetClusterConfig{}).
		Complete(r)
}
