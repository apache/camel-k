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
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	routev1 "github.com/openshift/api/route/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

const (
	TestDeploymentName = "test"
)

func TestOpenShiftTraits(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterOpenShift, "camel:core")
	res := processTestEnv(t, env)

	assert.NotEmpty(t, env.ExecutedTraits)
	assert.NotNil(t, env.GetTrait("deployment"))
	assert.Nil(t, env.GetTrait("service"))
	assert.Nil(t, env.GetTrait("route"))
	assert.NotNil(t, env.GetTrait("owner"))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels["camel.apache.org/properties.type"] != ""
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeploymentName
	}))
}

func TestOpenShiftTraitsWithWeb(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterOpenShift, "from('netty-http:http').to('log:info')")
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait("deployment"))
	assert.NotNil(t, env.GetTrait("service"))
	assert.NotNil(t, env.GetTrait("route"))
	assert.NotNil(t, env.GetTrait("owner"))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels["camel.apache.org/properties.type"] != ""
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeploymentName
	}))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == TestDeploymentName
	}))
	assert.NotNil(t, res.GetRoute(func(svc *routev1.Route) bool {
		return svc.Name == TestDeploymentName
	}))
}

func TestOpenShiftTraitsWithWebAndConfig(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterOpenShift, "from('netty-http:http').to('log:info')")
	env.Integration.Spec.Traits = make(map[string]v1.TraitSpec)
	env.Integration.Spec.Traits["service"] = test.TraitSpecFromMap(t, map[string]interface{}{
		"port": 7071,
	})
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait("service"))
	assert.NotNil(t, env.GetTrait("route"))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == TestDeploymentName && svc.Spec.Ports[0].TargetPort.StrVal == "http"
	}))
}

func TestOpenShiftTraitsWithWebAndDisabledTrait(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterOpenShift, "from('netty-http:http').to('log:info')")
	env.Integration.Spec.Traits = make(map[string]v1.TraitSpec)
	env.Integration.Spec.Traits["service"] = test.TraitSpecFromMap(t, map[string]interface{}{
		"enabled": false,
		"port":    7071,
	})
	res := processTestEnv(t, env)
	assert.Nil(t, env.GetTrait("service"))
	assert.Nil(t, env.GetTrait("route")) // No route without service
	assert.Nil(t, res.GetService(func(svc *corev1.Service) bool {
		return true
	}))
}

func TestKubernetesTraits(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterKubernetes, "from('timer:tick').to('log:info')")
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait("deployment"))
	assert.Nil(t, env.GetTrait("service"))
	assert.Nil(t, env.GetTrait("route"))
	assert.NotNil(t, env.GetTrait("owner"))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels["camel.apache.org/properties.type"] != ""
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeploymentName
	}))
}

func TestKubernetesTraitsWithWeb(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterKubernetes, "from('servlet:http').to('log:info')")
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait("deployment"))
	assert.NotNil(t, env.GetTrait("service"))
	assert.Nil(t, env.GetTrait("route"))
	assert.NotNil(t, env.GetTrait("owner"))
	assert.NotNil(t, res.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels["camel.apache.org/properties.type"] != ""
	}))
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeploymentName
	}))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == TestDeploymentName
	}))
}

func TestTraitDecode(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterOpenShift, "")
	env.Integration.Spec.Traits = make(map[string]v1.TraitSpec)
	svcTrait := test.TraitSpecFromMap(t, map[string]interface{}{
		"enabled": false,
		"port":    7071,
		"cippa":   "lippa",
	})
	env.Integration.Spec.Traits["service"] = svcTrait

	ctr := newContainerTrait().(*containerTrait)
	err := decodeTraitSpec(&svcTrait, ctr)

	assert.Nil(t, err)
	assert.Equal(t, 7071, ctr.Port)
	assert.NotNil(t, ctr.Enabled)
	assert.Equal(t, false, *ctr.Enabled)
}

