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

package platform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
)

func TestFindIntegrationProfile(t *testing.T) {
	profile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom",
			Namespace: "ns",
		},
	}

	profile.ResyncStatusFullConfig()

	c, err := internal.NewFakeClient(&profile)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: "custom",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.NoError(t, err)
	assert.NotNil(t, found)
}

func TestFindIntegrationProfileWithNamespace(t *testing.T) {
	profile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom",
			Namespace: "other",
		},
	}

	profile.ResyncStatusFullConfig()

	c, err := internal.NewFakeClient(&profile)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation:          "custom",
				v1.IntegrationProfileNamespaceAnnotation: "other",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.NoError(t, err)
	assert.NotNil(t, found)
}

func TestFindIntegrationProfileInOperatorNamespace(t *testing.T) {
	profile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom",
			Namespace: "operator-namespace",
		},
	}

	profile.ResyncStatusFullConfig()

	c, err := internal.NewFakeClient(&profile)
	require.NoError(t, err)

	t.Setenv(operatorNamespaceEnvVariable, "operator-namespace")

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: "custom",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.NoError(t, err)
	assert.NotNil(t, found)
}

func TestApplyIntegrationProfile(t *testing.T) {
	profile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom",
			Namespace: "ns",
		},
		Spec: v1.IntegrationProfileSpec{
			Build: v1.IntegrationProfileBuildSpec{
				Maven: v1.MavenSpec{
					Properties: map[string]string{
						"global_prop1": "global_value1",
						"global_prop2": "global_value2",
					},
					CLIOptions: []string{
						"-V",
						"--no-transfer-progress",
						"-Dstyle.color=never",
						"-E",
					},
				},
				RuntimeVersion: "0.99.0",
			},
			Traits: v1.Traits{
				Logging: &trait.LoggingTrait{
					Level: "DEBUG",
				},
				Container: &trait.ContainerTrait{
					ImagePullPolicy: corev1.PullAlways,
					LimitCPU:        "0.1",
				},
			},
		},
	}

	profile.ResyncStatusFullConfig()

	c, err := internal.NewFakeClient(&profile)
	require.NoError(t, err)

	ip := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-camel-k",
			Namespace: "ns",
		},
		Spec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				BuildConfiguration: v1.BuildConfiguration{
					Strategy:      v1.BuildStrategyRoutine,
					OrderStrategy: v1.BuildOrderStrategyFIFO,
				},
			},
			Cluster: v1.IntegrationPlatformClusterOpenShift,
			Profile: v1.TraitProfileOpenShift,
		},
	}

	ip.ResyncStatusFullConfig()

	err = ConfigureDefaults(context.TODO(), c, &ip, true)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: "custom",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
		},
	}

	_, err = ApplyIntegrationProfile(context.TODO(), c, &ip, &integration)
	require.NoError(t, err)

	assert.Equal(t, v1.IntegrationPlatformClusterOpenShift, ip.Status.Cluster)
	assert.Equal(t, v1.TraitProfileOpenShift, ip.Status.Profile)
	assert.Equal(t, v1.BuildStrategyRoutine, ip.Status.Build.BuildConfiguration.Strategy)
	assert.Equal(t, v1.BuildOrderStrategyFIFO, ip.Status.Build.BuildConfiguration.OrderStrategy)
	assert.True(t, ip.Status.Build.MaxRunningBuilds == 3) // default for build strategy routine
	assert.Equal(t, len(profile.Status.Build.Maven.CLIOptions), len(ip.Status.Build.Maven.CLIOptions))
	assert.Equal(t, profile.Status.Build.Maven.CLIOptions, ip.Status.Build.Maven.CLIOptions)
	assert.NotNil(t, ip.Status.Traits)
	assert.NotNil(t, ip.Status.Traits.Logging)
	assert.Equal(t, "DEBUG", ip.Status.Traits.Logging.Level)
	assert.NotNil(t, ip.Status.Traits.Container)
	assert.Equal(t, corev1.PullAlways, ip.Status.Traits.Container.ImagePullPolicy)
	assert.Equal(t, "0.1", ip.Status.Traits.Container.LimitCPU)
	assert.Equal(t, 2, len(ip.Status.Build.Maven.Properties))
	assert.Equal(t, "global_value1", ip.Status.Build.Maven.Properties["global_prop1"])
	assert.Equal(t, "global_value2", ip.Status.Build.Maven.Properties["global_prop2"])
}

