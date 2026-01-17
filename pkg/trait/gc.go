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
	"maps"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	appsv1 "k8s.io/api/apps/v1"
	authorization "k8s.io/api/authorization/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/knative"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

var (
	lock                  sync.Mutex
	rateLimiter           = rate.NewLimiter(rate.Every(time.Minute), 1)
	collectableGVKs       = make(map[schema.GroupVersionKind]struct{})
	defaultDeletableTypes = map[schema.GroupVersionKind]struct{}{
		{
			Kind:    "ConfigMap",
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
		}: {},
		{
			Kind:    "Deployment",
			Group:   appsv1.SchemeGroupVersion.Group,
			Version: appsv1.SchemeGroupVersion.Version,
		}: {},
		{
			Kind:    "Secret",
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
		}: {},
		{
			Kind:    "Service",
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
		}: {},
		{
			Kind:    "CronJob",
			Group:   batchv1.SchemeGroupVersion.Group,
			Version: batchv1.SchemeGroupVersion.Version,
		}: {},
		{
			Kind:    "Job",
			Group:   batchv1.SchemeGroupVersion.Group,
			Version: batchv1.SchemeGroupVersion.Version,
		}: {},
	}
)

const (
	gcTraitID    = "gc"
	gcTraitOrder = 1200
)

type gcTrait struct {
	BaseTrait
	traitv1.GCTrait `property:",squash"`
}

func newGCTrait() Trait {
	return &gcTrait{
		BaseTrait: NewBaseTrait(gcTraitID, gcTraitOrder),
	}
}

func (t *gcTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}
	if !ptr.Deref(t.Enabled, true) {
		return false, NewIntegrationConditionUserDisabled("GC"), nil
	}

	// We need to execute this trait only when all resources have been created and
	// deployed with a new generation if there is was any change during the Integration drift.
	return e.IntegrationInRunningPhases() || e.IntegrationInPhase(v1.IntegrationPhaseBuildComplete), nil, nil
}

func (t *gcTrait) Apply(e *Environment) error {
	// Garbage collection runs when:
	// 1. Generation > 1: resource was updated, clean up old generation resources
	// 2. BuildComplete phase AND integration has previously been deployed: undeploy scenario
	shouldRunGC := e.Integration.GetGeneration() > 1

	if !shouldRunGC && e.IntegrationInPhase(v1.IntegrationPhaseBuildComplete) {
		// Only run GC if integration was previously deployed (undeploy case)
		if !hasNeverDeployed(e.Integration) {
			shouldRunGC = true
		}
	}

	if shouldRunGC {
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
			resourceLabels := resource.GetLabels()
			// Label the resource with the current integration generation
			resourceLabels[v1.IntegrationGenerationLabel] = generation
			// Make sure the integration label is set
			resourceLabels[v1.IntegrationLabel] = env.Integration.Name
			resource.SetLabels(resourceLabels)
		})

		return nil
	})

	return nil
}

func (t *gcTrait) garbageCollectResources(e *Environment) error {
	deletableGVKs, err := t.getDeletableTypes(e)
	if err != nil {
		return fmt.Errorf("cannot discover GVK types: %w", err)
	}

	profile := e.DetermineProfile()
	deletableTypesByProfile := map[schema.GroupVersionKind]struct{}{}

	if profile == v1.TraitProfileKnative {
		if ok, _ := knative.IsServingInstalled(e.Client); ok {
			deletableTypesByProfile[schema.GroupVersionKind{
				Kind:    "Service",
				Group:   "serving.knative.dev",
				Version: "v1",
			}] = struct{}{}
		}

		if ok, _ := knative.IsEventingInstalled(e.Client); ok {
			deletableTypesByProfile[schema.GroupVersionKind{
				Kind:    "Trigger",
				Group:   "eventing.knative.dev",
				Version: "v1",
			}] = struct{}{}
		}
	}

	// copy profile related deletable types if not already present
	for key, value := range deletableTypesByProfile {
		if _, found := deletableGVKs[key]; !found {
			deletableGVKs[key] = value
		}
	}

	integration, _ := labels.NewRequirement(v1.IntegrationLabel, selection.Equals, []string{e.Integration.Name})
	generation, err := labels.NewRequirement(v1.IntegrationGenerationLabel, selection.LessThan, []string{strconv.FormatInt(e.Integration.GetGeneration(), 10)})
	if err != nil {
		return fmt.Errorf("cannot determine generation requirement: %w", err)
	}
	selector := labels.NewSelector().
		Add(*integration)

	// On undeploy, delete all resources regardless of generation.
	// On generation upgrade, filter to only delete old resources.
	isUndeploying := e.IntegrationInPhase(v1.IntegrationPhaseBuildComplete) && !hasNeverDeployed(e.Integration)
	if !isUndeploying {
		selector = selector.Add(*generation)
	}

	return t.deleteEachOf(e.Ctx, deletableGVKs, e, selector)
}

