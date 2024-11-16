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
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestIntegrationPlatformDefaults(t *testing.T) {
	ip := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultPlatformName,
			Namespace: "ns",
		},
	}

	c, err := internal.NewFakeClient(&ip)
	require.NoError(t, err)

	err = ConfigureDefaults(context.TODO(), c, &ip, false)
	require.NoError(t, err)

	assert.Equal(t, v1.IntegrationPlatformClusterKubernetes, ip.Status.Cluster)
	assert.Equal(t, v1.TraitProfile(""), ip.Status.Profile)
	assert.Equal(t, v1.BuildStrategyRoutine, ip.Status.Build.BuildConfiguration.Strategy)
	assert.Equal(t, v1.BuildOrderStrategyDependencies, ip.Status.Build.BuildConfiguration.OrderStrategy)
	assert.Equal(t, defaults.BaseImage(), ip.Status.Build.BaseImage)
	assert.Equal(t, defaults.LocalRepository, ip.Status.Build.Maven.LocalRepository)
	assert.Equal(t, int32(3), ip.Status.Build.MaxRunningBuilds) // default for build strategy routine
	assert.Len(t, ip.Status.Build.Maven.CLIOptions, 3)
	assert.NotNil(t, ip.Status.Traits)
}

func TestApplyGlobalPlatformSpec(t *testing.T) {
	global := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultPlatformName,
			Namespace: "ns",
		},
		Spec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				BuildConfiguration: v1.BuildConfiguration{
					Strategy:      v1.BuildStrategyRoutine,
					OrderStrategy: v1.BuildOrderStrategyFIFO,
				},
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
					ImagePullPolicy: corev1.PullAlways,
					LimitCPU:        "0.1",
				},
			},
			Cluster: v1.IntegrationPlatformClusterOpenShift,
			Profile: v1.TraitProfileOpenShift,
		},
	}

	c, err := internal.NewFakeClient(&global)
	require.NoError(t, err)

	err = ConfigureDefaults(context.TODO(), c, &global, false)
	require.NoError(t, err)

	ip := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-camel-k",
			Namespace: "ns",
		},
	}

	ip.ResyncStatusFullConfig()

	applyPlatformSpec(&global, &ip)

	assert.Equal(t, v1.IntegrationPlatformClusterOpenShift, ip.Status.Cluster)
	assert.Equal(t, v1.TraitProfileOpenShift, ip.Status.Profile)
	assert.Equal(t, v1.BuildStrategyRoutine, ip.Status.Build.BuildConfiguration.Strategy)
	assert.Equal(t, v1.BuildOrderStrategyFIFO, ip.Status.Build.BuildConfiguration.OrderStrategy)
	assert.Equal(t, int32(3), ip.Status.Build.MaxRunningBuilds) // default for build strategy routine
	assert.Equal(t, len(global.Status.Build.Maven.CLIOptions), len(ip.Status.Build.Maven.CLIOptions))
	assert.Equal(t, global.Status.Build.Maven.CLIOptions, ip.Status.Build.Maven.CLIOptions)
	assert.NotNil(t, ip.Status.Traits)
	assert.NotNil(t, ip.Status.Traits.Logging)
	assert.Equal(t, "DEBUG", ip.Status.Traits.Logging.Level)
	assert.NotNil(t, ip.Status.Traits.Container)
	assert.Equal(t, corev1.PullAlways, ip.Status.Traits.Container.ImagePullPolicy)
	assert.Equal(t, "0.1", ip.Status.Traits.Container.LimitCPU)
	assert.Len(t, ip.Status.Build.Maven.Properties, 2)
	assert.Equal(t, "global_value1", ip.Status.Build.Maven.Properties["global_prop1"])
	assert.Equal(t, "global_value2", ip.Status.Build.Maven.Properties["global_prop2"])
}

