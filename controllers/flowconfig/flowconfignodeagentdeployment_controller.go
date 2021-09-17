// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flowconfig

import (
	"context"
	"log"
	"strconv"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"
)

// FlowConfigNodeAgentDeploymentReconciler reconciles a FlowConfigNodeAgentDeployment object
type FlowConfigNodeAgentDeploymentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const networkAnnotation = "k8s.v1.cni.cncf.io/networks"

//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=flowconfignodeagentdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=flowconfignodeagentdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=flowconfignodeagentdeployments/finalizers,verbs=update

//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=nodes,verbs=get;list;watch;update;
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=nodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=nodes/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;watch;list;create;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;watch;list;delete
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=nodeflowconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=nodeflowconfigs/status,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FlowConfigNodeAgentDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *FlowConfigNodeAgentDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("flowconfignodeagentdeployment", req.NamespacedName)
	reqLogger.Info("Reconciling flowconfignodeagentdeployment")

	instance := &flowconfigv1.FlowConfigNodeAgentDeployment{}
	err := r.Client.Get(context.Background(), req.NamespacedName, instance)

	if err != nil {
		reqLogger.Info("failed to get FlowConfigNodeAgentDeployment instance")
		return ctrl.Result{}, err
	}

	var uftContainerIndex int
	var uftPresent bool
	for container := range instance.Spec.Template.Spec.Containers {
		if instance.Spec.Template.Spec.Containers[container].Name == "uft" {
			uftContainerIndex = container
			uftPresent = true
		}
	}

	if !uftPresent {
		reqLogger.Info("ERROR: uft container not found in podSpec, no pods will be created")
		return ctrl.Result{}, nil
	}

	vfPoolName := corev1.ResourceName(instance.Spec.DCFVfPoolName)

	nodes := &corev1.NodeList{}
	err = r.List(context.Background(), nodes)

	if err != nil {
		reqLogger.Info("failed to get nodes")
		return ctrl.Result{}, err
	}

	for _, node := range nodes.Items {
		pod := &corev1.Pod{}
		pod.Spec = instance.Spec.Template.Spec
		podName := "flowconfig-daemon-"

		pod.ObjectMeta.Name = podName
		pod.ObjectMeta.Namespace = instance.Namespace
		pod.Spec.NodeName = node.Name
		pod.Name += node.Name
		numResources := r.getNodeResources(node, vfPoolName.String())

		uftContainer := instance.Spec.Template.Spec.Containers[uftContainerIndex]

		var annotation string

		for i := int64(1); i <= numResources; i++ {
			annotation += instance.Spec.NADAnnotation

			if i != numResources {
				annotation += ", "
			}
		}

		podAnnotations := pod.ObjectMeta.Annotations

		if podAnnotations == nil {
			podAnnotations = make(map[string]string)
		}

		podAnnotations[networkAnnotation] = annotation

		pod.ObjectMeta.Annotations = podAnnotations
		uftContainer = r.addResources(uftContainer, vfPoolName, numResources)
		pod.Spec.Containers[uftContainerIndex] = uftContainer

		err := r.Client.Create(context.TODO(), pod)
		if err != nil {
			log.Printf("%v", err)
		}
	}
	return ctrl.Result{}, nil
}

func (r *FlowConfigNodeAgentDeploymentReconciler) addResources(container corev1.Container, vfPoolName corev1.ResourceName, numResources int64) corev1.Container {
	if container.Resources.Limits == nil {
		limits := corev1.ResourceList{}
		container.Resources.Limits = limits
	}

	if container.Resources.Requests == nil {
		requests := corev1.ResourceList{}
		container.Resources.Requests = requests
	}

	container.Resources.Limits[vfPoolName] = *resource.NewQuantity(numResources, resource.DecimalSI)
	container.Resources.Requests[vfPoolName] = *resource.NewQuantity(numResources, resource.DecimalSI)

	return container
}

func (r *FlowConfigNodeAgentDeploymentReconciler) getNodeResources(node corev1.Node, vfPoolName string) int64 {
	var count string
	for k, v := range node.Status.Capacity {
		resource := k.String()
		if resource == vfPoolName {
			count = v.String()
		}
	}

	numResource, err := strconv.ParseInt(count, 10, 64)

	if err != nil {
		log.Printf("%v", err)
	}

	return numResource
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlowConfigNodeAgentDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&flowconfigv1.FlowConfigNodeAgentDeployment{}).
		Complete(r)
}
