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

package build

import (
	"context"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewBuildPodConfiguration(t *testing.T) {

	ctx := context.TODO()
	c, err := internal.NewFakeClient()
	require.NoError(t, err)

	build := v1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: "theBuildName",
		},
		Spec: v1.BuildSpec{
			Tasks: []v1.Task{
				{
					Builder: &v1.BuilderTask{
						BaseTask: v1.BaseTask{
							Name: "builder",
							Configuration: v1.BuildConfiguration{
								BuilderPodNamespace: "theNamespace",
								NodeSelector:        map[string]string{"node": "selector"},
								Annotations:         map[string]string{"annotation": "value"},
							},
						},
					},
				},
			},
		},
	}

	pod := newBuildPod(ctx, c, &build)

	assert.Equal(t, "Pod", pod.Kind)
	assert.Equal(t, "theNamespace", pod.Namespace)
	assert.Equal(t, map[string]string{
		"camel.apache.org/build":     "theBuildName",
		"camel.apache.org/component": "builder",
	}, pod.Labels)
	assert.Equal(t, map[string]string{"node": "selector"}, pod.Spec.NodeSelector)
	assert.Equal(t, map[string]string{"annotation": "value"}, pod.Annotations)
}

func TestConfigureResourcesDefault(t *testing.T) {
	build := v1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: "theBuildName",
		},
		Spec: v1.BuildSpec{
			Tasks: []v1.Task{
				{
					Builder: &v1.BuilderTask{
						BaseTask: v1.BaseTask{
							Name:          "builder",
							Configuration: v1.BuildConfiguration{},
						},
					},
				},
			},
		},
	}

	container := corev1.Container{}
	configureResources("builder", &build, &container, "250m", "500m", "512Mi", "1Gi")
	configureTaskResources("builder", &build, &container)

	assert.Equal(t, "250m", container.Resources.Requests.Cpu().String())
	assert.Equal(t, "500m", container.Resources.Limits.Cpu().String())
	assert.Equal(t, "512Mi", container.Resources.Requests.Memory().String())
	assert.Equal(t, "1Gi", container.Resources.Limits.Memory().String())
}

func TestConfigureResources(t *testing.T) {
	build := v1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: "theBuildName",
		},
		Spec: v1.BuildSpec{
			Tasks: []v1.Task{
				{
					Builder: &v1.BuilderTask{
						BaseTask: v1.BaseTask{
							Name: "builder",
							Configuration: v1.BuildConfiguration{
								RequestCPU:    "500m",
								LimitCPU:      "1000m",
								RequestMemory: "512Mi",
								LimitMemory:   "2048Mi",
							},
						},
					},
				},
			},
		},
	}
	container := corev1.Container{}
	configureTaskResources("builder", &build, &container)

	assert.Equal(t, "500m", container.Resources.Requests.Cpu().String())
	assert.Equal(t, "1", container.Resources.Limits.Cpu().String())
	assert.Equal(t, "512Mi", container.Resources.Requests.Memory().String())
	assert.Equal(t, "2Gi", container.Resources.Limits.Memory().String())
}

func TestConfigureResourcesOverride(t *testing.T) {
	build := v1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name: "theBuildName",
		},
		Spec: v1.BuildSpec{
			Tasks: []v1.Task{
				{
					Builder: &v1.BuilderTask{
						BaseTask: v1.BaseTask{
							Name: "builder",
							Configuration: v1.BuildConfiguration{
								RequestCPU:    "500m",
								LimitCPU:      "1000m",
								RequestMemory: "512Mi",
								LimitMemory:   "2048Mi",
							},
						},
					},
				},
			},
		},
	}
	container := corev1.Container{}
	configureResources("builder", &build, &container, "10m", "50m", "100Mi", "200Mi")
	configureTaskResources("builder", &build, &container)

	assert.Equal(t, "500m", container.Resources.Requests.Cpu().String())
	assert.Equal(t, "1", container.Resources.Limits.Cpu().String())
	assert.Equal(t, "512Mi", container.Resources.Requests.Memory().String())
	assert.Equal(t, "2Gi", container.Resources.Limits.Memory().String())
}
