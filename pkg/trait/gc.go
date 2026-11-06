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
	"slices"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/utils/ptr"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/v2/pkg/apis"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/resources"
	"github.com/apache/camel-k/v2/pkg/util/log"
)

var (
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
	return e.IntegrationInRunningPhases() || e.IntegrationInPhase(v1.IntegrationPhaseUnDeploying), nil, nil
}

func (t *gcTrait) Apply(e *Environment) error {
	// Garbage collection runs when:
	// 1. Generation > 1: resource was updated, clean up old generation resources
	// 2. undeploy scenario
	if e.Integration.GetGeneration() > 1 || e.IntegrationInPhase(v1.IntegrationPhaseUnDeploying) {
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
	deletableGVKs, err := t.getDeletableTypes()
	if err != nil {
		return fmt.Errorf("cannot determine deletable types: %w", err)
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
	if !e.IntegrationInPhase(v1.IntegrationPhaseUnDeploying) {
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
			// The type may not exist in the cluster at all, for example when
			// the CRD of an optional component (such as Knative or KEDA) is
			// not installed: skip it instead of failing the garbage
			// collection of the remaining types.
			if k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) {
				t.L.ForIntegration(e.Integration).Debugf("skipping deletable type %s: %s", GVK.String(), err.Error())

				continue
			}

			return fmt.Errorf("cannot list child resources: %w", err)
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

// deletableTypesFromRules builds the in-memory map of deletable types from the
// rules granted to the operator service account. It is a pure function over its
// inputs, so the mapping from RBAC rules to deletable types can be tested
// without a cluster.
//
// A resource is deletable when some rule allows the "delete" verb (or "*") for
// its API group (or "*") and its resource name (or "*"). Only resources present
// in resourceTypes are considered: role rules carry no version or kind
// information, and subresources cannot be deleted as a whole. Every registered
// version of a deletable resource is included in the result.
func deletableTypesFromRules(rules []rbacv1.PolicyRule, resourceTypes map[schema.GroupResource][]schema.GroupVersionKind) map[schema.GroupVersionKind]struct{} {
	GVKs := make(map[schema.GroupVersionKind]struct{})

	for groupResource, resourceGVKs := range resourceTypes {
		for _, rule := range rules {
			if !slices.Contains(rule.Verbs, "delete") && !slices.Contains(rule.Verbs, "*") {
				continue
			}
			if !slices.Contains(rule.APIGroups, groupResource.Group) && !slices.Contains(rule.APIGroups, "*") {
				continue
			}
			if !slices.Contains(rule.Resources, groupResource.Resource) && !slices.Contains(rule.Resources, "*") {
				continue
			}
			for _, GVK := range resourceGVKs {
				GVKs[GVK] = struct{}{}
			}

			break
		}
	}

	return GVKs
}

// schemeResourceTypes indexes the types registered in a scheme by the API
// group and resource name they are served under, as those names appear in
// role rules. List types and the internal option types registered for every
// group version are skipped, as they cannot be deleted. Every registered
// version of a resource is returned: which versions are actually served is
// a property of the cluster, not of the operator binary, so garbage
// collection attempts them all and skips the ones the cluster does not
// serve.
func schemeResourceTypes(s *runtime.Scheme) map[schema.GroupResource][]schema.GroupVersionKind {
	resourceTypes := make(map[schema.GroupResource][]schema.GroupVersionKind)
	for GVK := range s.AllKnownTypes() {
		if strings.HasSuffix(GVK.Kind, "List") || isInternalKind(GVK.Kind) {
			continue
		}
		groupResource := schema.GroupResource{Group: GVK.Group, Resource: kindToResource(GVK.Kind)}
		resourceTypes[groupResource] = append(resourceTypes[groupResource], GVK)
	}
	// Sort every version list, so the result does not depend on the map
	// iteration or registration order.
	for _, GVKs := range resourceTypes {
		slices.SortFunc(GVKs, func(a, b schema.GroupVersionKind) int {
			return strings.Compare(a.String(), b.String())
		})
	}

	return resourceTypes
}

// isInternalKind reports whether a kind is one of the internal types that
// metav1.AddToGroupVersion registers for every group version in a scheme.
// They are API machinery, not resources an operator can delete.
func isInternalKind(kind string) bool {
	switch kind {
	case "CreateOptions", "DeleteOptions", "GetOptions", "ListOptions",
		"PatchOptions", "UpdateOptions", "WatchEvent", "Status":
		return true
	}

	return false
}

// kindToResource converts a kind to the resource name it is served under,
// applying the usual Kubernetes pluralization rules: "Gateway" becomes
// "gateways", "NetworkPolicy" becomes "networkpolicies" and "Ingress"
// becomes "ingresses". "Endpoints" is the one irregular case among the
// registered types: its resource is the kind itself, unmodified. The
// conversion is verified exhaustively against the shipped operator roles
// by TestOperatorRolesDeletableCoverage: a resource whose name is computed
// wrongly here would silently stop being garbage collected, so the test
// fails the build instead, naming the resource.
func kindToResource(kind string) string {
	if kind == "Endpoints" {
		return "endpoints"
	}
	lower := strings.ToLower(kind)
	switch {
	case strings.HasSuffix(lower, "s"), strings.HasSuffix(lower, "x"),
		strings.HasSuffix(lower, "z"), strings.HasSuffix(lower, "ch"),
		strings.HasSuffix(lower, "sh"):
		return lower + "es"
	case strings.HasSuffix(lower, "y"):
		if len(lower) > 1 && strings.ContainsRune("aeiou", rune(lower[len(lower)-2])) {
			return lower + "s"
		}

		return lower[:len(lower)-1] + "ies"
	default:
		return lower + "s"
	}
}

// operatorResourceTypes resolves the group and resource names used in role
// rules to the types the operator manages. The versions and kinds come from
// the operator's own scheme -- the Kubernetes built-in types plus every type
// registered through apis.AddToScheme, including the Knative, KEDA and other
// duck types -- instead of a hand-written table, so they always track the
// types the operator is actually built with. Types with no Go type
// registered in the scheme are not resolved: the operator does not create
// them, so it does not garbage collect them either.
func operatorResourceTypes() (map[schema.GroupResource][]schema.GroupVersionKind, error) {
	s := runtime.NewScheme()
	if err := clientscheme.AddToScheme(s); err != nil {
		return nil, fmt.Errorf("cannot register Kubernetes types in scheme: %w", err)
	}
	if err := apis.AddToScheme(s); err != nil {
		return nil, fmt.Errorf("cannot register operator types in scheme: %w", err)
	}

	return schemeResourceTypes(s), nil
}

// operatorRoleRules returns the rules of all the operator roles for namespaced
// installations, as shipped in the embedded resource files. Those roles are
// the static source of truth for the types the operator is allowed to delete:
// no discovery or SelfSubjectRulesReview query against the cluster is needed.
func operatorRoleRules() ([]rbacv1.PolicyRule, error) {
	files, err := resources.WithPrefix("config/rbac/namespaced/operator-role")
	if err != nil {
		return nil, err
	}

	rules := make([]rbacv1.PolicyRule, 0)
	for _, file := range files {
		// Role bindings grant the roles but carry no rule themselves.
		if strings.Contains(file, "binding") {
			continue
		}
		content, err := resources.Resource(file)
		if err != nil {
			return nil, err
		}
		role := rbacv1.Role{}
		if err := yaml.Unmarshal(content, &role); err != nil {
			return nil, fmt.Errorf("cannot parse operator role %s: %w", file, err)
		}
		rules = append(rules, role.Rules...)
	}

	return rules, nil
}

// loadDeletableTypes computes the set of deletable types from the operator
// roles shipped with the operator.
func loadDeletableTypes() (map[schema.GroupVersionKind]struct{}, error) {
	rules, err := operatorRoleRules()
	if err != nil {
		return nil, err
	}
	resourceTypes, err := operatorResourceTypes()
	if err != nil {
		return nil, err
	}

	return deletableTypesFromRules(rules, resourceTypes), nil
}

// staticDeletableTypes is the set of deletable types derived from the
// operator roles, computed once when the package is initialized: the roles
// are bundled in the binary, so the set can only change when the operator
// is restarted with a different binary. A computation error does not fail
// the initialization: it is kept here and reported to every garbage
// collection run, as it was when the set was computed lazily.
var staticDeletableTypes, staticDeletableTypesErr = loadDeletableTypes()

// getDeletableTypes returns the set of deletable types for garbage collection:
// the types the operator is allowed to delete according to the operator roles
// shipped with the operator, plus the default minimal set of types the
// operator always creates for an Integration.
func (t *gcTrait) getDeletableTypes() (map[schema.GroupVersionKind]struct{}, error) {
	if staticDeletableTypesErr != nil {
		return nil, staticDeletableTypesErr
	}
	staticTypes := staticDeletableTypes

	if len(staticTypes) == 0 {
		// The static computation returned no results (probably an error).
		// Make sure to at least use a minimal set of deletable types for
		// garbage collection.
		log.Debugf("Static computation of deletable types returned no results. " +
			"Using default minimal set of deletable types for garbage collection")
	}

	// Return a fresh map, as it is local to the garbage collection run.
	GVKs := make(map[schema.GroupVersionKind]struct{}, len(staticTypes)+len(defaultDeletableTypes))
	maps.Copy(GVKs, staticTypes)
	maps.Copy(GVKs, defaultDeletableTypes)

	for gvk := range GVKs {
		log.Debugf("Found deletable type: %s", gvk.String())
	}

	return GVKs, nil
}