func TestTraitHierarchyDecode(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterOpenShift, "")

	env.Platform.Spec.Traits = make(map[string]v1.TraitSpec)
	env.Platform.Spec.Traits["knative-service"] = test.TraitSpecFromMap(t, map[string]interface{}{
		"enabled":           false,
		"minScale":          1,
		"maxScale":          10,
		"autoscalingTarget": 15,
	})
	env.Platform.ResyncStatusFullConfig()

	env.IntegrationKit.Spec.Traits = make(map[string]v1.TraitSpec)
	env.IntegrationKit.Spec.Traits["knative-service"] = test.TraitSpecFromMap(t, map[string]interface{}{
		"enabled":  true,
		"minScale": 5,
	})

	env.Integration.Spec.Traits = make(map[string]v1.TraitSpec)
	env.Integration.Spec.Traits["knative-service"] = test.TraitSpecFromMap(t, map[string]interface{}{
		"maxScale": 20,
	})

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

func TestConfigureVolumesAndMountsTextResourcesAndProperties(t *testing.T) {
	env := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      TestDeploymentName,
				Namespace: "ns",
			},
			Spec: v1.IntegrationSpec{
				Resources: []v1.ResourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:       "res1.txt",
							ContentRef: "my-cm1",
							ContentKey: "my-key1",
						},
						Type:      "data",
						MountPath: "/etc/m1",
					},
					{
						DataSpec: v1.DataSpec{
							Name:       "res2.txt",
							ContentRef: "my-cm2",
						},
						Type: "data",
					},
					{
						DataSpec: v1.DataSpec{
							Name:       "res3.txt",
							ContentKey: "my-key3",
						},
						Type: "data",
					},
					{
						DataSpec: v1.DataSpec{
							Name: "res4.txt",
						},
						Type: "data",
					},
				},
				Configuration: []v1.ConfigurationSpec{
					{
						Type:  "property",
						Value: "a=b",
					},
					{
						Type:         "configmap",
						Value:        "test-configmap",
						ResourceType: "config",
					},
					{
						Type:         "secret",
						Value:        "test-secret",
						ResourceType: "config",
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

	env.Resources.AddAll(env.computeConfigMaps())
	env.configureVolumesAndMounts(&vols, &mnts)

	assert.Len(t, vols, 8)
	assert.Len(t, mnts, 8)

	v := findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "test-user-properties" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "application.properties", v.VolumeSource.ConfigMap.Items[0].Key)
	assert.Equal(t, "user.properties", v.VolumeSource.ConfigMap.Items[0].Path)

	m := findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == v.Name })
	assert.NotNil(t, m)
	assert.Equal(t, "/etc/camel/conf.d/user.properties", m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "my-cm1" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "my-key1", v.VolumeSource.ConfigMap.Items[0].Key)
	assert.Equal(t, "res1.txt", v.VolumeSource.ConfigMap.Items[0].Path)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "i-resource-000" })
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
	assert.Equal(t, "/etc/camel/resources/res2.txt", m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == TestDeploymentName+"-resource-002" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "my-key3", v.VolumeSource.ConfigMap.Items[0].Key)
	assert.Equal(t, "res3.txt", v.VolumeSource.ConfigMap.Items[0].Path)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "i-resource-002" })
	assert.NotNil(t, m)
	assert.Equal(t, "/etc/camel/resources/res3.txt", m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == TestDeploymentName+"-resource-003" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "content", v.VolumeSource.ConfigMap.Items[0].Key)
	assert.Equal(t, "res4.txt", v.VolumeSource.ConfigMap.Items[0].Path)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "i-resource-003" })
	assert.NotNil(t, m)
	assert.Equal(t, "/etc/camel/resources/res4.txt", m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "test-configmap" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.NotNil(t, v.VolumeSource.ConfigMap.LocalObjectReference)
	assert.Equal(t, "test-configmap", v.VolumeSource.ConfigMap.LocalObjectReference.Name)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "test-configmap" })
	assert.NotNil(t, m)
	assert.Equal(t, path.Join(configConfigmapsMountPath, "test-configmap"), m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.Name == "test-secret" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.Secret)
	assert.Equal(t, "test-secret", v.Secret.SecretName)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "test-secret" })
	assert.NotNil(t, m)
	assert.Equal(t, path.Join(configSecretsMountPath, "test-secret"), m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.Name == "testvolume-data" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource)
	assert.NotNil(t, v.VolumeSource.PersistentVolumeClaim)
	assert.Equal(t, "testvolume", v.VolumeSource.PersistentVolumeClaim.ClaimName)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == "testvolume-data" })
	assert.NotNil(t, m)
	assert.Equal(t, "/foo/bar", m.MountPath)
}

