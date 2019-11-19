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
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/disk"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	util "github.com/apache/camel-k/pkg/util/controller"
)

var (
	toFileName                = regexp.MustCompile(`[^(\w/\.)]`)
	diskCachedDiscoveryClient discovery.CachedDiscoveryInterface
	DiscoveryClientLock       sync.Mutex
)

type garbageCollectorTrait struct {
	BaseTrait      `property:",squash"`
	DiscoveryCache string `property:"discovery-cache"`
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

	collectionGVKs, deletableGVKs, err := t.getDeletableTypes()
	if err != nil {
		t.L.ForIntegration(e.Integration).Errorf(err, "cannot discover GVK types")
		return
	}

	t.deleteAllOf(collectionGVKs, e, selector)
	// TODO: DeleteCollection is currently not supported for Service resources, so we have to keep
	// client-side collection deletion around until it becomes supported.
	t.deleteEachOf(deletableGVKs, e, selector)
}

func (t *garbageCollectorTrait) deleteAllOf(GKVs map[schema.GroupVersionKind]struct{}, e *Environment, selector labels.Selector) {
	for GVK := range GKVs {
		err := e.Client.DeleteAllOf(context.TODO(),
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": GVK.GroupVersion().String(),
					"kind":       GVK.Kind,
					"metadata": map[string]interface{}{
						"namespace": e.Integration.Namespace,
					},
				},
			},
			// FIXME: The unstructured client doesn't take the namespace option into account
			//controller.InNamespace(e.Integration.Namespace),
			util.MatchingSelector{Selector: selector},
			client.PropagationPolicy(metav1.DeletePropagationBackground),
		)
		if err != nil {
			t.L.ForIntegration(e.Integration).Errorf(err, "cannot delete child resources: %v", GVK)
		} else {
			t.L.ForIntegration(e.Integration).Debugf("child resources deleted: %v", GVK)
		}
	}
}

func (t *garbageCollectorTrait) deleteEachOf(GKVs map[schema.GroupVersionKind]struct{}, e *Environment, selector labels.Selector) {
	for GVK := range GKVs {
		resources := unstructured.UnstructuredList{
			Object: map[string]interface{}{
				"apiVersion": GVK.GroupVersion().String(),
				"kind":       GVK.Kind,
			},
		}
		options := []client.ListOption{
			client.InNamespace(e.Integration.Namespace),
			util.MatchingSelector{Selector: selector},
		}
		if err := t.client.List(context.TODO(), &resources, options...); err != nil {
			if !k8serrors.IsNotFound(err) && !k8serrors.IsForbidden(err) {
				t.L.ForIntegration(e.Integration).Errorf(err, "cannot list child resources: %v", GVK)
			}
			continue
		}

		for _, resource := range resources.Items {
			err := t.client.Delete(context.TODO(), &resource, client.PropagationPolicy(metav1.DeletePropagationBackground))
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
}

func (t *garbageCollectorTrait) getDeletableTypes() (map[schema.GroupVersionKind]struct{}, map[schema.GroupVersionKind]struct{}, error) {
	// We rely on the discovery API to retrieve all the resources GVK,
	// that results in an unbounded set that can impact garbage collection latency when scaling up.
	discoveryClient, err := t.discoveryClient()
	if err != nil {
		return nil, nil, err
	}
	resources, err := discoveryClient.ServerPreferredNamespacedResources()
	// Swallow group discovery errors, e.g., Knative serving exposes
	// an aggregated API for custom.metrics.k8s.io that requires special
	// authentication scheme while discovering preferred resources
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, nil, err
	}

	// We only take types that support the "delete" and "deletecollection" verbs,
	// to prevents from performing queries that we know are going to return "MethodNotAllowed".
	return groupVersionKinds(discovery.FilteredBy(discovery.SupportsAllVerbs{Verbs: []string{"deletecollection"}}, resources)),
		groupVersionKinds(discovery.FilteredBy(supportsDeleteVerbOnly{}, resources)),
		nil
}

func groupVersionKinds(rls []*metav1.APIResourceList) map[schema.GroupVersionKind]struct{} {
	GVKs := map[schema.GroupVersionKind]struct{}{}
	for _, rl := range rls {
		for _, r := range rl.APIResources {
			GVKs[schema.FromAPIVersionAndKind(rl.GroupVersion, r.Kind)] = struct{}{}
		}
	}
	return GVKs
}

// supportsDeleteVerbOnly is a predicate matching a resource if it supports the delete verb, but not deletecollection.
type supportsDeleteVerbOnly struct{}

func (p supportsDeleteVerbOnly) Match(groupVersion string, r *metav1.APIResource) bool {
	verbs := sets.NewString([]string(r.Verbs)...)
	return verbs.Has("delete") && !verbs.Has("deletecollection")
}

func (t *garbageCollectorTrait) discoveryClient() (discovery.DiscoveryInterface, error) {
	DiscoveryClientLock.Lock()
	defer DiscoveryClientLock.Unlock()

	if t.DiscoveryCache != "disk" {
		return t.client.Discovery(), nil
	}

	if diskCachedDiscoveryClient != nil {
		return diskCachedDiscoveryClient, nil
	}

	config := t.client.GetConfig()
	httpCacheDir := filepath.Join(mustHomeDir(), ".kube", "http-cache")
	discCacheDir := filepath.Join(mustHomeDir(), ".kube", "cache", "discovery", toHostDir(config.Host))

	var err error
	diskCachedDiscoveryClient, err = disk.NewCachedDiscoveryClientForConfig(config, discCacheDir, httpCacheDir, 10*time.Minute)
	return diskCachedDiscoveryClient, err
}
