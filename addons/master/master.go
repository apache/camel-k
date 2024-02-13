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
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/metadata"
	"github.com/apache/camel-k/v2/pkg/resources"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/uri"
)

// The Master trait allows to configure the integration to automatically leverage Kubernetes resources for doing
// leader election and starting *master* routes only on certain instances.
//
// It's activated automatically when using the master endpoint in a route, e.g. `from("master:lockname:telegram:bots")...`.
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
	// Name of the configmap that will be used to store the lock. Defaults to "<integration-name>-lock".
	// Name of the configmap/lease resource that will be used to store the lock. Defaults to "<integration-name>-lock".
	ResourceName *string `property:"resource-name" json:"resourceName,omitempty"`
	// Type of Kubernetes resource to use for locking ("ConfigMap" or "Lease"). Defaults to "Lease".
	ResourceType *string `property:"resource-type" json:"resourceType,omitempty"`
	// Label that will be used to identify all pods contending the lock. Defaults to "camel.apache.org/integration".
	LabelKey *string `property:"label-key" json:"labelKey,omitempty"`
	// Label value that will be used to identify all pods contending the lock. Defaults to the integration name.
	LabelValue *string `property:"label-value" json:"labelValue,omitempty"`
}

type masterTrait struct {
	trait.BaseTrait
	Trait                `property:",squash"`
	delegateDependencies []string `json:"-"`
}

// NewMasterTrait --.
func NewMasterTrait() trait.Trait {
	return &masterTrait{
		BaseTrait: trait.NewBaseTrait("master", 590),
	}
}

const (
	masterComponent = "master"
)

var (
	leaseResourceType     = "Lease"
	configMapResourceType = "ConfigMap"
)

func (t *masterTrait) Configure(e *trait.Environment) (bool, *trait.TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, trait.NewIntegrationConditionUserDisabled(), nil
	}
	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization, v1.IntegrationPhaseBuildingKit) && !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}
	if pointer.BoolDeref(t.Auto, true) {
		// Check if the master component has been used
		sources, err := kubernetes.ResolveIntegrationSources(e.Ctx, t.Client, e.Integration, e.Resources)
		if err != nil {
			return false, nil, err
		}

		meta, err := metadata.ExtractAll(e.CamelCatalog, sources)
		if err != nil {
			return false, nil, err
		}

		if t.Enabled == nil {
			for _, endpoint := range meta.FromURIs {
				if uri.GetComponent(endpoint) == masterComponent {
					enabled := true
					t.Enabled = &enabled
				}
			}
			// No master component, can skip the trait execution
			if !pointer.BoolDeref(t.Enabled, false) {
				return false, nil, nil
			}
		}

		if t.IncludeDelegateDependencies == nil || *t.IncludeDelegateDependencies {
			t.delegateDependencies = findAdditionalDependencies(e, meta)
		}

		if t.ResourceName == nil {
			val := fmt.Sprintf("%s-lock", e.Integration.Name)
			t.ResourceName = &val
		}

		if t.ResourceType == nil {
			canUseLeases, err := t.canUseLeases(e)
			if err != nil {
				return false, nil, err
			}
			if canUseLeases {
				t.ResourceType = &leaseResourceType
			} else {
				t.ResourceType = &configMapResourceType
			}
		}

		if t.LabelKey == nil {
			val := v1.IntegrationLabel
			t.LabelKey = &val
		}

		if t.LabelValue == nil {
			t.LabelValue = &e.Integration.Name
		}
	}

	return pointer.BoolDeref(t.Enabled, false), nil, nil
}

func (t *masterTrait) Apply(e *trait.Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, v1.CapabilityMaster)

		// Master sub endpoints need to be added to the list of dependencies
		for _, dep := range t.delegateDependencies {
			util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, dep)
		}
	} else if e.IntegrationInRunningPhases() {
		// Master trait requires the ServiceAccount certain privileges
		privileges, err := t.prepareRBAC(e.Client, e.Integration.Spec.ServiceAccountName, e.Integration.Name, e.Integration.Namespace)
		if err != nil {
			return err
		}
		// Add the RBAC privileges
		e.Resources.AddAll(privileges)
	}

	builderTraitUpdated := t.updateBuilderTrait(e.Integration.Spec.Traits.Builder)
	e.Integration.Spec.Traits.Builder = builderTraitUpdated

	return nil
}

func (t *masterTrait) updateBuilderTrait(bt *traitv1.BuilderTrait) *traitv1.BuilderTrait {
	if bt == nil {
		bt = &traitv1.BuilderTrait{}
	}
	if bt.Properties != nil {
		bt.Properties = make([]string, 0)
	}
	bt.Properties = append(bt.Properties, "quarkus.camel.cluster.kubernetes.enabled=true")
	if t.ResourceName != nil {
		bt.Properties = append(bt.Properties, fmt.Sprintf("quarkus.camel.cluster.kubernetes.resource-name=%s", *t.ResourceName))
	}
	if t.ResourceType != nil {
		bt.Properties = append(bt.Properties, fmt.Sprintf("quarkus.camel.cluster.kubernetes.lease-resource-type=%s", *t.ResourceType))
	}
	if t.LabelKey != nil {
		if t.LabelValue != nil {
			bt.Properties = append(bt.Properties, fmt.Sprintf("quarkus.camel.cluster.kubernetes.labels.\"%s\"=%s", *t.LabelKey, *t.LabelValue))
		}
	}

	return bt
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

func loadResource(cli client.Client, name string, params interface{}) (ctrl.Object, error) {
	data, err := resources.TemplateResource(fmt.Sprintf("resources/addons/master/%s", name), params)
	if err != nil {
		return nil, err
	}
	obj, err := kubernetes.LoadResourceFromYaml(cli.GetScheme(), data)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (t *masterTrait) prepareRBAC(cli client.Client, serviceAccount, itName, itNamespace string) ([]ctrl.Object, error) {
	objs := make([]ctrl.Object, 0, 2)
	if serviceAccount == "" {
		serviceAccount = "default"
	}

	templateData := struct {
		Namespace      string
		Name           string
		ServiceAccount string
	}{
		Namespace:      itNamespace,
		Name:           fmt.Sprintf("%s-master", itName),
		ServiceAccount: serviceAccount,
	}

	roleSuffix := leaseResourceType
	if t.ResourceType != nil {
		roleSuffix = *t.ResourceType
	}
	roleSuffix = strings.ToLower(roleSuffix)

	role, err := loadResource(cli, fmt.Sprintf("master-role-%s.tmpl", roleSuffix), templateData)
	if err != nil {
		return nil, err
	}
	objs = append(objs, role)
	roleBinding, err := loadResource(cli, "master-role-binding.tmpl", templateData)
	if err != nil {
		return nil, err
	}
	objs = append(objs, roleBinding)
	return objs, nil
}
