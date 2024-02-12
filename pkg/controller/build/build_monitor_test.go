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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/test"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestMonitorSequentialBuilds(t *testing.T) {
	testcases := []struct {
		name      string
		running   []*v1.Build
		finished  []*v1.Build
		build     *v1.Build
		allowed   bool
		condition *v1.BuildCondition
	}{
		{
			name:      "allowNewBuild",
			running:   []*v1.Build{},
			finished:  []*v1.Build{},
			build:     newBuild("ns", "my-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name:      "allowNewNativeBuild",
			running:   []*v1.Build{},
			finished:  []*v1.Build{},
			build:     newNativeBuild("ns", "my-native-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-native-build) is scheduled"),
		},
		{
			name:    "allowNewBuildWhenOthersFinished",
			running: []*v1.Build{},
			finished: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
				newBuildInPhase("ns", "my-build-failed", v1.BuildPhaseFailed),
			},
			build:     newBuild("ns", "my-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name:    "allowNewNativeBuildWhenOthersFinished",
			running: []*v1.Build{},
			finished: []*v1.Build{
				newNativeBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
				newNativeBuildInPhase("ns", "my-build-failed", v1.BuildPhaseFailed),
			},
			build:     newNativeBuild("ns", "my-native-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-native-build) is scheduled"),
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
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Maximum number of running builds (3) exceeded - the build (my-build) gets enqueued"),
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
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Maximum number of running builds (3) exceeded - the build (my-build) gets enqueued"),
		},
		{
			name: "allowParallelBuildsWithDifferentLayout",
			running: []*v1.Build{
				newNativeBuildInPhase("ns", "my-build-1", v1.BuildPhaseRunning),
			},
			build:     newBuild("ns", "my-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
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
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Found a running build in this namespace - the build (my-build) gets enqueued"),
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
			build:     newBuild("ns", "my-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
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
				maxRunningBuilds:   3,
				buildOrderStrategy: v1.BuildOrderStrategySequential,
			}

			// reset running builds in memory cache
			cleanRunningBuildsMonitor()
			for _, build := range tc.running {
				monitorRunningBuild(build)
			}

			allowed, condition, err := bm.canSchedule(context.TODO(), c, tc.build)

			assert.Nil(t, err)
			assert.Equal(t, tc.allowed, allowed)
			assert.Equal(t, tc.condition.Type, condition.Type)
			assert.Equal(t, tc.condition.Status, condition.Status)
			assert.Equal(t, tc.condition.Reason, condition.Reason)
			assert.Equal(t, tc.condition.Message, condition.Message)
		})
	}
}

func TestAllowBuildRequeue(t *testing.T) {
	c, err := test.NewFakeClient()

	assert.Nil(t, err)

	bm := Monitor{
		maxRunningBuilds:   3,
		buildOrderStrategy: v1.BuildOrderStrategySequential,
	}

	runningBuild := newBuild("some-ns", "my-build-1")
	// reset running builds in memory cache
	cleanRunningBuildsMonitor()
	monitorRunningBuild(runningBuild)
	monitorRunningBuild(newBuild("other-ns", "my-build-2"))
	monitorRunningBuild(newBuild("another-ns", "my-build-3"))

	build := newBuild("ns", "my-build")
	allowed, condition, err := bm.canSchedule(context.TODO(), c, build)

	assert.Nil(t, err)
	assert.False(t, allowed)
	assert.Equal(t, corev1.ConditionFalse, condition.Status)

	monitorFinishedBuild(runningBuild)

	allowed, condition, err = bm.canSchedule(context.TODO(), c, build)

	assert.Nil(t, err)
	assert.True(t, allowed)
	assert.Equal(t, corev1.ConditionTrue, condition.Status)
}