func TestApplyIntegrationProfileAndRetainPlatformSpec(t *testing.T) {
	profile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultPlatformName,
			Namespace: "ns",
		},
		Spec: v1.IntegrationProfileSpec{
			Build: v1.IntegrationProfileBuildSpec{
				Maven: v1.MavenSpec{
					Properties: map[string]string{
						"global_prop1": "global_value1",
						"global_prop2": "global_value2",
					},
				},
			},
			Traits: v1.Traits{
				Logging: &trait.LoggingTrait{
					Level: "DEBUG",
				},
				Container: &trait.ContainerTrait{
					ImagePullPolicy: corev1.PullIfNotPresent,
					LimitCPU:        "0.1",
				},
			},
		},
	}

	profile.ResyncStatusFullConfig()

	c, err := internal.NewFakeClient(&profile)
	require.NoError(t, err)

	ip := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-camel-k",
			Namespace: "ns",
		},
		Spec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				BuildConfiguration: v1.BuildConfiguration{
					Strategy:      v1.BuildStrategyPod,
					OrderStrategy: v1.BuildOrderStrategyFIFO,
				},
				MaxRunningBuilds: 1,
				Maven: v1.MavenSpec{
					Properties: map[string]string{
						"local_prop1":  "local_value1",
						"global_prop2": "local_value2",
					},
				},
			},
			Traits: v1.Traits{
				Container: &trait.ContainerTrait{
					ImagePullPolicy: corev1.PullAlways,
				},
			},
			Cluster: v1.IntegrationPlatformClusterKubernetes,
			Profile: v1.TraitProfileKnative,
		},
	}

	ip.ResyncStatusFullConfig()

	err = ConfigureDefaults(context.TODO(), c, &ip, true)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: DefaultPlatformName,
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
		},
	}

	_, err = ApplyIntegrationProfile(context.TODO(), c, &ip, &integration)
	require.NoError(t, err)

	assert.Equal(t, v1.IntegrationPlatformClusterKubernetes, ip.Status.Cluster)
	assert.Equal(t, v1.TraitProfileKnative, ip.Status.Profile)
	assert.Equal(t, v1.BuildStrategyPod, ip.Status.Build.BuildConfiguration.Strategy)
	assert.Equal(t, v1.BuildOrderStrategyFIFO, ip.Status.Build.BuildConfiguration.OrderStrategy)
	assert.True(t, ip.Status.Build.MaxRunningBuilds == 1)
	assert.Equal(t, 3, len(ip.Status.Build.Maven.CLIOptions))
	assert.NotNil(t, ip.Status.Traits)
	assert.NotNil(t, ip.Status.Traits.Logging)
	assert.Equal(t, "DEBUG", ip.Status.Traits.Logging.Level)
	assert.NotNil(t, ip.Status.Traits.Container)
	assert.Equal(t, corev1.PullAlways, ip.Status.Traits.Container.ImagePullPolicy)
	assert.Equal(t, "0.1", ip.Status.Traits.Container.LimitCPU)
	assert.Equal(t, 3, len(ip.Status.Build.Maven.Properties))
	assert.Equal(t, "global_value1", ip.Status.Build.Maven.Properties["global_prop1"])
	assert.Equal(t, "local_value2", ip.Status.Build.Maven.Properties["global_prop2"])
	assert.Equal(t, "local_value1", ip.Status.Build.Maven.Properties["local_prop1"])
}
