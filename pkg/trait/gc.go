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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/camel-k/pkg/util"
	authorization "k8s.io/api/authorization/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/discovery/cached/memory"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

var (
	toFileName                  = regexp.MustCompile(`[^(\w/\.)]`)
	diskCachedDiscoveryClient   discovery.CachedDiscoveryInterface
	memoryCachedDiscoveryClient discovery.CachedDiscoveryInterface
	discoveryClientLock         sync.Mutex
)

type discoveryCacheType string

const (
	disabledDiscoveryCache discoveryCacheType = "disabled"
	diskDiscoveryCache     discoveryCacheType = "disk"
	memoryDiscoveryCache   discoveryCacheType = "memory"
)

// The GC Trait garbage-collects all resources that are no longer necessary upon integration updates.
//
// +camel-k:trait=gc.
type garbageCollectorTrait struct {
	BaseTrait `property:",squash"`
	// Discovery client cache to be used, either `disabled`, `disk` or `memory` (default `memory`)
	DiscoveryCache *discoveryCacheType `property:"discovery-cache" json:"discoveryCache,omitempty"`
}

func newGarbageCollectorTrait() Trait {
	return &garbageCollectorTrait{
		BaseTrait: NewBaseTrait("gc", 1200),
	}
}

func (t *garbageCollectorTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		return false, nil
	}

	if t.DiscoveryCache == nil {
		s := memoryDiscoveryCache
		t.DiscoveryCache = &s
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseInitialization) || e.IntegrationInRunningPhases(), nil
}

func (t *garbageCollectorTrait) Apply(e *Environment) error {
	switch e.Integration.Status.Phase {

	case v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning, v1.IntegrationPhaseError:
		// Register a post action that deletes the existing resources that are labelled
		// with the previous integration generations.
		// TODO: this should be refined so that it's run when all the replicas for the newer generation
		// are ready.
		e.PostActions = append(e.PostActions, func(env *Environment) error {
			return t.garbageCollectResources(env)
		})

		fallthrough

	default:
		// Register a post processor that adds the required labels to the new resources
		e.PostProcessors = append(e.PostProcessors, func(env *Environment) error {
			generation := strconv.FormatInt(env.Integration.GetGeneration(), 10)
			env.Resources.VisitMetaObject(func(resource metav1.Object) {
				labels := resource.GetLabels()
				// Label the resource with the current integration generation
				labels["camel.apache.org/generation"] = generation
				// Make sure the integration label is set
				labels[v1.IntegrationLabel] = env.Integration.Name
				resource.SetLabels(labels)
			})
			return nil
		})
	}

	return nil
}

func (t *garbageCollectorTrait) garbageCollectResources(e *Environment) error {
	deletableGVKs, err := t.getDeletableTypes(e)
	if err != nil {
		return fmt.Errorf("cannot discover GVK types: %v", err)
	}

	integration, _ := labels.NewRequirement(v1.IntegrationLabel, selection.Equals, []string{e.Integration.Name})
	generation, err := labels.NewRequirement("camel.apache.org/generation", selection.LessThan, []string{strconv.FormatInt(e.Integration.GetGeneration(), 10)})
	if err != nil {
		return fmt.Errorf("cannot determine generation requirement: %v", err)
	}
	selector := labels.NewSelector().
		Add(*integration).
		Add(*generation)

	return t.deleteEachOf(e.Ctx, deletableGVKs, e, selector)
}