func TestMonitorFIFOBuilds(t *testing.T) {
	testcases := []struct {
		name      string
		running   []*v1.Build
		builds    []*v1.Build
		build     *v1.Build
		allowed   bool
		condition *v1.BuildCondition
	}{
		{
			name:      "allowNewBuild",
			running:   []*v1.Build{},
			builds:    []*v1.Build{},
			build:     newBuild("ns", "my-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name:      "allowNewNativeBuild",
			running:   []*v1.Build{},
			builds:    []*v1.Build{},
			build:     newNativeBuild("ns1", "my-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name:    "allowNewBuildWhenOthersFinished",
			running: []*v1.Build{},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
				newBuildInPhase("ns", "my-build-failed", v1.BuildPhaseFailed),
			},
			build:     newBuild("ns", "my-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name:    "allowNewNativeBuildWhenOthersFinished",
			running: []*v1.Build{},
			builds: []*v1.Build{
				newNativeBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
				newNativeBuildInPhase("ns", "my-build-failed", v1.BuildPhaseFailed),
			},
			build:     newNativeBuild("ns", "my-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name: "limitMaxRunningBuilds",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1"),
				newBuild("other-ns", "my-build-2"),
				newBuild("another-ns", "my-build-3"),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
			},
			build:   newBuild("ns", "my-build"),
			allowed: false,
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Maximum number of running builds (3) exceeded - the build (my-build) gets enqueued"),
		},
		{
			name: "limitMaxRunningNativeBuilds",
			running: []*v1.Build{
				newBuildInPhase("some-ns", "my-build-1", v1.BuildPhaseRunning),
				newNativeBuildInPhase("other-ns", "my-build-2", v1.BuildPhaseRunning),
				newNativeBuildInPhase("another-ns", "my-build-3", v1.BuildPhaseRunning),
			},
			builds: []*v1.Build{
				newNativeBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
			},
			build:   newNativeBuildInPhase("ns", "my-build", v1.BuildPhaseInitialization),
			allowed: false,
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Maximum number of running builds (3) exceeded - the build (my-build) gets enqueued"),
		},
		{
			name: "allowParallelBuildsWithDifferentLayout",
			running: []*v1.Build{
				newNativeBuildInPhase("ns", "my-build-1", v1.BuildPhaseRunning),
			},
			build:     newBuild("ns", "my-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name: "allowParallelBuildsInSameNamespace",
			running: []*v1.Build{
				newBuild("ns", "my-build-1"),
				newBuild("other-ns", "my-build-2"),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
				newBuildInPhase("other-ns", "my-build-new", v1.BuildPhaseScheduling),
			},
			build:     newBuild("ns", "my-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name: "queueBuildWhenOlderBuildIsAlreadyInitialized",
			running: []*v1.Build{
				newBuild("ns", "my-build-1"),
				newBuild("other-ns", "my-build-2"),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
				newBuildInPhase("ns", "my-build-new", v1.BuildPhaseInitialization),
			},
			build:   newBuild("ns", "my-build"),
			allowed: false,
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Waiting for build (my-build-new) because it has been created before - the build (my-build) gets enqueued"),
		},
		{
			name: "queueBuildWhenOlderBuildIsAlreadyScheduled",
			running: []*v1.Build{
				newBuild("ns", "my-build-1"),
				newBuild("other-ns", "my-build-2"),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
				newBuildInPhase("ns", "my-build-new", v1.BuildPhaseScheduling),
			},
			build:   newBuild("ns", "my-build"),
			allowed: false,
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Waiting for build (my-build-new) because it has been created before - the build (my-build) gets enqueued"),
		},
		{
			name: "allowBuildsInNewNamespace",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1"),
				newBuild("other-ns", "my-build-2"),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded),
			},
			build:     newBuild("ns", "my-build"),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var initObjs []runtime.Object
			for _, build := range append(tc.running, tc.builds...) {
				initObjs = append(initObjs, build)
			}

			c, err := test.NewFakeClient(initObjs...)

			assert.Nil(t, err)

			bm := Monitor{
				maxRunningBuilds:   3,
				buildOrderStrategy: v1.BuildOrderStrategyFIFO,
			}

			// reset running builds in memory cache
			cleanRunningBuildsMonitor()
			for _, build := range tc.running {
				monitorRunningBuild(build)
			}

			allowed, condition, err := bm.canSchedule(context.TODO(), c, tc.build)

			assert.Nil(t, err)
			assert.Equal(t, tc.allowed, allowed)
			assert.Equal(t, tc.condition.Type, condition.Type)
			assert.Equal(t, tc.condition.Status, condition.Status)
			assert.Equal(t, tc.condition.Reason, condition.Reason)
			assert.Equal(t, tc.condition.Message, condition.Message)
		})
	}
}

