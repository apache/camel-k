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
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/gzip"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
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

	var emptyDirVolume *corev1.Volume
	for _, v := range spec.Volumes {
		if v.Name == "my-empty-dir" {
			emptyDirVolume = &v
			break
		}
	}

	assert.NotNil(t, emptyDirVolume)
	// Default applied by operator
	assert.Equal(t, "500Mi", emptyDirVolume.EmptyDir.SizeLimit.String())

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

func TestEmptyDirVolumeWithSizeLimitIntegrationPhaseDeploying(t *testing.T) {
	traitCatalog := NewCatalog(nil)

	environment := getNominalEnv(t, traitCatalog)
	environment.Integration.Spec.Traits.Mount = &traitv1.MountTrait{
		EmptyDirs: []string{"my-empty-dir:/some/path:450Mi"},
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

	var emptyDirVolume *corev1.Volume
	for _, v := range spec.Volumes {
		if v.Name == "my-empty-dir" {
			emptyDirVolume = &v
			break
		}
	}
	assert.NotNil(t, emptyDirVolume)
	assert.Equal(t, "450Mi", emptyDirVolume.EmptyDir.SizeLimit.String())

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
	fakeClient, _ := internal.NewFakeClient(&pvc)
	catalog, _ := camel.DefaultCatalog()
	compressedRoute, _ := gzip.CompressBase64([]byte(`from("platform-http:test").log("hello");`))

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
							Name:        "routes.java",
							Content:     string(compressedRoute),
							Compression: true,
						},
						Language: v1.LanguageJavaSource,
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
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
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
	fakeClient, _ := internal.NewFakeClient(&sc)
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
	fakeClient, _ := internal.NewFakeClient(&sc)
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

func TestConfigureVolumesAndMountsSources(t *testing.T) {
	trait, _ := newMountTrait().(*mountTrait)
	env := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      TestDeploymentName,
				Namespace: "ns",
			},
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:       "source1.java",
							ContentRef: "my-cm1",
							ContentKey: "source1.java",
						},
						Type: "data",
					},
					{
						DataSpec: v1.DataSpec{
							Name:       "source2.java",
							ContentRef: "my-cm2",
						},
						Type: "data",
					},
				},
			},
		},
		Catalog: &Catalog{},
	}

	vols := make([]corev1.Volume, 0)
	mnts := make([]corev1.VolumeMount, 0)

	trait.configureCamelVolumesAndMounts(&env, &vols, &mnts)

	assert.Len(t, vols, 2)
	assert.Len(t, mnts, 2)

	v := findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "my-cm1" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "source1.java", v.VolumeSource.ConfigMap.Items[0].Key)

	m := findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == v.Name })
	assert.NotNil(t, m)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "my-cm2" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "content", v.VolumeSource.ConfigMap.Items[0].Key)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == v.Name })
	assert.NotNil(t, m)
}

func TestConfigureVolumesAndMountsSourcesInNativeMode(t *testing.T) {
	trait, _ := newMountTrait().(*mountTrait)
	traitList := make([]Trait, 0, len(FactoryList))
	quarkus, ok := newQuarkusTrait().(*quarkusTrait)
	assert.True(t, ok, "A Quarkus trait was expected")
	quarkus.Modes = []traitv1.QuarkusMode{traitv1.NativeQuarkusMode}
	traitList = append(traitList, quarkus)
	env := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      TestDeploymentName,
				Namespace: "ns",
			},
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:       "source1.xml",
							ContentRef: "my-cm1",
							ContentKey: "source1.xml",
						},
						Type: "data",
					},
					{
						DataSpec: v1.DataSpec{
							Name:       "source2.java",
							ContentRef: "my-cm2",
						},
						Type: "data",
					},
					{
						DataSpec: v1.DataSpec{
							Name:       "source1.java",
							ContentRef: "my-cm3",
							ContentKey: "source1.java",
						},
						Type: "data",
					},
					{
						DataSpec: v1.DataSpec{
							Name:       "source2.xml",
							ContentRef: "my-cm4",
						},
						Type: "data",
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					v1.IntegrationKitLayoutLabel: v1.IntegrationKitLayoutNativeSources,
				},
				Namespace: "ns",
			},
		},
		Catalog: &Catalog{
			traits: traitList,
		},
		CamelCatalog: &camel.RuntimeCatalog{
			CamelCatalogSpec: v1.CamelCatalogSpec{
				Loaders: map[string]v1.CamelLoader{
					"java": {
						Metadata: map[string]string{
							"native":                         boolean.TrueString,
							"sources-required-at-build-time": boolean.TrueString,
						},
					},
				},
			},
		},
	}

	vols := make([]corev1.Volume, 0)
	mnts := make([]corev1.VolumeMount, 0)

	trait.configureCamelVolumesAndMounts(&env, &vols, &mnts)

	assert.Len(t, vols, 2)
	assert.Len(t, mnts, 2)

	v := findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "my-cm1" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "source1.xml", v.VolumeSource.ConfigMap.Items[0].Key)

	m := findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == v.Name })
	assert.NotNil(t, m)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "my-cm4" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "content", v.VolumeSource.ConfigMap.Items[0].Key)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == v.Name })
	assert.NotNil(t, m)
}

func TestMountVolumesInitContainers(t *testing.T) {
	traitCatalog := NewCatalog(nil)
	environment := getNominalEnv(t, traitCatalog)
	// We must provide some init container
	environment.Integration.Spec.Traits.InitContainers = &traitv1.InitContainersTrait{
		InitTasks: []string{"init;my-init-image:1.2.3;echo hello"},
	}
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

	assert.Len(t, spec.InitContainers[0].VolumeMounts, 1)

	assert.Condition(t, func() bool {
		for _, v := range spec.InitContainers[0].VolumeMounts {
			if v.Name == "my-pvc" {
				return true
			}
		}
		return false
	})
}

func TestAgentVolume(t *testing.T) {
	traitCatalog := NewCatalog(nil)

	environment := getNominalEnv(t, traitCatalog)
	// Reset nominal test value
	environment.Integration.Spec.Traits.Mount = &traitv1.MountTrait{}
	environment.Integration.Spec.Traits.JVM = &traitv1.JVMTrait{
		Agents: []string{"my-agent;my-url"},
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

	var emptyDirVolume *corev1.Volume
	for _, v := range spec.Volumes {
		if v.Name == defaultAgentVolume {
			emptyDirVolume = &v
			break
		}
	}
	assert.NotNil(t, emptyDirVolume)

	assert.Condition(t, func() bool {
		for _, container := range spec.Containers {
			if container.Name == "integration" {
				for _, volumeMount := range container.VolumeMounts {
					if volumeMount.Name == defaultAgentVolume {
						return true
					}
				}
			}
		}
		return false
	})
}
