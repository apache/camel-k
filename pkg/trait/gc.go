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
	"strconv"


	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

type garbageCollectorTrait struct {
	BaseTrait `property:",squash"`
}

func newGarbageCollectorTrait() *garbageCollectorTrait {
	return &garbageCollectorTrait{
		BaseTrait: newBaseTrait("gc"),
	}
}

func (t *garbageCollectorTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return e.IntegrationInPhase(v1alpha1.IntegrationPhaseInitial) ||
		e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
}

func (t *garbageCollectorTrait) Apply(e *Environment) error {
	// Register a post processor that adds the required labels to the new resources
	e.PostProcessors = append(e.PostProcessors, func(env *Environment) error {
		env.Resources.VisitMetaObject(func(resource metav1.Object) {
			labels := resource.GetLabels()
			if labels == nil {
				labels = map[string]string{}
			}
			// Label the resource with the current integration generation
			labels["camel.apache.org/generation"] = strconv.FormatInt(env.Integration.GetGeneration(), 10)
			// Make sure the integration label is set
			labels["camel.apache.org/integration"] = env.Integration.Name
			resource.SetLabels(labels)
		})
		return nil
	})

	// Let's run garbage collection during the integration deploying phase
	if !e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		return nil
	}
	// Register a post action that deletes the existing resources that are labelled
	// with the previous integration generations.
	// The collection and deletion are performed asynchronously to avoid blocking
	// the reconcile loop.
	e.PostActions = append(e.PostActions, func(environment *Environment) error {
		go t.garbageCollectResources(e)
		return nil
	})

	return nil
}

func (t *garbageCollectorTrait) garbageCollectResources(e *Environment) {
	// Retrieve older generation resources that may can enlisted for garbage collection
	// We rely on the discovery API to retrieve all the resources group and kind.
	// That results in an unbounded collection that can be a bit slow.
	// We may want to refine that step by white-listing or enlisting types to speed-up
	// the collection duration.

	selectors := []string{
		fmt.Sprintf("camel.apache.org/integration=%s", e.Integration.Name),
		"camel.apache.org/generation",
	}
	resources, err := kubernetes.LookUpResources(context.TODO(), e.Client, e.Integration.Namespace, selectors)
	if err != nil {
		t.L.ForIntegration(e.Integration).Errorf(err, "cannot collect older generation resources")
		return
	}

	// And delete them
	for _, resource := range resources {
		// pin the resource
		resource := resource

		labels := resource.GetLabels()
		generation, err := strconv.ParseInt(labels["camel.apache.org/generation"], 10, 64)
		if err != nil {
			t.L.ForIntegration(e.Integration).Errorf(err, "cannot parse generation label: %s", labels["camel.apache.org/generation"])
		}

		// Garbage collect older generation resource only.
		// By the time async garbage collecting is executed, newer generations may exist.
		if generation >= e.Integration.GetGeneration() {
			continue
		}

		err = e.Client.Delete(context.TODO(), &resource, client.PropagationPolicy(metav1.DeletePropagationBackground))
		if err != nil {
			// The resource may have already been deleted
			if !k8serrors.IsNotFound(err) {
				t.L.ForIntegration(e.Integration).Errorf(err, "cannot delete child resource: %s/%s", resource.GetKind(), resource.GetName())
			}
		} else {
			t.L.ForIntegration(e.Integration).Debugf("child resource deleted: %s/%s", resource.GetKind(), resource.GetName())
		}
	}
}
