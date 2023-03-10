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

package master

import (
	"fmt"
	"strings"

	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/metadata"
	"github.com/apache/camel-k/v2/pkg/resources"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/property"
	"github.com/apache/camel-k/v2/pkg/util/uri"
)

// The Master trait allows to configure the integration to automatically leverage Kubernetes resources for doing
// leader election and starting *master* routes only on certain instances.
//
// It's activated automatically when using the master endpoint in a route, e.g. `from("master:lockname:telegram:bots")...`.
//
// This trait is backed by camel-quarkus-kubernetes component and it requires build time properties if your integration needs customization.
//
// These are the three most commonly used parameters:
//
// `quarkus.camel.cluster.kubernetes.resource-name`: The name of the lease resource used to do optimistic locking (defaults to 'leaders').
//
// `quarkus.camel.cluster.kubernetes.lease-resource-type`: The lease resource type used in Kubernetes, either `ConfigMap` or `Lease`` (defaults to `Lease`).
//
// `quarkus.camel.cluster.kubernetes.labels`: The labels key/value used to identify the pods composing the cluster, defaults to empty map.
//
// The parameters must be set with `--build-property`, example: `--build-property quarkus.camel.cluster.kubernetes.resource-name=foobar`
//
// NOTE: this trait adds special permissions to the integration service account in order to read/write configmaps and read pods.
// It's recommended to use a different service account than "default" when running the integration.
//
// +camel-k:trait=master.
type Trait struct {
	traitv1.Trait `property:",squash" json:",inline"`
	// Enables automatic configuration of the trait.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// When this flag is active, the operator analyzes the source code to add dependencies required by delegate endpoints.
	// E.g. when using `master:lockname:timer`, then `camel:timer` is automatically added to the set of dependencies.
	// It's enabled by default.
	IncludeDelegateDependencies *bool `property:"include-delegate-dependencies" json:"includeDelegateDependencies,omitempty"`
}

type masterTrait struct {
	trait.BaseTrait
	Trait                `property:",squash"`
	delegateDependencies []string `json:"-"`
	resourceType         string
}

// NewMasterTrait --.
func NewMasterTrait() trait.Trait {
	return &masterTrait{
		BaseTrait: trait.NewBaseTrait("master", trait.TraitOrderBeforeControllerCreation),
	}
}

const (
	masterComponent = "master"
)

var (
	leaseResourceType     = "Lease"
	configMapResourceType = "ConfigMap"
)

func (t *masterTrait) Configure(e *trait.Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() && !e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted) {
		return false, nil
	}

	if e.Integration != nil {
		if pointer.BoolDeref(t.Auto, true) {
			// Check if the master component has been used
			sources, err := kubernetes.ResolveIntegrationSources(e.Ctx, t.Client, e.Integration, e.Resources)
			if err != nil {
				return false, err
			}

			meta, err := metadata.ExtractAll(e.CamelCatalog, sources)
			if err != nil {
				return false, err
			}

			if t.Enabled == nil {
				for _, endpoint := range meta.FromURIs {
					if uri.GetComponent(endpoint) == masterComponent {
						enabled := true
						t.Enabled = &enabled
					}
				}
			}

			if !pointer.BoolDeref(t.Enabled, false) {
				return false, nil
			}

			if t.IncludeDelegateDependencies == nil || *t.IncludeDelegateDependencies {
				t.delegateDependencies = findAdditionalDependencies(e, meta)
			}
		}
	}
	if e.IntegrationInRunningPhases() {

		if e.Integration.Spec.Traits.Builder != nil && e.Integration.Spec.Traits.Builder.Properties != nil {
			for _, v := range e.Integration.Spec.Traits.Builder.Properties {
				key, value := property.SplitPropertyFileEntry(v)
				if len(key) == 0 || len(value) == 0 {
					t.L.Infof("maven property must have key=value format, it was %v", v)
				}
				if key == "quarkus.camel.cluster.kubernetes.lease-resource-type" {
					t.resourceType = value
				}
			}
		}

		if t.resourceType == "" {
			canUseLeases, err := t.canUseLeases(e)
			if err != nil {
				return false, err
			}
			if canUseLeases {
				t.resourceType = leaseResourceType
			} else {
				t.resourceType = configMapResourceType
			}
		}
	}

	return pointer.BoolDeref(t.Enabled, true), nil
}

func (t *masterTrait) Apply(e *trait.Environment) error {

	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		// util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityMaster)
		// Master sub endpoints need to be added to the list of dependencies
		for _, dep := range t.delegateDependencies {
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, dep)
		}
	} else if e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted) {
		// if the master trait is enabled, then set this camel-quarkus-kubernetes build property
		e.BuildProperties["quarkus.camel.cluster.kubernetes.enabled"] = "true"

	} else if e.IntegrationInRunningPhases() {
		serviceAccount := e.Integration.Spec.ServiceAccountName
		if serviceAccount == "" {
			serviceAccount = "default"
		}

		templateData := struct {
			Namespace      string
			Name           string
			ServiceAccount string
		}{
			Namespace:      e.Integration.Namespace,
			Name:           fmt.Sprintf("%s-master", e.Integration.Name),
			ServiceAccount: serviceAccount,
		}

		roleSuffix := leaseResourceType
		if t.resourceType != "" {
			roleSuffix = t.resourceType
		}
		roleSuffix = strings.ToLower(roleSuffix)

		role, err := loadResource(e, fmt.Sprintf("master-role-%s.tmpl", roleSuffix), templateData)
		if err != nil {
			return err
		}
		roleBinding, err := loadResource(e, "master-role-binding.tmpl", templateData)
		if err != nil {
			return err
		}

		e.Resources.Add(role)
		e.Resources.Add(roleBinding)
	}

	return nil
}

func (t *masterTrait) canUseLeases(e *trait.Environment) (bool, error) {
	return kubernetes.CheckPermission(e.Ctx, t.Client, "coordination.k8s.io", "leases", e.Integration.Namespace, "", "create")
}

func findAdditionalDependencies(e *trait.Environment, meta metadata.IntegrationMetadata) []string {
	var dependencies []string
	for _, endpoint := range meta.FromURIs {
		if uri.GetComponent(endpoint) == masterComponent {
			parts := strings.Split(endpoint, ":")
			if len(parts) > 2 {
				// syntax "master:lockname:endpoint:..."
				childComponent := strings.ReplaceAll(parts[2], "/", "")
				if artifact := e.CamelCatalog.GetArtifactByScheme(childComponent); artifact != nil {
					dependencies = append(dependencies, artifact.GetDependencyID())
					dependencies = append(dependencies, artifact.GetConsumerDependencyIDs(childComponent)...)
				}
			}
		}
	}
	return dependencies
}

func loadResource(e *trait.Environment, name string, params interface{}) (ctrl.Object, error) {
	data, err := resources.TemplateResource(fmt.Sprintf("/addons/master/%s", name), params)
	if err != nil {
		return nil, err
	}
	obj, err := kubernetes.LoadResourceFromYaml(e.Client.GetScheme(), data)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
