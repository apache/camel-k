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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	routev1 "github.com/openshift/api/route/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/resources"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

const (
	TestDeploymentName = "test"
)

func TestOpenShiftTraits(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterOpenShift, "from('timer:my-timer').to('log:info')")
	res := processTestEnv(t, env)

	assert.NotEmpty(t, env.ExecutedTraits)
	assert.NotNil(t, env.GetTrait("deployment"))
	assert.Nil(t, env.GetTrait("service"))
	assert.Nil(t, env.GetTrait("route"))
	assert.NotNil(t, env.GetTrait("owner"))
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
	res := processTestEnv(t, env)
	assert.NotNil(t, env.GetTrait("service"))
	assert.NotNil(t, env.GetTrait("route"))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == TestDeploymentName && svc.Spec.Ports[0].TargetPort.StrVal == "http"
	}))
}

func TestOpenShiftTraitsWithWebAndDisabledTrait(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterOpenShift, "from('netty-http:http').to('log:info')")
	env.Integration.Spec.Traits.Service = &traitv1.ServiceTrait{
		Trait: traitv1.Trait{
			Enabled: ptr.To(false),
		},
	}
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
	assert.NotNil(t, res.GetDeployment(func(deployment *appsv1.Deployment) bool {
		return deployment.Name == TestDeploymentName
	}))
	assert.NotNil(t, res.GetService(func(svc *corev1.Service) bool {
		return svc.Name == TestDeploymentName
	}))
}

func TestTraitHierarchyDecode(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterOpenShift, "")

	env.Platform.Spec.Traits.KnativeService = &traitv1.KnativeServiceTrait{
		Trait: traitv1.Trait{
			Enabled: ptr.To(false),
		},
		MinScale: ptr.To(1),
		MaxScale: ptr.To(10),
		Target:   ptr.To(15),
	}
	env.Platform.ResyncStatusFullConfig()

	env.Integration.Spec.Traits.KnativeService = &traitv1.KnativeServiceTrait{
		Trait: traitv1.Trait{
			Enabled: ptr.To(true),
		},
		MinScale: ptr.To(5),
		MaxScale: ptr.To(20),
	}

	c := NewTraitTestCatalog()
	err := c.Configure(env)

	require.NoError(t, err)

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

func TestOnlySomeTraitsInfluenceBuild(t *testing.T) {
	c := NewTraitTestCatalog()
	buildTraits := []string{"builder", "camel", "quarkus", "registry"}

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
	platformTraits := []string{
		"builder", "camel", "jvm", "runtime", "container", "security-context", "mount", "dependencies", "deployer",
		"deployment", "environment", "error-handler", "init-containers", "kamelets", "openapi", "owner", "platform", "quarkus",
	}

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
	t.Helper()

	catalog := NewTraitTestCatalog()
	_, _, err := catalog.apply(env)
	require.NoError(t, err)
	return env.Resources
}

func createTestEnv(t *testing.T, cluster v1.IntegrationPlatformCluster, script string) *Environment {
	t.Helper()

	client, _ := internal.NewFakeClient()
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	res := &Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(nil),
		Client:       client,
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
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeVersion: catalog.Runtime.Version,
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
	res.Platform.ResyncStatusFullConfig()
	return res
}

func NewTraitTestCatalog() *Catalog {
	return NewCatalog(nil)
}

func TestExecutedTraitsCondition(t *testing.T) {
	env := createTestEnv(t, v1.IntegrationPlatformClusterOpenShift, "from('timer:test').to('log:info')")
	catalog := NewTraitTestCatalog()
	conditions, traits, err := catalog.apply(env)
	require.NoError(t, err)
	assert.Empty(t, traits)
	expectedCondition := NewIntegrationCondition(
		"",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		"TraitConfiguration",
		"Applied traits: camel,environment,logging,deployer,deployment,gc,container,security-context,mount,quarkus,jvm,owner",
	)
	assert.Contains(t, conditions, expectedCondition)
}

