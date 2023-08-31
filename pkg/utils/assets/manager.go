// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2023 Intel Corporation

package assets

import (
	"context"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var proxies = []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"}

// Manager loads & deploys assets specified in the Asset field
type Manager struct {
	Client    client.Client
	Namespace string

	Log    logr.Logger
	Assets []Asset

	// Prefix used to gather enviroment variables for the templating the assets
	EnvPrefix string

	// Can be removed after sigs.k8s.io/controller-runtime v0.7.0 release where client.Scheme() is available
	Scheme *runtime.Scheme

	Owner metav1.Object
}

// buildTemplateVars creates map with variables for templating.
// Template variables are env variables with specified prefix and additional information
// from cluster such as kernel
func (m *Manager) buildTemplateVars() (map[string]string, error) {
	tp := make(map[string]string)

	for _, pair := range os.Environ() {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 && (strings.HasPrefix(kv[0], m.EnvPrefix) || isProxy(kv[0])) {
			tp[kv[0]] = kv[1]
		}
	}

	for _, proxy := range proxies {
		if _, exists := tp[proxy]; !exists {
			tp[proxy] = ""
		}
	}
	
	fwPath, err := utils.GetFwSearchPath()
	if err != nil {
		m.Log.Error(err, "Error getting customized fw search path, using default", "path", fwPath)
	}
	tp["FW_HOST_PATH"] = fwPath

	return tp, nil
}

func isProxy(name string) bool {
	//ignores lowercase proxy settings because our operator only sets/reads uppercase values
	// and OCP proxy/cluster object sets uppercase proxy

	for _, proxy := range proxies {
		if name == proxy {
			return true
		}
	}
	return false
}

// DeployConfigMaps issues an asset load from the path and then deployment
func (m *Manager) DeployConfigMaps(ctx context.Context) error {
	if err := m.LoadFromFile(); err != nil {
		return err
	}
	if err := m.Deploy(ctx); err != nil {
		return err
	}

	return nil
}

// LoadFromFile loads given asset from the path
func (m *Manager) LoadFromFile() error {
	tv, err := m.buildTemplateVars()
	if err != nil {
		m.Log.Error(err, "failed to build template vars")
		return err
	}

	m.Log.Info("template vars", "tv", tv)

	for idx := range m.Assets {
		m.Log.Info("loading asset", "path", m.Assets[idx].Path)

		m.Assets[idx].log = m.Log.WithName("asset")
		m.Assets[idx].substitutions = tv

		if err := m.Assets[idx].loadFromFile(); err != nil {
			m.Log.Error(err, "failed to loadFromFile asset", "path", m.Assets[idx].Path)
			return err
		}

		m.Log.Info("asset loaded successfully", "path", m.Assets[idx].Path, "num of objects", len(m.Assets[idx].objects))
	}

	return nil
}

// LoadFromConfigMapAndDeploy issues an asset load from the ConfigMap and then deployment
func (m *Manager) LoadFromConfigMapAndDeploy(ctx context.Context) error {
	if err := m.LoadFromConfigMap(ctx); err != nil {
		return err
	}
	if err := m.Deploy(ctx); err != nil {
		return err
	}

	return nil
}

// LoadFromConfigMap loads given asset from the ConfigMap
func (m *Manager) LoadFromConfigMap(ctx context.Context) error {
	for idx := range m.Assets {
		m.Log.Info("loading asset", "configMapName", m.Assets[idx].ConfigMapName)

		if err := m.Assets[idx].loadFromConfigMap(ctx, m.Client, m.Namespace); err != nil {
			m.Log.Error(err, "failed to loadFromConfigMap", "ConfigMap name", m.Assets[idx].ConfigMapName)
			return err
		}

		m.Log.Info("asset loaded successfully", "ConfigMap name", m.Assets[idx].ConfigMapName, "num of objects", len(m.Assets[idx].objects))
	}

	return nil
}

// Deploy will create (or update) selected asset
func (m *Manager) Deploy(ctx context.Context) error {
	for _, asset := range m.Assets {
		m.Log.Info("deploying asset",
			"path", asset.Path,
			"retries", asset.BlockingReadiness.Retries,
			"delay", asset.BlockingReadiness.Delay.String(),
			"num of objects", len(asset.objects))

		if err := asset.createOrUpdate(ctx, m.Client, m.Owner, m.Scheme); err != nil {
			m.Log.Error(err, "failed to create asset", "path", asset.Path)
			return err
		}

		m.Log.Info("asset created successfully", "path", asset.Path)

		if err := asset.waitUntilReady(ctx, m.Client); err != nil {
			m.Log.Error(err, "waitUntilReady")
			return err
		}
	}

	return nil
}