func TestMonitorDependencyMatchingBuilds(t *testing.T) {
	deps := []string{"camel:core", "camel:timer", "camel:log", "mvn:org.apache.camel.k:camel-k-runtime"}

	testcases := []struct {
		name      string
		running   []*v1.Build
		builds    []*v1.Build
		build     *v1.Build
		allowed   bool
		condition *v1.BuildCondition
	}{
		{
			name:      "allowNewBuild",
			running:   []*v1.Build{},
			builds:    []*v1.Build{},
			build:     newBuild("ns", "my-build", deps...),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name:      "allowNewNativeBuild",
			running:   []*v1.Build{},
			builds:    []*v1.Build{},
			build:     newNativeBuild("ns1", "my-build", deps...),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name:    "allowNewBuildWhenOthersFinished",
			running: []*v1.Build{},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
				newBuildInPhase("ns", "my-build-failed", v1.BuildPhaseFailed, deps...),
			},
			build:     newBuild("ns", "my-build", deps...),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name:    "allowNewNativeBuildWhenOthersFinished",
			running: []*v1.Build{},
			builds: []*v1.Build{
				newNativeBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
				newNativeBuildInPhase("ns", "my-build-failed", v1.BuildPhaseFailed, deps...),
			},
			build:     newNativeBuild("ns", "my-build", deps...),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name: "limitMaxRunningBuilds",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1", deps...),
				newBuild("other-ns", "my-build-2", deps...),
				newBuild("another-ns", "my-build-3", deps...),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
			},
			build:   newBuild("ns", "my-build", deps...),
			allowed: false,
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Maximum number of running builds (3) exceeded - the build (my-build) gets enqueued"),
		},
		{
			name: "limitMaxRunningNativeBuilds",
			running: []*v1.Build{
				newBuildInPhase("some-ns", "my-build-1", v1.BuildPhaseRunning, deps...),
				newNativeBuildInPhase("other-ns", "my-build-2", v1.BuildPhaseRunning, deps...),
				newNativeBuildInPhase("another-ns", "my-build-3", v1.BuildPhaseRunning, deps...),
			},
			builds: []*v1.Build{
				newNativeBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
			},
			build:   newNativeBuildInPhase("ns", "my-build", v1.BuildPhaseInitialization, deps...),
			allowed: false,
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Maximum number of running builds (3) exceeded - the build (my-build) gets enqueued"),
		},
		{
			name: "allowParallelBuildsWithDifferentLayout",
			running: []*v1.Build{
				newNativeBuildInPhase("ns", "my-build-1", v1.BuildPhaseRunning, deps...),
			},
			build:     newBuild("ns", "my-build", deps...),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name: "allowParallelBuildsInSameNamespaceWithTooManyMissingDependencies",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1", deps...),
				newBuild("other-ns", "my-build-2", deps...),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
				newBuildInPhase("other-ns", "my-build-new", v1.BuildPhaseScheduling, deps...),
			},
			build:     newBuild("ns", "my-build", append(deps, "camel:test", "camel:foo")...),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name: "allowParallelBuildsInSameNamespaceWithLessDependencies",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1", deps...),
				newBuild("other-ns", "my-build-2", deps...),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
				newBuildInPhase("ns", "my-build-scheduled", v1.BuildPhaseScheduling, append(deps, "camel:test")...),
			},
			build:     newBuild("ns", "my-build", deps...),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
		{
			name: "queueBuildWhenSuitableBuildHasOnlyOneSingleMissingDependency",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1", deps...),
				newBuild("other-ns", "my-build-2", deps...),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
				newBuildInPhase("ns", "my-build-new", v1.BuildPhaseScheduling, deps...),
			},
			build:   newBuild("ns", "my-build", append(deps, "camel:test")...),
			allowed: false,
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Waiting for build (my-build-new) to finish in order to use incremental image builds - the build (my-build) gets enqueued"),
		},
		{
			name: "queueBuildWhenSuitableBuildIsAlreadyRunning",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1", deps...),
				newBuild("other-ns", "my-build-2", deps...),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
				newBuildInPhase("ns", "my-build-running", v1.BuildPhaseRunning, deps...),
			},
			build:   newBuild("ns", "my-build", deps...),
			allowed: false,
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Waiting for build (my-build-running) to finish in order to use incremental image builds - the build (my-build) gets enqueued"),
		},
		{
			name: "queueBuildWhenSuitableBuildIsAlreadyPending",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1", deps...),
				newBuild("other-ns", "my-build-2", deps...),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
				newBuildInPhase("ns", "my-build-pending", v1.BuildPhasePending, deps...),
			},
			build:   newBuild("ns", "my-build", deps...),
			allowed: false,
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Waiting for build (my-build-pending) to finish in order to use incremental image builds - the build (my-build) gets enqueued"),
		},
		{
			name: "queueBuildWhenSuitableBuildWithSupersetDependenciesIsAlreadyRunning",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1", deps...),
				newBuild("other-ns", "my-build-2", deps...),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
				newBuildInPhase("ns", "my-build-running", v1.BuildPhaseRunning, append(deps, "camel:test")...),
			},
			build:   newBuild("ns", "my-build", deps...),
			allowed: false,
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Waiting for build (my-build-running) to finish in order to use incremental image builds - the build (my-build) gets enqueued"),
		},
		{
			name: "queueBuildWhenSuitableBuildWithSameDependenciesIsAlreadyScheduled",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1", deps...),
				newBuild("other-ns", "my-build-2", deps...),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
				newBuildInPhase("ns", "my-build-existing", v1.BuildPhaseScheduling, deps...),
			},
			build:   newBuild("ns", "my-build", deps...),
			allowed: false,
			condition: newCondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason,
				"Waiting for build (my-build-existing) to finish in order to use incremental image builds - the build (my-build) gets enqueued"),
		},
		{
			name: "allowBuildsInNewNamespace",
			running: []*v1.Build{
				newBuild("some-ns", "my-build-1", deps...),
				newBuild("other-ns", "my-build-2", deps...),
			},
			builds: []*v1.Build{
				newBuildInPhase("ns", "my-build-x", v1.BuildPhaseSucceeded, deps...),
			},
			build:     newBuild("ns", "my-build", deps...),
			allowed:   true,
			condition: newCondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, "the build (my-build) is scheduled"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var initObjs []runtime.Object
			for _, build := range append(tc.running, tc.builds...) {
				initObjs = append(initObjs, build)
			}

			c, err := test.NewFakeClient(initObjs...)

			assert.Nil(t, err)

			bm := Monitor{
				maxRunningBuilds:   3,
				buildOrderStrategy: v1.BuildOrderStrategyDependencies,
			}

			// reset running builds in memory cache
			cleanRunningBuildsMonitor()
			for _, build := range tc.running {
				monitorRunningBuild(build)
			}

			allowed, condition, err := bm.canSchedule(context.TODO(), c, tc.build)

			assert.Nil(t, err)
			assert.Equal(t, tc.allowed, allowed)
			assert.Equal(t, tc.condition.Type, condition.Type)
			assert.Equal(t, tc.condition.Status, condition.Status)
			assert.Equal(t, tc.condition.Reason, condition.Reason)
			assert.Equal(t, tc.condition.Message, condition.Message)
		})
	}
}

