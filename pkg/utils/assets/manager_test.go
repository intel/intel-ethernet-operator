// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

package assets

import (
	"context"
	gerrors "errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// runtime.Object implementation
type InvalidRuntimeType struct {
}

func (*InvalidRuntimeType) GetNamespace() string                                   { return "" }
func (*InvalidRuntimeType) SetNamespace(namespace string)                          {}
func (*InvalidRuntimeType) GetName() string                                        { return "" }
func (*InvalidRuntimeType) SetName(name string)                                    {}
func (*InvalidRuntimeType) GetGenerateName() string                                { return "" }
func (*InvalidRuntimeType) SetGenerateName(name string)                            {}
func (*InvalidRuntimeType) GetUID() types.UID                                      { return "" }
func (*InvalidRuntimeType) SetUID(uid types.UID)                                   {}
func (*InvalidRuntimeType) GetResourceVersion() string                             { return "" }
func (*InvalidRuntimeType) SetResourceVersion(version string)                      {}
func (*InvalidRuntimeType) GetGeneration() int64                                   { return 0 }
func (*InvalidRuntimeType) SetGeneration(generation int64)                         {}
func (*InvalidRuntimeType) GetSelfLink() string                                    { return "" }
func (*InvalidRuntimeType) SetSelfLink(selfLink string)                            {}
func (*InvalidRuntimeType) GetCreationTimestamp() v1.Time                          { return v1.Now() }
func (*InvalidRuntimeType) SetCreationTimestamp(timestamp v1.Time)                 {}
func (*InvalidRuntimeType) GetDeletionTimestamp() *v1.Time                         { return nil }
func (*InvalidRuntimeType) SetDeletionTimestamp(timestamp *v1.Time)                {}
func (*InvalidRuntimeType) GetDeletionGracePeriodSeconds() *int64                  { return nil }
func (*InvalidRuntimeType) SetDeletionGracePeriodSeconds(*int64)                   {}
func (*InvalidRuntimeType) GetLabels() map[string]string                           { return nil }
func (*InvalidRuntimeType) SetLabels(labels map[string]string)                     {}
func (*InvalidRuntimeType) GetAnnotations() map[string]string                      { return nil }
func (*InvalidRuntimeType) SetAnnotations(annotations map[string]string)           {}
func (*InvalidRuntimeType) GetFinalizers() []string                                { return nil }
func (*InvalidRuntimeType) SetFinalizers(finalizers []string)                      {}
func (*InvalidRuntimeType) GetOwnerReferences() []v1.OwnerReference                { return nil }
func (*InvalidRuntimeType) SetOwnerReferences([]v1.OwnerReference)                 {}
func (*InvalidRuntimeType) GetClusterName() string                                 { return "" }
func (*InvalidRuntimeType) SetClusterName(clusterName string)                      {}
func (*InvalidRuntimeType) GetManagedFields() []v1.ManagedFieldsEntry              { return nil }
func (*InvalidRuntimeType) SetManagedFields(managedFields []v1.ManagedFieldsEntry) {}

func (i *InvalidRuntimeType) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}
func (i *InvalidRuntimeType) DeepCopyObject() runtime.Object {
	return i
}

