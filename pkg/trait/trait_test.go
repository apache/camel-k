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

	"github.com/scylladb/go-set/strset"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestDeployment = "test"
	TestProperties = "test-properties"
)

func TestOpenShiftTraits(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterOpenShift, "camel:core")
	res := processTestEnv(t, env)

	assert.NotEmpty(t, env.ExecutedTraits)
	assert.NotNil(t, env.GetTrait(ID("deployment")))
	assert.Nil(t, env.GetTrait(ID("service")))
	assert.Nil(t, env.GetTrait(ID("route")))
	assert.NotNil(t, env.GetTrait(ID("owner")))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == TestProperties
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeployment
	}))
}

func TestOpenShiftTraitsWithWeb(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterOpenShift, "from('undertow:http').to('log:info')")
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait(ID("deployment")))
	assert.NotNil(t, env.GetTrait(ID("service")))
	assert.NotNil(t, env.GetTrait(ID("route")))
	assert.NotNil(t, env.GetTrait(ID("owner")))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == TestProperties
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeployment
	}))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == TestDeployment
	}))
	assert.NotNil(t, res.GetRoute(func(svc *routev1.Route) bool {
		return svc.Name == TestDeployment
	}))
}

func TestOpenShiftTraitsWithWebAndConfig(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterOpenShift, "from('undertow:http').to('log:info')")
	env.Integration.Spec.Traits = make(map[string]v1alpha1.TraitSpec)
	env.Integration.Spec.Traits["service"] = v1alpha1.TraitSpec{
		Configuration: map[string]string{
			"port": "7071",
		},
	}
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait(ID("service")))
	assert.NotNil(t, env.GetTrait(ID("route")))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == TestDeployment && svc.Spec.Ports[0].TargetPort.StrVal == "http"
	}))
}

func TestOpenShiftTraitsWithWebAndDisabledTrait(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterOpenShift, "from('undertow:http').to('log:info')")
	env.Integration.Spec.Traits = make(map[string]v1alpha1.TraitSpec)
	env.Integration.Spec.Traits["service"] = v1alpha1.TraitSpec{
		Configuration: map[string]string{
			"enabled": "false",
			"port":    "7071",
		},
	}
	res := processTestEnv(t, env)
	assert.Nil(t, env.GetTrait(ID("service")))
	assert.Nil(t, env.GetTrait(ID("route"))) // No route without service
	assert.Nil(t, res.GetService(func(svc *corev1.Service) bool {
		return true
	}))
}

func TestKubernetesTraits(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterKubernetes, "from('timer:tick').to('log:info')")
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait(ID("deployment")))
	assert.Nil(t, env.GetTrait(ID("service")))
	assert.Nil(t, env.GetTrait(ID("route")))
	assert.NotNil(t, env.GetTrait(ID("owner")))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == TestProperties
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeployment
	}))
}

func TestKubernetesTraitsWithWeb(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterKubernetes, "from('servlet:http').to('log:info')")
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait(ID("deployment")))
	assert.NotNil(t, env.GetTrait(ID("service")))
	assert.Nil(t, env.GetTrait(ID("route")))
	assert.NotNil(t, env.GetTrait(ID("owner")))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Name == TestProperties
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeployment
	}))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == TestDeployment
	}))
}

func TestTraitDecode(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterOpenShift, "")
	env.Integration.Spec.Traits = make(map[string]v1alpha1.TraitSpec)
	svcTrait := v1alpha1.TraitSpec{
		Configuration: map[string]string{
			"enabled": "false",
			"port":    "7071",
			"cippa":   "lippa",
		},
	}
	env.Integration.Spec.Traits["service"] = svcTrait

	svc := newServiceTrait()
	err := svcTrait.Decode(svc)

	assert.Nil(t, err)
	assert.Equal(t, 7071, svc.Port)
	assert.NotNil(t, svc.Enabled)
	assert.Equal(t, false, *svc.Enabled)
}

func TestTraitHierarchyDecode(t *testing.T) {
	env := createTestEnv(t, v1alpha1.IntegrationPlatformClusterOpenShift, "")

	env.Platform.Spec.Traits = make(map[string]v1alpha1.TraitSpec)
	env.Platform.Spec.Traits["knative-service"] = v1alpha1.TraitSpec{
		Configuration: map[string]string{
			"enabled":            "false",
			"min-scale":          "1",
			"max-scale":          "10",
			"autoscaling-target": "15",
		},
	}

	env.IntegrationContext.Spec.Traits = make(map[string]v1alpha1.TraitSpec)
	env.IntegrationContext.Spec.Traits["knative-service"] = v1alpha1.TraitSpec{
		Configuration: map[string]string{
			"enabled":   "true",
			"min-scale": "5",
		},
	}

	env.Integration.Spec.Traits = make(map[string]v1alpha1.TraitSpec)
	env.Integration.Spec.Traits["knative-service"] = v1alpha1.TraitSpec{
		Configuration: map[string]string{
			"max-scale": "20",
		},
	}

	c := NewTraitTestCatalog()
	err := c.configure(env)

	assert.Nil(t, err)

	knt := c.GetTrait("knative-service")
	assert.NotNil(t, knt)

	kns, ok := knt.(*knativeServiceTrait)
	assert.True(t, ok)
	assert.NotNil(t, kns)

	assert.NotNil(t, kns.Enabled)
	assert.True(t, *kns.Enabled)

	assert.NotNil(t, kns.MinScale)
	assert.Equal(t, 5, *kns.MinScale)

	assert.NotNil(t, kns.MaxScale)
	assert.Equal(t, 20, *kns.MaxScale)

	assert.NotNil(t, kns.Target)
	assert.Equal(t, 15, *kns.Target)
}