func cleanRunningBuildsMonitor() {
	runningBuilds.Range(func(key interface{}, v interface{}) bool {
		runningBuilds.Delete(key)
		return true
	})
}

func newCondition(status corev1.ConditionStatus, reason string, msg string) *v1.BuildCondition {
	return &v1.BuildCondition{
		Type:    v1.BuildConditionScheduled,
		Status:  status,
		Reason:  reason,
		Message: msg,
	}
}

func newBuild(namespace string, name string, dependencies ...string) *v1.Build {
	return newBuildWithLayoutInPhase(namespace, name, v1.IntegrationKitLayoutFastJar, v1.BuildPhasePending, dependencies...)
}

func newNativeBuild(namespace string, name string, dependencies ...string) *v1.Build {
	return newBuildWithLayoutInPhase(namespace, name, v1.IntegrationKitLayoutNativeSources, v1.BuildPhasePending, dependencies...)
}

func newBuildInPhase(namespace string, name string, phase v1.BuildPhase, dependencies ...string) *v1.Build {
	return newBuildWithLayoutInPhase(namespace, name, v1.IntegrationKitLayoutFastJar, phase, dependencies...)
}

func newNativeBuildInPhase(namespace string, name string, phase v1.BuildPhase, dependencies ...string) *v1.Build {
	return newBuildWithLayoutInPhase(namespace, name, v1.IntegrationKitLayoutNativeSources, phase, dependencies...)
}

func newBuildWithLayoutInPhase(namespace string, name string, layout string, phase v1.BuildPhase, dependencies ...string) *v1.Build {
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
			CreationTimestamp: metav1.NewTime(time.Now()),
		},
		Spec: v1.BuildSpec{
			Tasks: []v1.Task{
				{
					Builder: &v1.BuilderTask{
						BaseTask: v1.BaseTask{
							Configuration: v1.BuildConfiguration{
								Strategy:            v1.BuildStrategyRoutine,
								OrderStrategy:       v1.BuildOrderStrategySequential,
								ToolImage:           "camel:latest",
								BuilderPodNamespace: "ns",
							},
						},
						Dependencies: dependencies,
					},
				},
			},
			Timeout: metav1.Duration{Duration: 5 * time.Minute},
		},
		Status: v1.BuildStatus{
			Phase: phase,
		},
	}
}
