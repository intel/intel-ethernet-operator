// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flowconfig

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"
)

// ClusterFlowConfigReconciler reconciles a ClusterFlowConfig object
type ClusterFlowConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=clusterflowconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=clusterflowconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=clusterflowconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterFlowConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ClusterFlowConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("clusterflowconfig", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterFlowConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&flowconfigv1.ClusterFlowConfig{}).
		Complete(r)
}
