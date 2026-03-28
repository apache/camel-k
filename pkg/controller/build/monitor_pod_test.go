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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestHandle_PodSucceeded(t *testing.T) {
	now := time.Now()
	start := metav1.NewTime(now.Add(-2 * time.Minute))
	end := metav1.NewTime(now.Add(2 * time.Minute))

	build := &v1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-build",
			Namespace: "default",
		},
		Spec: v1.BuildSpec{
			Timeout: metav1.Duration{Duration: 5 * time.Minute},
			Tasks: []v1.Task{
				{
					Custom: &v1.UserTask{
						BaseTask: v1.BaseTask{
							Configuration: v1.BuildConfiguration{
								BuilderPodNamespace: "default",
							},
						},
						PublishingImage: "my-camel-k-container-image",
					},
				},
			},
		},
		Status: v1.BuildStatus{
			Phase:     v1.BuildPhaseRunning,
			StartedAt: &start,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        buildPodName(build),
			Namespace:   build.BuilderPodNamespace(),
			Annotations: map[string]string{},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode:   0,
							FinishedAt: end,
							Message:    "sha:123456",
						},
					},
				},
			},
		},
	}

	c, err := internal.NewFakeClient(build, pod)
	require.NoError(t, err)

	a := monitorPodAction{
		reader: c,
	}
	a.InjectLogger(log.Log)
	a.InjectClient(c)

	result, err := a.Handle(context.Background(), build)

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, v1.BuildPhaseSucceeded, result.Status.Phase)
	assert.Equal(t, "sha:123456", result.Status.Digest)
	assert.Equal(t, "my-camel-k-container-image", result.Status.Image)
}

func TestHandle_PodFailed(t *testing.T) {
	now := time.Now()
	start := metav1.NewTime(now.Add(-2 * time.Minute))

	build := &v1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-build",
			Namespace: "default",
		},
		Spec: v1.BuildSpec{
			Timeout: metav1.Duration{Duration: 5 * time.Minute},
			Tasks: []v1.Task{
				{
					Custom: &v1.UserTask{
						BaseTask: v1.BaseTask{
							Configuration: v1.BuildConfiguration{
								BuilderPodNamespace: "default",
							},
						},
						PublishingImage: "my-camel-k-container-image",
					},
				},
			},
		},
		Status: v1.BuildStatus{
			Phase:     v1.BuildPhaseRunning,
			StartedAt: &start,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        buildPodName(build),
			Namespace:   build.BuilderPodNamespace(),
			Annotations: map[string]string{},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
		},
	}

	c, err := internal.NewFakeClient(build, pod)
	require.NoError(t, err)

	a := monitorPodAction{
		reader: c,
	}
	a.InjectLogger(log.Log)
	a.InjectClient(c)

	result, err := a.Handle(context.Background(), build)

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, v1.BuildPhaseFailed, result.Status.Phase)
	assert.Equal(t, "Builder Pod camel-k-test-build-builder failed (see conditions for more details)", result.Status.Error)
}
