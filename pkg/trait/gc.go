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
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"

	authorization "k8s.io/api/authorization/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"
	"k8s.io/utils/pointer"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util"
)

var (
	lock            sync.Mutex
	rateLimiter     = rate.NewLimiter(rate.Every(time.Minute), 1)
	collectableGVKs = make(map[schema.GroupVersionKind]struct{})
)

type gcTrait struct {
	BaseTrait
	traitv1.GCTrait `property:",squash"`
}

func newGCTrait() Trait {
	return &gcTrait{
		BaseTrait: NewBaseTrait("gc", 1200),
	}
}

func (t *gcTrait) Configure(e *Environment) (bool, error) {
	if e.Integration == nil || !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseInitialization) || e.IntegrationInRunningPhases(), nil
}

func (t *gcTrait) Apply(e *Environment) error {
	if e.IntegrationInRunningPhases() && e.Integration.GetGeneration() > 1 {
		// Register a post action that deletes the existing resources that are labelled
		// with the previous integration generation(s).
		// We make the assumption generation is a monotonically increasing strictly positive integer,
		// in which case we can skip garbage collection on the first generation.
		// TODO: this should be refined so that it's run when all the replicas for the newer generation are ready.
		e.PostActions = append(e.PostActions, func(env *Environment) error {
			return t.garbageCollectResources(env)
		})
	}

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

	return nil
}

func (t *gcTrait) garbageCollectResources(e *Environment) error {
	deletableGVKs, err := t.getDeletableTypes(e)
	if err != nil {
		return errors.Wrap(err, "cannot discover GVK types")
	}

	integration, _ := labels.NewRequirement(v1.IntegrationLabel, selection.Equals, []string{e.Integration.Name})
	generation, err := labels.NewRequirement("camel.apache.org/generation", selection.LessThan, []string{strconv.FormatInt(e.Integration.GetGeneration(), 10)})
	if err != nil {
		return errors.Wrap(err, "cannot determine generation requirement")
	}
	selector := labels.NewSelector().
		Add(*integration).
		Add(*generation)

	return t.deleteEachOf(e.Ctx, deletableGVKs, e, selector)
}

func (t *gcTrait) deleteEachOf(ctx context.Context, deletableGVKs map[schema.GroupVersionKind]struct{}, e *Environment, selector labels.Selector) error {
	for GVK := range deletableGVKs {
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
				return errors.Wrap(err, "cannot list child resources")
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

func (t *gcTrait) canBeDeleted(e *Environment, u unstructured.Unstructured) bool {
	// Only delete direct children of the integration, otherwise we can affect the behavior of external controllers (i.e. Knative)
	for _, o := range u.GetOwnerReferences() {
		if o.Kind == v1.IntegrationKind && strings.HasPrefix(o.APIVersion, v1.SchemeGroupVersion.Group) && o.Name == e.Integration.Name {
			return true
		}
	}
	return false
}

func (t *gcTrait) getDeletableTypes(e *Environment) (map[schema.GroupVersionKind]struct{}, error) {
	lock.Lock()
	defer lock.Unlock()

	// Rate limit to avoid Discovery and SelfSubjectRulesReview requests at every reconciliation.
	if !rateLimiter.Allow() {
		// Return the cached set of garbage collectable GVKs.
		return collectableGVKs, nil
	}

	// We rely on the discovery API to retrieve all the resources GVK,
	// that results in an unbounded set that can impact garbage collection latency when scaling up.
	resources, err := t.Client.Discovery().ServerPreferredNamespacedResources()
	// Swallow group discovery errors, e.g., Knative serving exposes
	// an aggregated API for custom.metrics.k8s.io that requires special
	// authentication scheme while discovering preferred resources.
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, err
	}

	// We only take types that support the "delete" verb,
	// to prevents from performing queries that we know are going to return "MethodNotAllowed".
	APIResourceLists := discovery.FilteredBy(discovery.SupportsAllVerbs{Verbs: []string{"delete"}}, resources)

	// Retrieve the permissions granted to the operator service account.
	// We assume the operator has only to garbage collect the resources it has created.
	ssrr := &authorization.SelfSubjectRulesReview{
		Spec: authorization.SelfSubjectRulesReviewSpec{
			Namespace: e.Integration.Namespace,
		},
	}
	ssrr, err = e.Client.AuthorizationV1().SelfSubjectRulesReviews().Create(e.Ctx, ssrr, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	GVKs := make(map[schema.GroupVersionKind]struct{})
	for _, APIResourceList := range APIResourceLists {
		for _, resource := range APIResourceList.APIResources {
			resourceGroup := resource.Group
			if resourceGroup == "" {
				// Empty implies the group of the containing resource list should be used
				gv, err := schema.ParseGroupVersion(APIResourceList.GroupVersion)
				if err != nil {
					return nil, err
				}
				resourceGroup = gv.Group
			}
		rule:
			for _, rule := range ssrr.Status.ResourceRules {
				if !util.StringSliceContainsAnyOf(rule.Verbs, "delete", "*") {
					continue
				}
				for _, ruleGroup := range rule.APIGroups {
					for _, ruleResource := range rule.Resources {
						if (resourceGroup == ruleGroup || ruleGroup == "*") && (resource.Name == ruleResource || ruleResource == "*") {
							GVK := schema.FromAPIVersionAndKind(APIResourceList.GroupVersion, resource.Kind)
							GVKs[GVK] = struct{}{}
							break rule
						}
					}
				}
			}
		}
	}
	collectableGVKs = GVKs

	return collectableGVKs, nil
}
