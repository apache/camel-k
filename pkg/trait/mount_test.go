/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package trait

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/gzip"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/test"
)

func TestMountVolumesEmpty(t *testing.T) {
	traitCatalog := NewCatalog(nil)

	environment := getNominalEnv(t, traitCatalog)
	environment.Integration.Spec.Traits = v1.Traits{} // empty traits
	environment.Platform.ResyncStatusFullConfig()

	conditions, traits, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("mount"))

	s := environment.Resources.GetDeployment(func(service *appsv1.Deployment) bool {
		return service.Name == "hello"
	})
	assert.NotNil(t, s)
	spec := s.Spec.Template.Spec

	assert.Len(t, spec.Containers[0].VolumeMounts, 2)
	assert.Len(t, spec.Volumes, 2)
}

func TestMountVolumesIntegrationPhaseDeploying(t *testing.T) {
	traitCatalog := NewCatalog(nil)
	environment := getNominalEnv(t, traitCatalog)
	environment.Platform.ResyncStatusFullConfig()

	conditions, traits, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("mount"))

	s := environment.Resources.GetDeployment(func(service *appsv1.Deployment) bool {
		return service.Name == "hello"
	})
	assert.NotNil(t, s)
	spec := s.Spec.Template.Spec

	assert.Len(t, spec.Containers[0].VolumeMounts, 5)
	assert.Len(t, spec.Volumes, 5)

	assert.Condition(t, func() bool {
		for _, v := range spec.Containers[0].VolumeMounts {
			if v.Name == "my-pvc" {
				return true
			}
		}
		return false
	})
	assert.Condition(t, func() bool {
		for _, v := range spec.Containers[0].VolumeMounts {
			if v.Name == "my-cm" {
				return true
			}
		}
		return false
	})
	assert.Condition(t, func() bool {
		for _, v := range spec.Volumes {
			if v.Name == "my-secret" {
				return true
			}
		}
		return false
	})
}

func TestEmptyDirVolumeIntegrationPhaseDeploying(t *testing.T) {
	traitCatalog := NewCatalog(nil)

	environment := getNominalEnv(t, traitCatalog)
	environment.Integration.Spec.Traits.Mount = &traitv1.MountTrait{
		EmptyDirs: []string{"my-empty-dir:/some/path"},
	}
	environment.Platform.ResyncStatusFullConfig()
	conditions, traits, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("mount"))

	deployment := environment.Resources.GetDeployment(func(service *appsv1.Deployment) bool {
		return service.Name == "hello"
	})
	assert.NotNil(t, deployment)
	spec := deployment.Spec.Template.Spec

	assert.Len(t, spec.Containers[0].VolumeMounts, 3)
	assert.Len(t, spec.Volumes, 3)

	assert.Condition(t, func() bool {
		for _, v := range spec.Volumes {
			if v.Name == "my-empty-dir" {
				return true
			}
		}
		return false
	})
	assert.Condition(t, func() bool {
		for _, container := range spec.Containers {
			if container.Name == "integration" {
				for _, volumeMount := range container.VolumeMounts {
					if volumeMount.Name == "my-empty-dir" {
						return true
					}
				}
			}
		}
		return false
	})
}

func TestMountVolumesIntegrationPhaseInitialization(t *testing.T) {
	traitCatalog := NewCatalog(nil)

	environment := getNominalEnv(t, traitCatalog)
	environment.Integration.Status.Phase = v1.IntegrationPhaseInitialization
	environment.Platform.ResyncStatusFullConfig()

	conditions, traits, err := traitCatalog.apply(environment)

	require.NoError(t, err)
	assert.Empty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Nil(t, environment.GetTrait("mount"))

	s := environment.Resources.GetDeployment(func(service *appsv1.Deployment) bool {
		return service.Name == "hello"
	})
	assert.Nil(t, s)
}