func (t *garbageCollectorTrait) deleteEachOf(ctx context.Context, GVKs map[schema.GroupVersionKind]struct{}, e *Environment, selector labels.Selector) error {
	for GVK := range GVKs {
		resources := unstructured.UnstructuredList{
			Object: map[string]interface{}{
				"apiVersion": GVK.GroupVersion().String(),
				"kind":       GVK.Kind,
			},
		}
		options := []ctrl.ListOption{
			ctrl.InNamespace(e.Integration.Namespace),
			ctrl.MatchingLabelsSelector{Selector: selector},
		}
		if err := t.Client.List(ctx, &resources, options...); err != nil {
			if !k8serrors.IsNotFound(err) {
				return fmt.Errorf("cannot list child resources: %v", err)
			}
			continue
		}

		for _, resource := range resources.Items {
			r := resource
			if !t.canBeDeleted(e, r) {
				continue
			}
			err := t.Client.Delete(ctx, &r, ctrl.PropagationPolicy(metav1.DeletePropagationBackground))
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

	return nil
}

func (t *garbageCollectorTrait) canBeDeleted(e *Environment, u unstructured.Unstructured) bool {
	// Only delete direct children of the integration, otherwise we can affect the behavior of external controllers (i.e. Knative)
	for _, o := range u.GetOwnerReferences() {
		if o.Kind == v1.IntegrationKind && strings.HasPrefix(o.APIVersion, v1.SchemeGroupVersion.Group) && o.Name == e.Integration.Name {
			return true
		}
	}
	return false
}

func (t *garbageCollectorTrait) getDeletableTypes(e *Environment) (map[schema.GroupVersionKind]struct{}, error) {
	// We rely on the discovery API to retrieve all the resources GVK,
	// that results in an unbounded set that can impact garbage collection latency when scaling up.
	discoveryClient, err := t.discoveryClient()
	if err != nil {
		return nil, err
	}
	resources, err := discoveryClient.ServerPreferredNamespacedResources()
	// Swallow group discovery errors, e.g., Knative serving exposes
	// an aggregated API for custom.metrics.k8s.io that requires special
	// authentication scheme while discovering preferred resources
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, err
	}

	// We only take types that support the "delete" verb,
	// to prevents from performing queries that we know are going to return "MethodNotAllowed".
	APIResourceLists := discovery.FilteredBy(discovery.SupportsAllVerbs{Verbs: []string{"delete"}}, resources)

	// Retrieve the permissions granted to the operator service account.
	// We assume the operator has only to garbage collect the resources it has created.
	srr := &authorization.SelfSubjectRulesReview{
		Spec: authorization.SelfSubjectRulesReviewSpec{
			Namespace: e.Integration.Namespace,
		},
	}
	res, err := e.Client.AuthorizationV1().SelfSubjectRulesReviews().Create(e.Ctx, srr, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	GVKs := make(map[schema.GroupVersionKind]struct{})
	for _, APIResourceList := range APIResourceLists {
		for _, resource := range APIResourceList.APIResources {
		rule:
			for _, rule := range res.Status.ResourceRules {
				if !util.StringSliceContainsAnyOf(rule.Verbs, "delete", "*") {
					continue
				}
				for _, group := range rule.APIGroups {
					for _, name := range rule.Resources {
						if (resource.Group == group || group == "*") && (resource.Name == name || name == "*") {
							GVK := schema.FromAPIVersionAndKind(APIResourceList.GroupVersion, resource.Kind)
							GVKs[GVK] = struct{}{}
							break rule
						}
					}
				}
			}
		}
	}

	return GVKs, nil
}

func (t *garbageCollectorTrait) discoveryClient() (discovery.DiscoveryInterface, error) {
	discoveryClientLock.Lock()
	defer discoveryClientLock.Unlock()

	switch *t.DiscoveryCache {
	case diskDiscoveryCache:
		if diskCachedDiscoveryClient != nil {
			return diskCachedDiscoveryClient, nil
		}
		config := t.Client.GetConfig()
		httpCacheDir := filepath.Join(mustHomeDir(), ".kube", "http-cache")
		diskCacheDir := filepath.Join(mustHomeDir(), ".kube", "cache", "discovery", toHostDir(config.Host))
		var err error
		diskCachedDiscoveryClient, err = disk.NewCachedDiscoveryClientForConfig(config, diskCacheDir, httpCacheDir, 10*time.Minute)
		return diskCachedDiscoveryClient, err

	case memoryDiscoveryCache:
		if memoryCachedDiscoveryClient != nil {
			return memoryCachedDiscoveryClient, nil
		}
		memoryCachedDiscoveryClient = memory.NewMemCacheClient(t.Client.Discovery())
		return memoryCachedDiscoveryClient, nil

	case disabledDiscoveryCache, "":
		return t.Client.Discovery(), nil

	default:
		return nil, fmt.Errorf("unsupported discovery cache type: %s", *t.DiscoveryCache)
	}
}
