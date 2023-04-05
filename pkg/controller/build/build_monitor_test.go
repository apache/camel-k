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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestMonitorBuilds(t *testing.T) {
	testcases := []struct {
		name     string
		running  []*v1.Build
		finished []*v1.Build
		build    *v1.Build
		allowed  bool
	}{
		{
			name:     "allowNewBuild",
			running:  []*v1.Build{},
			finished: []*v1.Build{},
			build:    newBuild("ns", "my-build"),
			allowed:  true,
		},
		{
			name:     "allowNewNativeBuild",
			running:  []*v1.Build{},
			finished: []*v1.Build{},
			build:    newNativeBuild("ns", "my-build"),
			allowed:  true,
		},
		{
			name:    "allowNewBuildWhenOthersFinished",
			running: []*v1.Build{},
			finished: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
				newBuildInPhase("ns", "my-build-failed", v1.BuildPhaseFailed),
			},
			build:   newBuild("ns", "my-build"),
			allowed: true,
		},
		{
			name:    "allowNewNativeBuildWhenOthersFinished",
			running: []*v1.Build{},
			finished: []*v1.Build{
				newNativeBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
				newNativeBuildInPhase("ns", "my-build-failed", v1.BuildPhaseFailed),
			},
			build:   newNativeBuild("ns", "my-build"),
			allowed: true,
		},
		{
			name: "limitMaxRunningBuilds",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1"),
				newBuild("other-ns", "my-build-2"),
				newBuild("another-ns", "my-build-3"),
			},
			finished: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
			},
			build:   newBuild("ns", "my-build"),
			allowed: false,
		},
		{
			name: "limitMaxRunningNativeBuilds",
			running: []*v1.Build{
				newBuildInPhase("some-ns", "my-build-1", v1.BuildPhaseRunning),
				newNativeBuildInPhase("other-ns", "my-build-2", v1.BuildPhaseRunning),
				newNativeBuildInPhase("another-ns", "my-build-3", v1.BuildPhaseRunning),
			},
			finished: []*v1.Build{
				newNativeBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
			},
			build:   newNativeBuildInPhase("ns", "my-build", v1.BuildPhaseInitialization),
			allowed: false,
		},
		{
			name: "allowParallelBuildsWithDifferentLayout",
			running: []*v1.Build{
				newNativeBuildInPhase("ns", "my-build-1", v1.BuildPhaseRunning),
			},
			build:   newBuild("ns", "my-build"),
			allowed: true,
		},
		{
			name: "queueBuildsInSameNamespaceWithSameLayout",
			running: []*v1.Build{
				newBuild("ns", "my-build-1"),
				newBuild("other-ns", "my-build-2"),
			},
			finished: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
			},
			build:   newBuild("ns", "my-build"),
			allowed: false,
		},
		{
			name: "allowBuildsInNewNamespace",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1"),
				newBuild("other-ns", "my-build-2"),
			},
			finished: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
			},
			build:   newBuild("ns", "my-build"),
			allowed: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var initObjs []runtime.Object
			for _, build := range append(tc.running, tc.finished...) {
				initObjs = append(initObjs, build)
			}

			c, err := test.NewFakeClient(initObjs...)

			assert.Nil(t, err)

			bm := Monitor{
				maxRunningBuilds: 3,
			}

			// reset running builds in memory cache
			cleanRunningBuildsMonitor()
			for _, build := range tc.running {
				monitorRunningBuild(build)
			}

			allowed, err := bm.canSchedule(context.TODO(), c, tc.build)

			assert.Nil(t, err)
			assert.Equal(t, tc.allowed, allowed)
		})
	}
}

func TestAllowBuildRequeue(t *testing.T) {
	c, err := test.NewFakeClient()

	assert.Nil(t, err)

	bm := Monitor{
		maxRunningBuilds: 3,
	}

	runningBuild := newBuild("some-ns", "my-build-1")
	// reset running builds in memory cache
	cleanRunningBuildsMonitor()
	monitorRunningBuild(runningBuild)
	monitorRunningBuild(newBuild("other-ns", "my-build-2"))
	monitorRunningBuild(newBuild("another-ns", "my-build-3"))

	build := newBuild("ns", "my-build")
	allowed, err := bm.canSchedule(context.TODO(), c, build)

	assert.Nil(t, err)
	assert.False(t, allowed)

	monitorFinishedBuild(runningBuild)

	allowed, err = bm.canSchedule(context.TODO(), c, build)

	assert.Nil(t, err)
	assert.True(t, allowed)
}

func cleanRunningBuildsMonitor() {
	runningBuilds.Range(func(key interface{}, v interface{}) bool {
		runningBuilds.Delete(key)
		return true
	})
}

func newBuild(namespace string, name string) *v1.Build {
	return newBuildWithLayoutInPhase(namespace, name, v1.IntegrationKitLayoutFastJar, v1.BuildPhasePending)
}

func newNativeBuild(namespace string, name string) *v1.Build {
	return newBuildWithLayoutInPhase(namespace, name, v1.IntegrationKitLayoutNative, v1.BuildPhasePending)
}

func newBuildInPhase(namespace string, name string, phase v1.BuildPhase) *v1.Build {
	return newBuildWithLayoutInPhase(namespace, name, v1.IntegrationKitLayoutFastJar, phase)
}

func newNativeBuildInPhase(namespace string, name string, phase v1.BuildPhase) *v1.Build {
	return newBuildWithLayoutInPhase(namespace, name, v1.IntegrationKitLayoutNative, phase)
}

func newBuildWithLayoutInPhase(namespace string, name string, layout string, phase v1.BuildPhase) *v1.Build {
	return &v1.Build{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.BuildKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				v1.IntegrationKitLayoutLabel: layout,
			},
		},
		Spec: v1.BuildSpec{
			Strategy:         v1.BuildStrategyRoutine,
			Tasks:            []v1.Task{},
			Timeout:          metav1.Duration{Duration: 5 * time.Minute},
			MaxRunningBuilds: 3,
		},
		Status: v1.BuildStatus{
			Phase: phase,
		},
	}
}