func getNominalEnv(t *testing.T, traitCatalog *Catalog) *Environment {
	t.Helper()
	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "my-pvc",
		},
	}
	fakeClient, _ := test.NewFakeClient(&pvc)
	catalog, _ := camel.DefaultCatalog()
	compressedRoute, _ := gzip.CompressBase64([]byte(`from("platform-http:test").log("hello")`))

	return &Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Ctx:          context.Background(),
		Client:       fakeClient,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hello",
				Namespace: "default",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:        "routes.js",
							Content:     string(compressedRoute),
							Compression: true,
						},
						Language: v1.LanguageJavaScript,
					},
				},
				Traits: v1.Traits{
					Mount: &traitv1.MountTrait{
						Configs:   []string{"configmap:my-cm"},
						Resources: []string{"secret:my-secret"},
						Volumes:   []string{"my-pvc:/over/the/rainbow"},
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
}

func TestMountVolumesExist(t *testing.T) {
	traitCatalog := NewCatalog(nil)
	e := getNominalEnv(t, traitCatalog)
	vol, vm, err := ParseAndCreateVolume(e, "my-pvc:/tmp/my-pvc")
	assert.NoError(t, err)
	assert.Equal(t, "my-pvc", vol.PersistentVolumeClaim.ClaimName)
	assert.Equal(t, "my-pvc", vm.Name)
	assert.Equal(t, "/tmp/my-pvc", vm.MountPath)
}

func TestMountVolumesNotExistAndFail(t *testing.T) {
	traitCatalog := NewCatalog(nil)
	e := getNominalEnv(t, traitCatalog)
	_, _, err := ParseAndCreateVolume(e, "my-pvc-2:/tmp/my-pvc")
	assert.Error(t, err)
	assert.Equal(t,
		"volume my-pvc-2 does not exist. Make sure to provide one or configure a dynamic PVC as trait volume configuration "+
			"pvcName:path/to/mount:size:accessMode<:storageClassName>", err.Error())
	// Wrong configuration
	_, _, err = ParseAndCreateVolume(e, "my-pvc-2:/tmp/my-pvc:fail")
	assert.Error(t, err)
	assert.Equal(t, "volume mount syntax error, must be name:path/to/mount:size:accessMode<:storageClassName> was my-pvc-2:/tmp/my-pvc:fail", err.Error())
	// Wrong size parsing
	_, _, err = ParseAndCreateVolume(e, "my-pvc-2:/tmp/my-pvc:10MM:ReadOnly")
	assert.Error(t, err)
	assert.Equal(t, "could not parse size 10MM, unable to parse quantity's suffix", err.Error())
	// No default storage class
	_, _, err = ParseAndCreateVolume(e, "my-pvc-2:/tmp/my-pvc:10Mi:ReadOnly")
	assert.Error(t, err)
	assert.Equal(t, "could not find any default StorageClass", err.Error())
	// No given storage class
	_, _, err = ParseAndCreateVolume(e, "my-pvc-2:/tmp/my-pvc:10Mi:ReadOnly:my-storage-class")
	assert.Error(t, err)
	assert.Equal(t, "could not find any my-storage-class StorageClass", err.Error())
}

func TestMountVolumesCreateDefaultStorageClass(t *testing.T) {
	traitCatalog := NewCatalog(nil)
	e := getNominalEnv(t, traitCatalog)
	sc := storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: storagev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "default-sc",
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
		},
	}
	fakeClient, _ := test.NewFakeClient(&sc)
	e.Client = fakeClient
	// Default storage class
	vol, vm, err := ParseAndCreateVolume(e, "my-pvc:/tmp/my-pvc:10Mi:ReadOnly")
	assert.NoError(t, err)
	assert.Equal(t, "my-pvc", vol.PersistentVolumeClaim.ClaimName)
	assert.Equal(t, "my-pvc", vm.Name)
	assert.Equal(t, "/tmp/my-pvc", vm.MountPath)
	pvc, err := kubernetes.LookupPersistentVolumeClaim(e.Ctx, e.Client, e.Integration.Namespace, "my-pvc")
	assert.NoError(t, err)
	assert.NotNil(t, pvc)
}

func TestMountVolumesCreateUserStorageClass(t *testing.T) {
	traitCatalog := NewCatalog(nil)
	e := getNominalEnv(t, traitCatalog)
	sc := storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: storagev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "my-sc",
		},
	}
	fakeClient, _ := test.NewFakeClient(&sc)
	e.Client = fakeClient
	// Default storage class
	vol, vm, err := ParseAndCreateVolume(e, "my-pvc:/tmp/my-pvc:10Mi:ReadOnly:my-sc")
	assert.NoError(t, err)
	assert.Equal(t, "my-pvc", vol.PersistentVolumeClaim.ClaimName)
	assert.Equal(t, "my-pvc", vm.Name)
	assert.Equal(t, "/tmp/my-pvc", vm.MountPath)
	pvc, err := kubernetes.LookupPersistentVolumeClaim(e.Ctx, e.Client, e.Integration.Namespace, "my-pvc")
	assert.NoError(t, err)
	assert.NotNil(t, pvc)
}