var _ = Describe("Asset Tests", func() {

	log := ctrl.Log

	var _ = Describe("Manager - load configmap from file and deploy", func() {
		var _ = It("Run Manager with no assets", func() {
			manager := Manager{Client: k8sClient, Log: log}
			Expect(manager.DeployConfigMaps(context.TODO())).ToNot(HaveOccurred())
		})
		var _ = It("Run Manager", func() {
			assets := []Asset{
				{
					log:  log,
					Path: "/tmp/dummy.bin",
				},
			}

			manager := Manager{Client: k8sClient,
				Log:    log,
				Assets: assets}

			getConfigMap = func(ctx context.Context, c client.Client, cmName string, ns string) (corev1.ConfigMap, error) {
				return corev1.ConfigMap{}, gerrors.New("not found")
			}

			Expect(manager.DeployConfigMaps(context.TODO())).To(HaveOccurred())
		})
		var _ = It("Run Manager loadFromDir", func() {
			assets := []Asset{
				{
					log:  log,
					Path: "/tmp/",
				},
			}

			manager := Manager{Client: k8sClient,
				Log:    log,
				Assets: assets}

			getConfigMap = func(ctx context.Context, c client.Client, cmName string, ns string) (corev1.ConfigMap, error) {
				return corev1.ConfigMap{}, gerrors.New("not found")
			}

			Expect(manager.DeployConfigMaps(context.TODO())).To(HaveOccurred())
		})
		var _ = It("Run Manager loadFromFile", func() {
			assets := []Asset{
				{
					log:           log,
					Path:          fakeAssetFile,
					ConfigMapName: fakeConfigMapName,
					substitutions: map[string]string{"one": "two"},
				},
			}

			manager := Manager{Client: k8sClient,
				Log:    log,
				Assets: assets,
				Owner:  fakeOwner,
				Scheme: scheme.Scheme}

			Expect(manager.DeployConfigMaps(context.TODO())).ToNot(HaveOccurred())
		})
		var _ = It("Run DeployConfigMaps (fail setting Owner)", func() {
			var invalidObject InvalidRuntimeType

			assets := []Asset{
				{
					log:           log,
					Path:          fakeAssetFile,
					substitutions: map[string]string{"one": "two"},
					objects: []client.Object{
						&invalidObject},
				},
			}

			manager := Manager{Client: k8sClient,
				Log:    log,
				Assets: assets,
				Owner:  fakeOwner,
				Scheme: scheme.Scheme}

			Expect(manager).ToNot(Equal(nil))

			node := &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Name: "dummy",
					Labels: map[string]string{
						"fpga.intel.com/intel-accelerator-present": "",
					},
				},
			}

			err := k8sClient.Create(context.Background(), node)
			Expect(err).ToNot(HaveOccurred())

			err = manager.DeployConfigMaps(context.TODO())
			Expect(err).To(HaveOccurred())

			// Cleanup
			err = k8sClient.Delete(context.TODO(), node)
			Expect(err).ToNot(HaveOccurred())
		})
		var _ = It("Run Manager loadFromFile (bad file)", func() {
			assets := []Asset{
				{
					log:           log,
					Path:          "/dev/null",
					substitutions: map[string]string{"one": "two"},
					BlockingReadiness: ReadinessPollConfig{
						Retries: 1,
					},
				},
			}

			manager := Manager{Client: k8sClient,
				Log:    log,
				Assets: assets,
				Owner:  fakeOwner,
				Scheme: scheme.Scheme}

			Expect(manager.DeployConfigMaps(context.TODO())).To(HaveOccurred())
		})
		var _ = It("Run Manager loadFromFile (missing file)", func() {
			assets := []Asset{
				{
					log:           log,
					Path:          "/dev/null_fake",
					substitutions: map[string]string{"one": "two"},
					BlockingReadiness: ReadinessPollConfig{
						Retries: 1,
					},
				},
			}

			manager := Manager{Client: k8sClient,
				Log:    log,
				Assets: assets,
				Owner:  fakeOwner,
				Scheme: scheme.Scheme}

			err := manager.DeployConfigMaps(context.TODO())
			Expect(err).To(HaveOccurred())
		})
	})

	var _ = Describe("Manager - load objects from configmap and deploy", func() {
		var _ = It("Run Manager loadFromConfigMap (no retries)", func() {
			assets := []Asset{
				{
					log:           log,
					Path:          fakeAssetFile,
					ConfigMapName: fakeConfigMapName,
					BlockingReadiness: ReadinessPollConfig{
						Retries: 1,
					},
				},
			}

			manager := Manager{Client: k8sClient,
				Log:    log,
				Assets: assets,
				Owner:  fakeOwner,
				Scheme: scheme.Scheme}

			getConfigMap = func(ctx context.Context, c client.Client, cmName string, ns string) (corev1.ConfigMap, error) {
				configMap := corev1.ConfigMap{
					Data: map[string]string{
						"daemonSet": "apiVersion: apps/v1\nkind: DaemonSet\nmetadata:\n  labels:\n    app: any-daemon\n  name: any-daemon\n  namespace: default\nspec:\n  minReadySeconds: 10\n  selector:\n    matchLabels:\n      app: any-app\n  template:\n    metadata:\n      labels:\n        app: any-app\n      name: test-app\n    spec:\n      serviceAccount: any-service-account\n      serviceAccountName: any-service-account\n      containers:\n      - image: \"any_image\"\n        name: test-app\n        securityContext:\n          readOnlyRootFilesystem: true",
					},
				}
				return configMap, nil
			}

			err := manager.LoadFromConfigMapAndDeploy(context.TODO())
			Expect(err).ToNot(HaveOccurred())
		})
		var _ = It("Run Manager loadFromConfigMap (invalid retries count)", func() {
			assets := []Asset{
				{
					log:  log,
					Path: fakeAssetFile,
					BlockingReadiness: ReadinessPollConfig{
						Retries: -1,
					},
				},
			}

			manager := Manager{Client: k8sClient,
				Log:    log,
				Assets: assets,
				Owner:  fakeOwner,
				Scheme: scheme.Scheme}

			getConfigMap = func(ctx context.Context, c client.Client, cmName string, ns string) (corev1.ConfigMap, error) {
				configMap := corev1.ConfigMap{
					Data: map[string]string{
						"daemonSet": "apiVersion: apps/v1\nkind: DaemonSet\nmetadata:\n  labels:\n    app: any-daemon\n  name: any-daemon\n  namespace: default\nspec:\n  minReadySeconds: 10\n  selector:\n    matchLabels:\n      app: any-app\n  template:\n    metadata:\n      labels:\n        app: any-app\n      name: test-app\n    spec:\n      serviceAccount: any-service-account\n      serviceAccountName: any-service-account\n      containers:\n      - image: \"any_image\"\n        name: test-app\n        securityContext:\n          readOnlyRootFilesystem: true",
					},
				}
				return configMap, nil
			}

			err := manager.LoadFromConfigMapAndDeploy(context.TODO())
			Expect(err).To(HaveOccurred())
		})
		var _ = It("Run Manager loadFromConfigMap (nonexistent configmap name)", func() {
			assets := []Asset{
				{
					log:           log,
					Path:          fakeAssetFile,
					ConfigMapName: fakeConfigMapName,
				},
			}

			manager := Manager{Client: k8sClient,
				Log:    log,
				Assets: assets,
				Owner:  fakeOwner,
				Scheme: scheme.Scheme}

			getConfigMap = getConfigMapData

			Expect(manager.LoadFromConfigMapAndDeploy(context.TODO())).To(HaveOccurred())
		})
		var _ = It("Run Manager loadFromConfigMap (update existing valid configmap)", func() {
			assets := []Asset{
				{
					log:           log,
					Path:          fakeAssetFile,
					ConfigMapName: fakeConfigMapName,
				},
			}

			manager := Manager{Client: k8sClient,
				Log:    log,
				Assets: assets,
				Owner:  fakeOwner,
				Scheme: scheme.Scheme}

			getConfigMap = func(ctx context.Context, c client.Client, cmName string, ns string) (corev1.ConfigMap, error) {
				configMap := corev1.ConfigMap{
					Data: map[string]string{
						"configMap":         "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: supported-clv-devices\n  namespace: default\nimmutable: false\ndata:\n  fake-key-1: fake-value-1\n  fake-key-2: fake-value-2",
						"configMap-updated": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: supported-clv-devices\n  namespace: default\nimmutable: false\ndata:\n  fake-key-1: new-fake-value-1\n  fake-key-2: fake-value-2",
					},
				}
				return configMap, nil
			}

			err := manager.LoadFromConfigMapAndDeploy(context.TODO())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
