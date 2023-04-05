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
	"sync"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

var runningBuilds sync.Map

type Monitor struct {
	maxRunningBuilds int32
}

func (bm *Monitor) canSchedule(ctx context.Context, c ctrl.Reader, build *v1.Build) (bool, error) {
	var runningBuildsTotal int32
	runningBuilds.Range(func(_, v interface{}) bool {
		runningBuildsTotal++
		return true
	})

	if runningBuildsTotal >= bm.maxRunningBuilds {
		requestName := build.Name
		requestNamespace := build.Namespace
		buildCreator := kubernetes.GetCamelCreator(build)
		if buildCreator != nil {
			requestName = buildCreator.Name
			requestNamespace = buildCreator.Namespace
		}

		Log.WithValues("request-namespace", requestNamespace, "request-name", requestName, "max-running-builds-limit", runningBuildsTotal).
			ForBuild(build).Infof("Maximum number of running builds (%d) exceeded - the build gets enqueued", runningBuildsTotal)

		// max number of running builds limit exceeded
		return false, nil
	}

	layout := build.Labels[v1.IntegrationKitLayoutLabel]

	// Native builds can be run in parallel, as incremental images is not applicable.
	if layout == v1.IntegrationKitLayoutNative {
		return true, nil
	}

	// We assume incremental images is only applicable across images whose layout is identical
	withCompatibleLayout, err := labels.NewRequirement(v1.IntegrationKitLayoutLabel, selection.Equals, []string{layout})
	if err != nil {
		return false, err
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
		return false, err
	}

	// Emulate a serialized working queue to only allow one build to run at a given time.
	// This is currently necessary for the incremental build to work as expected.
	// We may want to explicitly manage build priority as opposed to relying on
	// the reconciliation loop to handle the queuing.
	for _, b := range builds.Items {
		if b.Status.Phase == v1.BuildPhasePending || b.Status.Phase == v1.BuildPhaseRunning {
			// Let's requeue the build in case one is already running
			return false, nil
		}
	}

	return true, nil
}

func monitorRunningBuild(build *v1.Build) {
	runningBuilds.Store(types.NamespacedName{Namespace: build.Namespace, Name: build.Name}.String(), true)
}

func monitorFinishedBuild(build *v1.Build) {
	runningBuilds.Delete(types.NamespacedName{Namespace: build.Namespace, Name: build.Name}.String())
}
