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
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

const enqueuedMsg = "%s - the build (%s) gets enqueued"

var runningBuilds sync.Map

type Monitor struct {
	maxRunningBuilds   int32
	buildOrderStrategy v1.BuildOrderStrategy
}

func (bm *Monitor) canSchedule(ctx context.Context, c ctrl.Reader, build *v1.Build) (bool, *v1.BuildCondition, error) {

	var runningBuildsTotal int32
	runningBuilds.Range(func(_, v interface{}) bool {
		runningBuildsTotal++
		return true
	})

	requestName := build.Name
	requestNamespace := build.Namespace
	buildCreator := kubernetes.GetCamelCreator(build)
	if buildCreator != nil {
		requestName = buildCreator.Name
		requestNamespace = buildCreator.Namespace
	}

	if runningBuildsTotal >= bm.maxRunningBuilds {
		reason := fmt.Sprintf(
			"Maximum number of running builds (%d) exceeded",
			runningBuildsTotal,
		)
		Log.WithValues("request-namespace", requestNamespace, "request-name", requestName, "max-running-builds-limit", runningBuildsTotal).
			ForBuild(build).Infof(enqueuedMsg, reason, build.Name)
		// max number of running builds limit exceeded
		return false, scheduledWaitingBuildcondition(build.Name, reason), nil
	}

	layout := build.Labels[v1.IntegrationKitLayoutLabel]

	// Native builds can be run in parallel, as incremental images is not applicable.
	if layout == v1.IntegrationKitLayoutNativeSources {
		return true, scheduledReadyBuildcondition(build.Name), nil
	}

	// We assume incremental images is only applicable across images whose layout is identical
	withCompatibleLayout, err := labels.NewRequirement(v1.IntegrationKitLayoutLabel, selection.Equals, []string{layout})
	if err != nil {
		return false, nil, err
	}

	builds := &v1.BuildList{}
	// We use the non-caching client as informers cache is not invalidated nor updated
	// atomically by write operations
	err = c.List(ctx, builds,
		ctrl.InNamespace(build.Namespace),
		ctrl.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(*withCompatibleLayout),
		})
	if err != nil {
		return false, nil, err
	}

	var reason string
	allowed := true
	condition := scheduledReadyBuildcondition(build.Name)
	switch bm.buildOrderStrategy {
	case v1.BuildOrderStrategyFIFO:
		// Check on builds that have been created before the current build and grant precedence if any.
		if hasScheduledBuildsBefore, otherBuild := builds.HasScheduledBuildsBefore(build); hasScheduledBuildsBefore {
			reason = fmt.Sprintf("Waiting for build (%s) because it has been created before", otherBuild.Name)
			allowed = false
		}
	case v1.BuildOrderStrategyDependencies:
		// Check on the Integration dependencies and see if we should queue the build in order to leverage incremental builds
		// because there is already another build in the making that matches the requirements
		if hasMatchingBuild, otherBuild := builds.HasMatchingBuild(build); hasMatchingBuild {
			reason = fmt.Sprintf("Waiting for build (%s) to finish in order to use incremental image builds", otherBuild.Name)
			allowed = false
		}
	case v1.BuildOrderStrategySequential:
		// Emulate a serialized working queue to only allow one build to run at a given time.
		// Let's requeue the build in case one is already running
		if hasRunningBuilds := builds.HasRunningBuilds(); hasRunningBuilds {
			reason = "Found a running build in this namespace"
			allowed = false
		}
	default:
		reason = fmt.Sprintf("Unsupported build order strategy (%s)", bm.buildOrderStrategy)
		allowed = false
	}

	if !allowed {
		Log.WithValues("request-namespace", requestNamespace, "request-name", requestName, "order-strategy", bm.buildOrderStrategy).
			ForBuild(build).Infof(enqueuedMsg, reason, build.Name)
		condition = scheduledWaitingBuildcondition(build.Name, reason)
	}

	return allowed, condition, nil
}

func monitorRunningBuild(build *v1.Build) {
	runningBuilds.Store(types.NamespacedName{Namespace: build.Namespace, Name: build.Name}.String(), true)
}

func monitorFinishedBuild(build *v1.Build) {
	runningBuilds.Delete(types.NamespacedName{Namespace: build.Namespace, Name: build.Name}.String())
}

func scheduledReadyBuildcondition(buildName string) *v1.BuildCondition {
	return scheduledBuildcondition(corev1.ConditionTrue, v1.BuildConditionReadyReason, fmt.Sprintf(
		"the build (%s) is scheduled",
		buildName,
	))
}

func scheduledWaitingBuildcondition(buildName string, reason string) *v1.BuildCondition {
	return scheduledBuildcondition(corev1.ConditionFalse, v1.BuildConditionWaitingReason, fmt.Sprintf(
		enqueuedMsg,
		reason,
		buildName,
	))
}

func scheduledBuildcondition(status corev1.ConditionStatus, reason string, msg string) *v1.BuildCondition {
	return &v1.BuildCondition{
		Type:    v1.BuildConditionScheduled,
		Status:  status,
		Reason:  reason,
		Message: msg,
	}
}