func TestConfigureVolumesAndMountsSources(t *testing.T) {
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
	}

	vols := make([]corev1.Volume, 0)
	mnts := make([]corev1.VolumeMount, 0)

	env.Resources.AddAll(env.computeConfigMaps())
	env.configureVolumesAndMounts(&vols, &mnts)

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

func TestConfigureVolumesAndMountsBinaryAndTextResources(t *testing.T) {
	env := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      TestDeploymentName,
				Namespace: "ns",
			},
			Spec: v1.IntegrationSpec{
				Resources: []v1.ResourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:        "res1.bin",
							RawContent:  []byte{1, 2, 3, 4},
							ContentRef:  "my-cm1",
							ContentKey:  "my-binary",
							ContentType: "application/octet-stream",
						},
						Type: "data",
					},
					{
						DataSpec: v1.DataSpec{
							Name:        "res2.txt",
							ContentRef:  "my-cm2",
							Content:     "hello",
							ContentKey:  "my-text",
							ContentType: "text/plain",
						},
						Type: "data",
					},
				},
			},
		},
	}

	vols := make([]corev1.Volume, 0)
	mnts := make([]corev1.VolumeMount, 0)

	env.Resources.AddAll(env.computeConfigMaps())
	env.configureVolumesAndMounts(&vols, &mnts)

	assert.Len(t, vols, 2)
	assert.Len(t, mnts, 2)

	v := findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "my-cm1" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "my-binary", v.VolumeSource.ConfigMap.Items[0].Key)
	assert.Equal(t, "res1.bin", v.VolumeSource.ConfigMap.Items[0].Path)

	m := findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == v.Name })
	assert.NotNil(t, m)
	assert.Equal(t, "/etc/camel/resources/res1.bin", m.MountPath)

	v = findVolume(vols, func(v corev1.Volume) bool { return v.ConfigMap.Name == "my-cm2" })
	assert.NotNil(t, v)
	assert.NotNil(t, v.VolumeSource.ConfigMap)
	assert.Len(t, v.VolumeSource.ConfigMap.Items, 1)
	assert.Equal(t, "my-text", v.VolumeSource.ConfigMap.Items[0].Key)
	assert.Equal(t, "res2.txt", v.VolumeSource.ConfigMap.Items[0].Path)

	m = findVVolumeMount(mnts, func(m corev1.VolumeMount) bool { return m.Name == v.Name })
	assert.NotNil(t, m)
	assert.Equal(t, "/etc/camel/resources/res2.txt", m.MountPath)
}

func TestOnlySomeTraitsInfluenceBuild(t *testing.T) {
	c := NewTraitTestCatalog()
	buildTraits := []string{"builder", "quarkus"}

	for _, trait := range c.AllTraits() {
		if trait.InfluencesKit() {
			assert.Contains(t, buildTraits, string(trait.ID()))
		} else {
			assert.NotContains(t, buildTraits, trait.ID())
		}
	}
}

func TestOnlySomeTraitsArePlatform(t *testing.T) {
	c := NewTraitTestCatalog()
	platformTraits := []string{"builder", "camel", "jvm", "configuration", "container", "dependencies", "deployer",
		"deployment", "environment", "error-handler", "kamelets", "openapi", "owner", "platform", "quarkus"}

	for _, trait := range c.AllTraits() {
		if trait.IsPlatformTrait() {
			assert.Contains(t, platformTraits, string(trait.ID()))
		} else {
			assert.NotContains(t, platformTraits, trait.ID())
		}
	}
}

func TestOnlySomeTraitsDoNotRequireIntegrationPlatform(t *testing.T) {
	c := NewTraitTestCatalog()
	doNotRequirePlatformTraits := []string{"deployer", "platform"}

	for _, trait := range c.AllTraits() {
		if !trait.RequiresIntegrationPlatform() {
			assert.Contains(t, doNotRequirePlatformTraits, string(trait.ID()))
		} else {
			assert.NotContains(t, doNotRequirePlatformTraits, trait.ID())
		}
	}
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

func createTestEnv(t *testing.T, cluster v1.IntegrationPlatformCluster, script string) *Environment {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	res := &Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(nil),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      TestDeploymentName,
				Namespace: "ns",
			},
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "file.groovy",
							Content: script,
						},
						Language: v1.LanguageGroovy,
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: cluster,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	res.Platform.ResyncStatusFullConfig()
	return res
}

func NewTraitTestCatalog() *Catalog {
	return NewCatalog(nil)
}