func testDefaultIntegrationPhaseTraitsSetting(t *testing.T, phase v1.IntegrationPhase) {
	it := &v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-it",
			Namespace: "ns",
		},
		Spec: v1.IntegrationSpec{
			Sources: []v1.SourceSpec{
				{
					DataSpec: v1.DataSpec{
						Name:    "file.groovy",
						Content: "from('timer:test').to('log:info')",
					},
					Language: v1.LanguageGroovy,
				},
			},
		},
		Status: v1.IntegrationStatus{
			Phase: phase,
			Conditions: []v1.IntegrationCondition{
				{
					Type:   v1.IntegrationConditionDeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	platform := &v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-platform",
			Namespace: "ns",
		},
		Status: v1.IntegrationPlatformStatus{
			IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeProvider: v1.RuntimeProviderQuarkus,
					RuntimeVersion:  defaults.DefaultRuntimeVersion,
				},
			},
			Phase: v1.IntegrationPlatformPhaseReady,
		},
	}
	// Load the default catalog
	camelCatalogData, err := resources.Resource(fmt.Sprintf("/resources/camel-catalog-%s.yaml", defaults.DefaultRuntimeVersion))
	require.NoError(t, err)
	var cat v1.CamelCatalog
	err = yaml.Unmarshal(camelCatalogData, &cat)
	require.NoError(t, err)
	cat.SetAnnotations(platform.Annotations)
	cat.SetNamespace(platform.Namespace)

	client, err := internal.NewFakeClient(platform, &cat)
	require.NoError(t, err)
	env, err := Apply(context.Background(), client, it, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, env.Integration)
	// We should get no values slicking from the spec
	// only traits marked as "auto" should be able to scan
	// the sources and set values.
	// Neither default traits values should not be included here.
	assert.Equal(t, &v1.Traits{}, env.Integration.Status.Traits)
}

func TestDefaultIntegrationRunningTraitsSetting(t *testing.T) {
	testDefaultIntegrationPhaseTraitsSetting(t, v1.IntegrationPhaseRunning)
}

func TestDefaultIntegrationNoneTraitsSetting(t *testing.T) {
	testDefaultIntegrationPhaseTraitsSetting(t, v1.IntegrationPhaseNone)
}

func TestDefaultIntegrationInitTraitsSetting(t *testing.T) {
	testDefaultIntegrationPhaseTraitsSetting(t, v1.IntegrationPhaseInitialization)
}

func TestDefaultIntegrationWaitingPlatformTraitsSetting(t *testing.T) {
	testDefaultIntegrationPhaseTraitsSetting(t, v1.IntegrationPhaseWaitingForPlatform)
}

func TestDefaultIntegrationBuildingKitTraitsSetting(t *testing.T) {
	testDefaultIntegrationPhaseTraitsSetting(t, v1.IntegrationPhaseBuildingKit)
}

func TestDefaultIntegrationDeployingTraitsSetting(t *testing.T) {
	testDefaultIntegrationPhaseTraitsSetting(t, v1.IntegrationPhaseDeploying)
}

func TestDefaultIntegrationErrorTraitsSetting(t *testing.T) {
	testDefaultIntegrationPhaseTraitsSetting(t, v1.IntegrationPhaseError)
}

func TestIntegrationTraitsSetting(t *testing.T) {
	it := &v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-it",
			Namespace: "ns",
		},
		Spec: v1.IntegrationSpec{
			Sources: []v1.SourceSpec{
				{
					DataSpec: v1.DataSpec{
						Name:    "file.groovy",
						Content: "from('timer:test').to('log:info')",
					},
					Language: v1.LanguageGroovy,
				},
			},
			Traits: v1.Traits{
				Builder: &traitv1.BuilderTrait{
					Properties: []string{
						"building=yes",
					},
				},
				Camel: &traitv1.CamelTrait{
					Properties: []string{
						"hello=world",
					},
				},
				Container: &traitv1.ContainerTrait{
					Name: "my-container-name",
				},
				Addons: map[string]v1.AddonTrait{
					"master": toAddonTrait(t, map[string]interface{}{
						"resourceName": "test-lock",
					}),
				},
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
			Conditions: []v1.IntegrationCondition{
				{
					Type:   v1.IntegrationConditionDeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	platform := &v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-platform",
			Namespace: "ns",
		},
		Status: v1.IntegrationPlatformStatus{
			IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeProvider: v1.RuntimeProviderQuarkus,
					RuntimeVersion:  defaults.DefaultRuntimeVersion,
				},
			},
			Phase: v1.IntegrationPlatformPhaseReady,
		},
	}
	// Load the default catalog
	camelCatalogData, err := resources.Resource(fmt.Sprintf("/resources/camel-catalog-%s.yaml", defaults.DefaultRuntimeVersion))
	require.NoError(t, err)
	var cat v1.CamelCatalog
	err = yaml.Unmarshal(camelCatalogData, &cat)
	require.NoError(t, err)
	cat.SetAnnotations(platform.Annotations)
	cat.SetNamespace(platform.Namespace)

	client, err := internal.NewFakeClient(platform, &cat)
	require.NoError(t, err)
	env, err := Apply(context.Background(), client, it, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, env.Integration)
	assert.Equal(t, &v1.Traits{
		Builder: &traitv1.BuilderTrait{
			Properties: []string{
				"building=yes",
			},
		},
		Camel: &traitv1.CamelTrait{
			Properties: []string{
				"hello=world",
			},
		},
		Container: &traitv1.ContainerTrait{
			Name: "my-container-name",
		},
		Addons: map[string]v1.AddonTrait{
			"master": toAddonTrait(t, map[string]interface{}{
				"resourceName": "test-lock",
			}),
		},
	}, env.Integration.Status.Traits)
}
