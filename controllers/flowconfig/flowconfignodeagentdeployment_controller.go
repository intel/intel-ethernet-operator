// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package flowconfig

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	flowconfigv1 "github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1"
)

// FlowConfigNodeAgentDeploymentReconciler reconciles a FlowConfigNodeAgentDeployment object
type FlowConfigNodeAgentDeploymentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	oldDCFVfPoolName  corev1.ResourceName
	oldNADAnnotation  string
	flowConfigPod     *corev1.Pod
	uftContainerIndex int
}

const (
	networkAnnotation = "k8s.v1.cni.cncf.io/networks"
	nodeLabel         = "kubernetes.io/hostname"
	uftContainerName  = "uft"
	podTemplateFile   = "../../assets/flowconfig-daemon/daemon.yaml"
)

//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=flowconfignodeagentdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=flowconfignodeagentdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=flowconfignodeagentdeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=flowconfig.intel.com,resources=nodes,verbs=get;list;watch;update;
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;watch;list;create;delete

func (r *FlowConfigNodeAgentDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("flowconfignodeagentdeployment", req.NamespacedName)
	reqLogger.Info("Reconciling flowconfignodeagentdeployment")

	instance := &flowconfigv1.FlowConfigNodeAgentDeployment{}
	err := r.Client.Get(context.Background(), req.NamespacedName, instance)

	if err != nil {
		reqLogger.Info("failed to get FlowConfigNodeAgentDeployment instance, will try to get one after 30 seconds")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	if instance.Spec.NADAnnotation == "" {
		reqLogger.Info("NADAnnotation is not defined, will try to get one after 30 seconds")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	vfPoolName := corev1.ResourceName(instance.Spec.DCFVfPoolName)

	nodes := &corev1.NodeList{}
	err = r.List(context.Background(), nodes)

	if err != nil {
		reqLogger.Info("failed to get nodes %v", err)
		return ctrl.Result{}, err
	}

	var wasDeleted bool
	for _, node := range nodes.Items {
		// get pod object for selected node
		pod := &corev1.Pod{}
		err = r.Client.Get(context.TODO(), client.ObjectKey{
			Namespace: instance.Namespace,
			Name:      fmt.Sprintf("flowconfig-daemon-%s", node.Name),
		}, pod)

		if err != nil {
			// if POD does not exists on NODE - create it
			if errors.IsNotFound(err) {
				err = r.CreatePod(r.flowConfigPod, instance, node, vfPoolName, r.uftContainerIndex)
				if err != nil {
					reqLogger.Info("Failed to create POD on node with error", node.Name, err)
				}
			} else {
				reqLogger.Info("Error getting pod instance on node %s with error %v", node.Name, err)
			}
		} else {
			// POD exists, verify if has still the same resources or update is needed
			if r.oldDCFVfPoolName != vfPoolName || r.oldNADAnnotation != instance.Spec.NADAnnotation { // pool name of NAD annotation has been changed
				if _, exists := pod.Spec.Containers[r.uftContainerIndex].Resources.Limits[r.oldDCFVfPoolName]; exists {
					// delete POD, and let the next reconciliation iteration do the creation job
					err = r.Client.Delete(context.TODO(), pod)
					if err != nil {
						reqLogger.Info("Failed to delete POD %s with error %v", pod.Name, err)
					}
					wasDeleted = true
				}
			}
		}
	}

	// update variable at the end when all nodes are verified, to have correct value in next reconciliation loop
	r.oldDCFVfPoolName = vfPoolName
	r.oldNADAnnotation = instance.Spec.NADAnnotation

	if wasDeleted {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

func (r *FlowConfigNodeAgentDeploymentReconciler) CreatePod(templatePod *corev1.Pod, instance *flowconfigv1.FlowConfigNodeAgentDeployment, node corev1.Node, vfPoolName corev1.ResourceName, uftContainerIndex int) error {
	podLogger := r.Log.WithName("flowconfignodeagentdeployment")

	numResources := r.getNodeResources(node, vfPoolName.String())
	if numResources == 0 {
		podLogger.Info("No resources present on node")
		return nil
	}

	pod := &corev1.Pod{}
	pod.Spec = templatePod.Spec
	podName := "flowconfig-daemon-"

	pod.ObjectMeta.Name = podName

	if pod.Spec.NodeSelector == nil {
		pod.Spec.NodeSelector = make(map[string]string)
	}

	pod.ObjectMeta.Namespace = instance.Namespace
	pod.Spec.NodeSelector[nodeLabel] = node.Name
	pod.Name += node.Name

	uftContainer := templatePod.Spec.Containers[uftContainerIndex]

	annotation := r.addAnnotations(numResources, instance)

	podAnnotations := pod.ObjectMeta.Annotations

	if podAnnotations == nil {
		podAnnotations = make(map[string]string)
	}

	podAnnotations[networkAnnotation] = annotation

	pod.ObjectMeta.Annotations = podAnnotations
	uftContainer = r.addResources(uftContainer, vfPoolName, numResources)
	pod.Spec.Containers[uftContainerIndex] = uftContainer

	if err := controllerutil.SetControllerReference(instance, pod, r.Scheme); err != nil {
		podLogger.Info("Failed to set controller reference")
		return err
	}

	err := r.Client.Create(context.TODO(), pod)
	if err != nil {
		podLogger.Info("Failed to create pod")
		return err
	}

	return nil
}

func (r *FlowConfigNodeAgentDeploymentReconciler) addAnnotations(numResources int64, instance *flowconfigv1.FlowConfigNodeAgentDeployment) string {

	var annotation string
	for i := int64(1); i <= numResources; i++ {
		annotation += instance.Spec.NADAnnotation

		if i != numResources {
			annotation += ", "
		}
	}

	return annotation
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
	resLogger := r.Log.WithName("flowconfignodeagentdeployment")
	quantity, ok := node.Status.Capacity[corev1.ResourceName(vfPoolName)]

	if !ok {
		resLogger.Info("Error getting number of resources on node")
	}

	numResources, ok := quantity.AsInt64()

	if !ok {
		resLogger.Info("Error parsing quantity to int64")
	}

	return numResources
}

func (r *FlowConfigNodeAgentDeploymentReconciler) getPodTemplate() (*corev1.Pod, error) {
	filename, _ := filepath.Abs(podTemplateFile)
	spec, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading %s file: %v", podTemplateFile, err)
	}

	pod := &corev1.Pod{}
	err = yaml.Unmarshal(spec, &pod)

	var uftPresent bool
	for container := range pod.Spec.Containers {
		if pod.Spec.Containers[container].Name == uftContainerName {
			r.uftContainerIndex = container
			uftPresent = true
		}
	}

	if !uftPresent {
		return nil, fmt.Errorf("uft container not found in podSpec, pod definition is invalid")
	}

	return pod, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlowConfigNodeAgentDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	if r.flowConfigPod, err = r.getPodTemplate(); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&flowconfigv1.FlowConfigNodeAgentDeployment{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