// deleteEachOf takes care of the effective deletion of each deletableGVKs passed for the given selector. It should be any older generation resource
// of the Integration.
func (t *gcTrait) deleteEachOf(ctx context.Context, deletableGVKs map[schema.GroupVersionKind]struct{}, e *Environment, selector labels.Selector) error {
	for GVK := range deletableGVKs {
		resources := unstructured.UnstructuredList{
			Object: map[string]any{
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
				return fmt.Errorf("cannot list child resources: %w", err)
			}

			continue
		}

		for _, resource := range resources.Items {
			r := resource
			if !canBeDeleted(e.Integration, r) {
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

// canBeDeleted is an important security check. We make sure that we are only deleting those resources belonging to the given Integration.
func canBeDeleted(it *v1.Integration, u unstructured.Unstructured) bool {
	for _, o := range u.GetOwnerReferences() {
		if o.Kind == v1.IntegrationKind && strings.HasPrefix(o.APIVersion, v1.SchemeGroupVersion.Group) && o.Name == it.Name {
			return true
		}
	}

	return false
}

// hasNeverDeployed returns true if the integration has never been deployed.
// Checks both DeploymentTimestamp and Ready condition for reliability.
func hasNeverDeployed(integration *v1.Integration) bool {
	// Primary check: DeploymentTimestamp is set when deployment is triggered
	if integration.Status.DeploymentTimestamp != nil && !integration.Status.DeploymentTimestamp.IsZero() {
		return false // has been deployed
	}

	// Secondary check: Ready condition becomes true only after successful deployment
	readyCond := integration.Status.GetCondition(v1.IntegrationConditionReady)
	if readyCond != nil && readyCond.FirstTruthyTime != nil && !readyCond.FirstTruthyTime.IsZero() {
		return false
	}

	return true
}

// getDeletableTypes returns the list of deletable types resources, inspecting the rules for which the operator SA is allowed in the
// Integration namespace.
func (t *gcTrait) getDeletableTypes(e *Environment) (map[schema.GroupVersionKind]struct{}, error) {
	lock.Lock()
	defer lock.Unlock()

	// Return a fresh map even when returning cached collectables
	GVKs := make(map[schema.GroupVersionKind]struct{})

	// Rate limit to avoid Discovery and SelfSubjectRulesReview requests at every reconciliation.
	if !rateLimiter.Allow() {
		// Return the cached set of garbage collectable GVKs.
		maps.Copy(GVKs, collectableGVKs)

		return GVKs, nil
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

	if len(GVKs) == 0 {
		// Auto discovery of deletable types has no results (probably an error)
		// Make sure to at least use a minimal set of deletable types for garbage collection
		t.L.ForIntegration(e.Integration).Debugf("Auto discovery of deletable types returned no results. " +
			"Using default minimal set of deletable types for garbage collection")
		maps.Copy(GVKs, defaultDeletableTypes)
	}

	collectableGVKs = make(map[schema.GroupVersionKind]struct{})
	maps.Copy(collectableGVKs, GVKs)

	for gvk := range GVKs {
		log.Debugf("Found deletable type: %s", gvk.String())
	}

	return GVKs, nil
}