func TestPlatformS2IhUpdateOverrideLocalPlatformSpec(t *testing.T) {
	global := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
		},
		Spec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
			},
		},
	}

	c, err := internal.NewFakeClient(&global)
	require.NoError(t, err)

	err = ConfigureDefaults(context.TODO(), c, &global, false)
	require.NoError(t, err)

	ip := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-camel-k",
			Namespace: "ns",
		},
		Spec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
				BaseImage:       "overridden",
			},
		},
	}

	ip.ResyncStatusFullConfig()

	applyPlatformSpec(&global, &ip)

	assert.Equal(t, v1.IntegrationPlatformBuildPublishStrategyS2I, ip.Status.Build.PublishStrategy)
	assert.Equal(t, "overridden", ip.Status.Build.BaseImage)
}

func TestPlatformS2IUpdateDefaultLocalPlatformSpec(t *testing.T) {
	global := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
		},
		Spec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
				BaseImage:       "overridden",
			},
		},
	}

	c, err := internal.NewFakeClient(&global)
	require.NoError(t, err)

	err = ConfigureDefaults(context.TODO(), c, &global, false)
	require.NoError(t, err)
	assert.Equal(t, "overridden", global.Status.Build.BaseImage)

	ip := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-camel-k",
			Namespace: "ns",
		},
		Spec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
			},
		},
	}

	ip.ResyncStatusFullConfig()

	applyPlatformSpec(&global, &ip)

	assert.Equal(t, v1.IntegrationPlatformBuildPublishStrategyS2I, ip.Status.Build.PublishStrategy)
}

func TestRetainLocalPlatformSpec(t *testing.T) {
	global := v1.IntegrationPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultPlatformName,
			Namespace: "ns",
		},
		Spec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				BuildConfiguration: v1.BuildConfiguration{
					Strategy:      v1.BuildStrategyRoutine,
					OrderStrategy: v1.BuildOrderStrategySequential,
				},
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
			Cluster: v1.IntegrationPlatformClusterOpenShift,
			Profile: v1.TraitProfileOpenShift,
		},
	}

	c, err := internal.NewFakeClient(&global)
	require.NoError(t, err)

	err = ConfigureDefaults(context.TODO(), c, &global, false)
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

	applyPlatformSpec(&global, &ip)

	assert.Equal(t, v1.IntegrationPlatformClusterKubernetes, ip.Status.Cluster)
	assert.Equal(t, v1.TraitProfileKnative, ip.Status.Profile)
	assert.Equal(t, v1.BuildStrategyPod, ip.Status.Build.BuildConfiguration.Strategy)
	assert.Equal(t, v1.BuildOrderStrategyFIFO, ip.Status.Build.BuildConfiguration.OrderStrategy)
	assert.Equal(t, int32(1), ip.Status.Build.MaxRunningBuilds)
	assert.Equal(t, len(global.Status.Build.Maven.CLIOptions), len(ip.Status.Build.Maven.CLIOptions))
	assert.Equal(t, global.Status.Build.Maven.CLIOptions, ip.Status.Build.Maven.CLIOptions)
	assert.NotNil(t, ip.Status.Traits)
	assert.NotNil(t, ip.Status.Traits.Logging)
	assert.Equal(t, "DEBUG", ip.Status.Traits.Logging.Level)
	assert.NotNil(t, ip.Status.Traits.Container)
	assert.Equal(t, corev1.PullAlways, ip.Status.Traits.Container.ImagePullPolicy)
	assert.Equal(t, "0.1", ip.Status.Traits.Container.LimitCPU)
	assert.Len(t, ip.Status.Build.Maven.Properties, 3)
	assert.Equal(t, "global_value1", ip.Status.Build.Maven.Properties["global_prop1"])
	assert.Equal(t, "local_value2", ip.Status.Build.Maven.Properties["global_prop2"])
	assert.Equal(t, "local_value1", ip.Status.Build.Maven.Properties["local_prop1"])
}
