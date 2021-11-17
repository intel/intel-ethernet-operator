// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package assets

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

//  runtime.Object implementation
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

	var _ = Describe("Manager", func() {
		var _ = It("Run Manager with no assets", func() {
			manager := Manager{Client: k8sClient, Log: log}
			Expect(manager.LoadAndDeploy(context.TODO())).ToNot(HaveOccurred())
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

			Expect(manager.LoadAndDeploy(context.TODO())).To(HaveOccurred())
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

			Expect(manager.LoadAndDeploy(context.TODO())).To(HaveOccurred())
		})
		var _ = It("Run Manager loadFromFile (no retries)", func() {

			assets := []Asset{
				{
					log:           log,
					Path:          fakeAssetFile,
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

			Expect(manager.LoadAndDeploy(context.TODO())).ToNot(HaveOccurred())
		})
		var _ = It("Run Manager loadFromFile", func() {
			assets := []Asset{
				{
					log:           log,
					Path:          fakeAssetFile,
					substitutions: map[string]string{"one": "two"},
				},
			}

			manager := Manager{Client: k8sClient,
				Log:    log,
				Assets: assets,
				Owner:  fakeOwner,
				Scheme: scheme.Scheme}

			Expect(manager.LoadAndDeploy(context.TODO())).ToNot(HaveOccurred())
		})
		var _ = It("Run LoadAndDeploy (fail setting Owner)", func() {

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

			err = manager.LoadAndDeploy(context.TODO())
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

			Expect(manager.LoadAndDeploy(context.TODO())).To(HaveOccurred())
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

			err := manager.LoadAndDeploy(context.TODO())
			Expect(err).To(HaveOccurred())
		})
		var _ = It("Run Manager loadFromFile (invalid retries count)", func() {
			assets := []Asset{
				{
					log:           log,
					Path:          fakeAssetFile,
					substitutions: map[string]string{"one": "two"},
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

			err := manager.LoadAndDeploy(context.TODO())
			Expect(err).To(HaveOccurred())
		})
	})
})
