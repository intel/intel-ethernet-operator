// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package assets

import (
	"bytes"
	"context"
	"errors"
	"github.com/go-logr/logr"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/apimachinery/pkg/api/equality"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const _BUFFER_SIZE = 4096

// ReadinessPollConfig stores config for waiting block
// Use when deployment of an asset should wait until the asset is ready
type ReadinessPollConfig struct {
	// How many times readiness should be checked before returning error
	Retries int

	// Delay between retries
	Delay time.Duration
}

// Asset represents a set of Kubernetes objects to be deployed.
type Asset struct {
	// Path contains a filepath to the asset
	Path string

	// BlockingReadiness stores polling configuration.
	BlockingReadiness ReadinessPollConfig

	substitutions map[string]string

	objects []client.Object

	log logr.Logger
}

func (a *Asset) load() error {
	cleanPath := filepath.Clean(a.Path)

	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		return err
	}

	if fileInfo.Mode().IsDir() {
		return errors.New("loading directory of assets is not supported")
	}

	content, err := ioutil.ReadFile(cleanPath)
	if err != nil {
		return err
	}

	t, err := template.New("asset").Option("missingkey=error").Parse(string(content))
	if err != nil {
		return err
	}

	var templatedContent bytes.Buffer
	if err = t.Execute(&templatedContent, a.substitutions); err != nil {
		return err
	}

	objectsDefs := regexp.MustCompile("\n-{3}").Split(templatedContent.String(), -1)
	for _, objectDef := range objectsDefs {
		r := strings.NewReader(objectDef)
		decoder := yaml.NewYAMLOrJSONDecoder(r, _BUFFER_SIZE)
		obj := new(unstructured.Unstructured)
		if err := decoder.Decode(obj); err != nil {
			return err
		}
		a.objects = append(a.objects, obj)
	}
	return nil
}

func (a *Asset) setOwner(owner metav1.Object, obj runtime.Object, s *runtime.Scheme) error {
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return errors.New(obj.GetObjectKind().GroupVersionKind().String() + " is not metav1.Object")
	}

	if owner.GetNamespace() == metaObj.GetNamespace() {
		a.log.Info(
			"set owner for object",
			"owner", owner.GetName()+"."+owner.GetNamespace(),
			"object", metaObj.GetName()+"."+metaObj.GetNamespace())
		if err := controllerutil.SetControllerReference(owner, metaObj, s); err != nil {
			return err
		}
	} else {
		a.log.Info("Unsupported owner for object...skipping",
			"owner", owner.GetName()+"."+owner.GetNamespace(),
			"object", metaObj.GetName()+"."+metaObj.GetNamespace())
	}
	return nil
}

func (a *Asset) createOrUpdate(ctx context.Context, c client.Client, o metav1.Object, s *runtime.Scheme) error {
	for _, obj := range a.objects {
		a.log.Info("create or update", "asset", a.Path, "kind", obj.GetObjectKind())

		err := a.setOwner(o, obj, s)
		if err != nil {
			return err
		}

		gvk := obj.GetObjectKind().GroupVersionKind()
		old := &unstructured.Unstructured{}
		old.SetGroupVersionKind(gvk)
		key := client.ObjectKeyFromObject(obj)
		if err := c.Get(ctx, key, old); err != nil {
			if !apierr.IsNotFound(err) {
				a.log.Error(err, "Failed to get an object", "key", key, "GroupVersionKind", gvk)
				return err
			}

			// Object does not exist
			if err := c.Create(ctx, obj); err != nil {
				a.log.Error(err, "create failed", "key", key, "GroupVersionKind", gvk)
				return err
			}
			a.log.Info("Object created", "key", key, "GroupVersionKind", gvk)
			continue
		}

		if strings.ToLower(old.GetObjectKind().GroupVersionKind().Kind) == "configmap" {
			isImmutable, ok := old.Object["immutable"].(bool)
			if !ok {
				a.log.Info("Failed to read 'immutable' field", "key", key, "GroupVersionKind", gvk)
			} else {
				if isImmutable {
					a.log.Info("Skipping update because it is marked as immutable", "key", key, "GroupVersionKind", gvk)
					continue
				}
			}
		}

		if !equality.Semantic.DeepDerivative(obj, old) {
			obj.SetResourceVersion(old.GetResourceVersion())
			if err := c.Update(ctx, obj); err != nil {
				a.log.Error(err, "Update failed", "key", key, "GroupVersionKind", gvk)
				return err
			}
			a.log.Info("Object updated", "key", key, "GroupVersionKind", gvk)
		} else {
			a.log.Info("Object has not changed", "key", key, "GroupVersionKind", gvk)
		}

	}

	return nil
}

func (a *Asset) waitUntilReady(ctx context.Context, apiReader client.Reader) error {
	if a.BlockingReadiness.Retries == 0 {
		return nil
	}

	for _, obj := range a.objects {
		if obj.GetObjectKind().GroupVersionKind().Kind != "DaemonSet" {
			continue
		}
		a.log.Info("waiting until daemonset is ready", "asset", a.Path)

		backoff := wait.Backoff{
			Steps:    a.BlockingReadiness.Retries,
			Duration: a.BlockingReadiness.Delay,
			Factor:   1,
		}
		f := func() (bool, error) {
			objKey := client.ObjectKeyFromObject(obj)
			ds := &appsv1.DaemonSet{}
			err := apiReader.Get(ctx, objKey, ds)
			if err != nil {
				return false, err
			}

			a.log.Info("daemonset status",
				"name", ds.GetName(),
				"NumberUnavailable", ds.Status.NumberUnavailable,
				"DesiredNumberScheduled", ds.Status.DesiredNumberScheduled)

			return ds.Status.NumberUnavailable == 0, nil
		}

		if err := wait.ExponentialBackoff(backoff, f); err != nil {
			a.log.Error(err, "wait for daemonset failed")
			return err
		}
	}

	return nil
}
