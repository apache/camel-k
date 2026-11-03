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
	"sync"

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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/yaml"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

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

	// deletableResourceTypes maps an API group and resource name, as they appear
	// in the operator role rules, to the corresponding type. Role rules carry
	// neither version nor kind information, so the mapping has to be provided
	// explicitly for every type the operator may be allowed to delete. Rules
	// referencing a resource that is not in this mapping (including any
	// subresource, such as "pods/exec") are skipped when the set of deletable
	// types is computed.
	deletableResourceTypes = map[schema.GroupResource]schema.GroupVersionKind{
		{Group: "", Resource: "configmaps"}:             corev1.SchemeGroupVersion.WithKind("ConfigMap"),
		{Group: "", Resource: "secrets"}:                corev1.SchemeGroupVersion.WithKind("Secret"),
		{Group: "", Resource: "pods"}:                   corev1.SchemeGroupVersion.WithKind("Pod"),
		{Group: "", Resource: "services"}:               corev1.SchemeGroupVersion.WithKind("Service"),
		{Group: "", Resource: "serviceaccounts"}:        corev1.SchemeGroupVersion.WithKind("ServiceAccount"),
		{Group: "", Resource: "persistentvolumeclaims"}: corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"),
		{Group: "", Resource: "events"}:                 corev1.SchemeGroupVersion.WithKind("Event"),
		{Group: "apps", Resource: "deployments"}:        appsv1.SchemeGroupVersion.WithKind("Deployment"),
		{Group: "batch", Resource: "cronjobs"}:          batchv1.SchemeGroupVersion.WithKind("CronJob"),
		{Group: "batch", Resource: "jobs"}:              batchv1.SchemeGroupVersion.WithKind("Job"),
		{Group: "policy", Resource: "poddisruptionbudgets"}: {
			Group: "policy", Version: "v1", Kind: "PodDisruptionBudget",
		},
		{Group: "networking.k8s.io", Resource: "ingresses"}: {
			Group: "networking.k8s.io", Version: "v1", Kind: "Ingress",
		},
		{Group: "networking.k8s.io", Resource: "networkpolicies"}: {
			Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy",
		},
		{Group: "gateway.networking.k8s.io", Resource: "gateways"}: {
			Group: "gateway.networking.k8s.io", Version: "v1", Kind: "Gateway",
		},
		{Group: "gateway.networking.k8s.io", Resource: "httproutes"}: {
			Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRoute",
		},
		{Group: "rbac.authorization.k8s.io", Resource: "roles"}:        rbacv1.SchemeGroupVersion.WithKind("Role"),
		{Group: "rbac.authorization.k8s.io", Resource: "rolebindings"}: rbacv1.SchemeGroupVersion.WithKind("RoleBinding"),
		{Group: "coordination.k8s.io", Resource: "leases"}: {
			Group: "coordination.k8s.io", Version: "v1", Kind: "Lease",
		},
		{Group: "events.k8s.io", Resource: "events"}: {
			Group: "events.k8s.io", Version: "v1", Kind: "Event",
		},
		{Group: v1.SchemeGroupVersion.Group, Resource: "builds"}:               v1.SchemeGroupVersion.WithKind("Build"),
		{Group: v1.SchemeGroupVersion.Group, Resource: "camelcatalogs"}:        v1.SchemeGroupVersion.WithKind("CamelCatalog"),
		{Group: v1.SchemeGroupVersion.Group, Resource: "integrationkits"}:      v1.SchemeGroupVersion.WithKind("IntegrationKit"),
		{Group: v1.SchemeGroupVersion.Group, Resource: "integrationplatforms"}: v1.SchemeGroupVersion.WithKind("IntegrationPlatform"),
		{Group: v1.SchemeGroupVersion.Group, Resource: "integrationprofiles"}:  v1.SchemeGroupVersion.WithKind("IntegrationProfile"),
		{Group: v1.SchemeGroupVersion.Group, Resource: "integrations"}:         v1.SchemeGroupVersion.WithKind("Integration"),
		{Group: v1.SchemeGroupVersion.Group, Resource: "pipes"}:                v1.SchemeGroupVersion.WithKind("Pipe"),
		{Group: v1.SchemeGroupVersion.Group, Resource: "kamelets"}:             v1.SchemeGroupVersion.WithKind("Kamelet"),
		{Group: "serving.knative.dev", Resource: "services"}: {
			Group: "serving.knative.dev", Version: "v1", Kind: "Service",
		},
		{Group: "serving.knative.dev", Resource: "routes"}: {
			Group: "serving.knative.dev", Version: "v1", Kind: "Route",
		},
		{Group: "eventing.knative.dev", Resource: "triggers"}: {
			Group: "eventing.knative.dev", Version: "v1", Kind: "Trigger",
		},
		{Group: "eventing.knative.dev", Resource: "brokers"}: {
			Group: "eventing.knative.dev", Version: "v1", Kind: "Broker",
		},
		{Group: "messaging.knative.dev", Resource: "subscriptions"}: {
			Group: "messaging.knative.dev", Version: "v1", Kind: "Subscription",
		},
		{Group: "messaging.knative.dev", Resource: "channels"}: {
			Group: "messaging.knative.dev", Version: "v1", Kind: "Channel",
		},
		{Group: "messaging.knative.dev", Resource: "inmemorychannels"}: {
			Group: "messaging.knative.dev", Version: "v1", Kind: "InMemoryChannel",
		},
		{Group: "sources.knative.dev", Resource: "sinkbindings"}: {
			Group: "sources.knative.dev", Version: "v1", Kind: "SinkBinding",
		},
		{Group: "keda.sh", Resource: "scaledobjects"}: {
			Group: "keda.sh", Version: "v1alpha1", Kind: "ScaledObject",
		},
		{Group: "keda.sh", Resource: "triggerauthentications"}: {
			Group: "keda.sh", Version: "v1alpha1", Kind: "TriggerAuthentication",
		},
		{Group: "monitoring.coreos.com", Resource: "podmonitors"}: {
			Group: "monitoring.coreos.com", Version: "v1", Kind: "PodMonitor",
		},
		{Group: "build.openshift.io", Resource: "buildconfigs"}: {
			Group: "build.openshift.io", Version: "v1", Kind: "BuildConfig",
		},
		{Group: "build.openshift.io", Resource: "builds"}: {
			Group: "build.openshift.io", Version: "v1", Kind: "Build",
		},
		{Group: "image.openshift.io", Resource: "imagestreams"}: {
			Group: "image.openshift.io", Version: "v1", Kind: "ImageStream",
		},
		{Group: "image.openshift.io", Resource: "imagestreamtags"}: {
			Group: "image.openshift.io", Version: "v1", Kind: "ImageStreamTag",
		},
		{Group: "route.openshift.io", Resource: "routes"}: {
			Group: "route.openshift.io", Version: "v1", Kind: "Route",
		},
	}

	deletableTypesOnce   sync.Once
	cachedDeletableTypes map[schema.GroupVersionKind]struct{}
	errCachedDeletable   error
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
// its API group (or "*") and its resource name (or "*"). Only resources listed
// in the deletableResourceTypes mapping are considered: role rules carry no
// version or kind information, and subresources cannot be deleted as a whole.
func deletableTypesFromRules(rules []rbacv1.PolicyRule) map[schema.GroupVersionKind]struct{} {
	GVKs := make(map[schema.GroupVersionKind]struct{})

	for groupResource, GVK := range deletableResourceTypes {
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
			GVKs[GVK] = struct{}{}

			break
		}
	}

	return GVKs
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
// roles shipped with the operator. The set only depends on the roles bundled
// in the binary, so it is computed once and cached for the operator lifetime.
func loadDeletableTypes() (map[schema.GroupVersionKind]struct{}, error) {
	deletableTypesOnce.Do(func() {
		var rules []rbacv1.PolicyRule
		rules, errCachedDeletable = operatorRoleRules()
		if errCachedDeletable != nil {
			return
		}
		cachedDeletableTypes = deletableTypesFromRules(rules)
	})

	return cachedDeletableTypes, errCachedDeletable
}

// getDeletableTypes returns the set of deletable types for garbage collection:
// the types the operator is allowed to delete according to the operator roles
// shipped with the operator, plus the default minimal set of types the
// operator always creates for an Integration.
func (t *gcTrait) getDeletableTypes() (map[schema.GroupVersionKind]struct{}, error) {
	staticTypes, err := loadDeletableTypes()
	if err != nil {
		return nil, err
	}

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
