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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
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
	e.PostActions = append(e.PostActions, func(environment *Environment) error {
		// Retrieve older generation resources that may can enlisted for garbage collection
		resources, err := getOldGenerationResources(e)
		if err != nil {
			return err
		}
		// And delete them
		for _, resource := range resources {
			err = e.Client.Delete(context.TODO(), resource)
			if err != nil {
				// The resource may have already been deleted
				if !k8serrors.IsNotFound(err) {
					t.L.ForIntegration(e.Integration).Errorf(err, "cannot delete child resource: %s/%s", resource.GetKind(), resource.GetName())
				}
			} else {
				t.L.ForIntegration(e.Integration).Debugf("child resource deleted: %s/%s", resource.GetKind(), resource.GetName())
			}
		}

		return nil
	})

	return nil
}

func getOldGenerationResources(e *Environment) ([]*unstructured.Unstructured, error) {
	// We rely on the discovery API to retrieve all the resources group and kind.
	// That results in an unbounded collection that can be a bit slow (a couple of seconds).
	// We may want to refine that step by white-listing or enlisting types to speed-up
	// the collection duration.
	types, err := getDiscoveryTypes(e.Client)
	if err != nil {
		return nil, err
	}

	selector, err := labels.Parse(fmt.Sprintf("camel.apache.org/integration=%s,camel.apache.org/generation,camel.apache.org/generation notin (%d)", e.Integration.Name, e.Integration.GetGeneration()))
	if err != nil {
		return nil, err
	}

	res := make([]*unstructured.Unstructured, 0)

	for _, t := range types {
		options := k8sclient.ListOptions{
			Namespace:     e.Integration.Namespace,
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
		if err := e.Client.List(context.TODO(), &options, &list); err != nil {
			if k8serrors.IsNotFound(err) ||
				k8serrors.IsForbidden(err) ||
				k8serrors.IsMethodNotSupported(err) {
				continue
			}
			return nil, err
		}
		for _, item := range list.Items {
			res = append(res, &item)
		}
	}
	return res, nil
}

func getDiscoveryTypes(client client.Client) ([]metav1.TypeMeta, error) {
	resources, err := client.Discovery().ServerPreferredNamespacedResources()
	if err != nil {
		return nil, err
	}

	types := make([]metav1.TypeMeta, 0)
	for _, resource := range resources {
		for _, r := range resource.APIResources {
			types = append(types, metav1.TypeMeta{
				Kind:       r.Kind,
				APIVersion: resource.GroupVersion,
			})
		}
	}

	return types, nil
}
