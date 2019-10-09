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
	"strconv"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"

	controller "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util"
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

	return e.IntegrationInPhase(
			v1alpha1.IntegrationPhaseInitialization,
			v1alpha1.IntegrationPhaseDeploying,
			v1alpha1.IntegrationPhaseRunning),
		nil
}

func (t *garbageCollectorTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1alpha1.IntegrationPhaseInitialization, v1alpha1.IntegrationPhaseDeploying) {
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
	} else if e.IntegrationInPhase(v1alpha1.IntegrationPhaseRunning) {
		// Let's run garbage collection during the integration running phase
		// TODO: this should be refined so that it's run when all the replicas for the newer generation
		// are ready. This is to be added when the integration scale status is refined with ready replicas

		// Register a post action that deletes the existing resources that are labelled
		// with the previous integration generations.
		e.PostActions = append(e.PostActions, func(environment *Environment) error {
			// The collection and deletion are performed asynchronously to avoid blocking
			// the reconcile loop.
			go t.garbageCollectResources(e)
			return nil
		})
	}

	return nil
}

func (t *garbageCollectorTrait) garbageCollectResources(e *Environment) {
	integration, _ := labels.NewRequirement("camel.apache.org/integration", selection.Equals, []string{e.Integration.Name})
	generation, err := labels.NewRequirement("camel.apache.org/generation", selection.LessThan, []string{strconv.FormatInt(e.Integration.GetGeneration(), 10)})
	if err != nil {
		t.L.ForIntegration(e.Integration).Errorf(err, "cannot determine generation requirement")
		return
	}
	selector := labels.NewSelector().
		Add(*integration).
		Add(*generation)

	// Retrieve older generation resources to be enlisted for garbage collection.
	// We rely on the discovery API to retrieve all the resources group and kind.
	// That results in an unbounded collection that can be a bit slow.
	// We may want to refine that step by white-listing or enlisting types to speed-up
	// the collection duration.
	resources, err := lookUpResources(context.TODO(), e.Client, e.Integration.Namespace, selector)
	if err != nil {
		t.L.ForIntegration(e.Integration).Errorf(err, "cannot collect older generation resources")
		return
	}

	// And delete them
	for _, resource := range resources {
		// pin the resource
		resource := resource
		err = e.Client.Delete(context.TODO(), &resource, controller.PropagationPolicy(metav1.DeletePropagationBackground))
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

func lookUpResources(ctx context.Context, client client.Client, namespace string, selector labels.Selector) ([]unstructured.Unstructured, error) {
	// We only take types that support the "create" and "list" verbs as:
	// - they have to be created to be deleted :) so that excludes read-only
	//   resources, e.g., aggregated APIs
	// - they are going to be iterated and a list query with labels selector
	//   is performed for each of them. That prevents from performing queries
	//   that we know are going to return "MethodNotAllowed".
	types, err := getDiscoveryTypesWithVerbs(client, []string{"create", "list"})
	if err != nil {
		return nil, err
	}

	res := make([]unstructured.Unstructured, 0)

	for _, t := range types {
		options := controller.ListOptions{
			Namespace:     namespace,
			LabelSelector: selector,
			Raw: &metav1.ListOptions{
				TypeMeta: t,
			},
		}
		list := unstructured.UnstructuredList{
			Object: map[string]interface{}{
				"apiVersion": t.APIVersion,
				"kind":       t.Kind,
			},
		}
		if err := client.List(ctx, &options, &list); err != nil {
			if k8serrors.IsNotFound(err) || k8serrors.IsForbidden(err) {
				continue
			}
			return nil, err
		}

		res = append(res, list.Items...)
	}
	return res, nil
}

func getDiscoveryTypesWithVerbs(client client.Client, verbs []string) ([]metav1.TypeMeta, error) {
	resources, err := client.Discovery().ServerPreferredNamespacedResources()
	// Swallow group discovery errors, e.g., Knative serving exposes
	// an aggregated API for custom.metrics.k8s.io that requires special
	// authentication scheme while discovering preferred resources
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, err
	}

	types := make([]metav1.TypeMeta, 0)
	for _, resource := range resources {
		for _, r := range resource.APIResources {
			if len(verbs) > 0 && !util.StringSliceContains(r.Verbs, verbs) {
				// Do not return the type if it does not support the provided verbs
				continue
			}
			types = append(types, metav1.TypeMeta{
				Kind:       r.Kind,
				APIVersion: resource.GroupVersion,
			})
		}
	}

	return types, nil
}