func TestConfigureVolumesAndMounts(t *testing.T) {
	env := Environment{
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      TestDeployment,
				Namespace: "ns",
			},
			Spec: v1alpha1.IntegrationSpec{
				Resources: []v1alpha1.ResourceSpec{
					{
						DataSpec: v1alpha1.DataSpec{
							Name:       "res1.txt",
							ContentRef: "my-cm1",
							ContentKey: "my-key1",
						},
						Type:      "data",
						MountPath: "/etc/m1",
					},
					{
						DataSpec: v1alpha1.DataSpec{
							Name:       "res2.txt",
							ContentRef: "my-cm2",
						},
						Type: "data",
					},
					{
						DataSpec: v1alpha1.DataSpec{
							Name:       "res3.txt",
							ContentKey: "my-key3",
						},
						Type: "data",
					},
					{
						DataSpec: v1alpha1.DataSpec{
							Name: "res4.txt",
						},
						Type: "data",
					},
				},
				Configuration: []v1alpha1.ConfigurationSpec{
					{
						Type:  "configmap",
						Value: "test-configmap",
					},
					{
						Type:  "secret",
						Value: "test-secret",
					},
					{
						Type:  "volume",
						Value: "testvolume:/foo/bar",
					},
					{
						Type:  "volume",
						Value: "an-invalid-volume-spec",
					},
				},
			},
		},
	}

	vols := make([]corev1.Volume, 0)
	mnts := make([]corev1.VolumeMount, 0)

	env.ConfigureVolumesAndMounts(false, &vols, &mnts)

	assert.Len(t, vols, 8)
	assert.Len(t, mnts, 8)

	v := findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "my-cm1" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "my-key1", v.VolumeSource.ConfigMap.Items[0].Key)
	assert.Equal(t, "res1.txt", v.VolumeSource.ConfigMap.Items[0].Path)

	m := findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "i-resource-000" })
	assert.NotNil(t, m)
	assert.Equal(t, "/etc/m1", m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "my-cm2" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "content", v.VolumeSource.ConfigMap.Items[0].Key)
	assert.Equal(t, "res2.txt", v.VolumeSource.ConfigMap.Items[0].Path)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "i-resource-001" })
	assert.NotNil(t, m)
	assert.Equal(t, "/etc/camel/resources/i-resource-001", m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == TestDeployment+"-resource-002" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "my-key3", v.VolumeSource.ConfigMap.Items[0].Key)
	assert.Equal(t, "res3.txt", v.VolumeSource.ConfigMap.Items[0].Path)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "i-resource-002" })
	assert.NotNil(t, m)
	assert.Equal(t, "/etc/camel/resources/i-resource-002", m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == TestDeployment+"-resource-003" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "content", v.VolumeSource.ConfigMap.Items[0].Key)
	assert.Equal(t, "res4.txt", v.VolumeSource.ConfigMap.Items[0].Path)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "i-resource-003" })
	assert.NotNil(t, m)
	assert.Equal(t, "/etc/camel/resources/i-resource-003", m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "test-configmap" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.NotNil(t, v.VolumeSource.ConfigMap.LocalObjectReference)
	assert.Equal(t, "test-configmap", v.VolumeSource.ConfigMap.LocalObjectReference.Name)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "test-configmap" })
	assert.NotNil(t, m)
	assert.Equal(t, "/etc/camel/conf.d/integration-cm-test-configmap", m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.Name == "test-secret" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.Secret)
	assert.Equal(t, "test-secret", v.Secret.SecretName)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "test-secret" })
	assert.NotNil(t, m)
	assert.Equal(t, "/etc/camel/conf.d/integration-secret-test-secret", m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.Name == "testvolume-data" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource)
	assert.NotNil(t, v.VolumeSource.PersistentVolumeClaim)
	assert.Equal(t, "testvolume", v.VolumeSource.PersistentVolumeClaim.ClaimName)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "testvolume-data" })
	assert.NotNil(t, m)
	assert.Equal(t, "/foo/bar", m.MountPath)
}

func findVolume(vols []corev1.Volume, condition func(corev1.Volume) bool) *corev1.Volume {
	for _, v := range vols {
		v := v
		if condition(v) {
			return &v
		}
	}

	return nil
}

func findVVolumeMount(vols []corev1.VolumeMount, condition func(corev1.VolumeMount) bool) *corev1.VolumeMount {
	for _, v := range vols {
		v := v
		if condition(v) {
			return &v
		}
	}

	return nil
}

func processTestEnv(t *testing.T, env *Environment) *kubernetes.Collection {
	catalog := NewTraitTestCatalog()
	err := catalog.apply(env)
	assert.Nil(t, err)
	return env.Resources
}

func createTestEnv(t *testing.T, cluster v1alpha1.IntegrationPlatformCluster, script string) *Environment {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	return &Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(context.TODO(), nil),
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      TestDeployment,
				Namespace: "ns",
			},
			Spec: v1alpha1.IntegrationSpec{
				Sources: []v1alpha1.SourceSpec{
					{
						DataSpec: v1alpha1.DataSpec{
							Name:    "file.groovy",
							Content: script,
						},
						Language: v1alpha1.LanguageGroovy,
					},
				},
			},
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
		},
		IntegrationContext: &v1alpha1.IntegrationContext{
			Status: v1alpha1.IntegrationContextStatus{
				Phase: v1alpha1.IntegrationContextPhaseReady,
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: cluster,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
		Classpath:      strset.New(),
	}
}

func NewTraitTestCatalog() *Catalog {
	return NewCatalog(context.TODO(), nil)
}
