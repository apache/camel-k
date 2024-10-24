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

	"k8s.io/utils/ptr"
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
	delegateDependencies []string
}

// NewMasterTrait --.
func NewMasterTrait() trait.Trait {
	return &masterTrait{
		BaseTrait: trait.NewBaseTrait("master", trait.TraitOrderBeforeControllerCreation),
	}
}

const (
	masterComponent   = "master"
	leaseResourceType = "Lease"
)

func (t *masterTrait) Configure(e *trait.Environment) (bool, *trait.TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}
	if !ptr.Deref(t.Enabled, true) {
		return false, trait.NewIntegrationConditionUserDisabled(masterComponent), nil
	}
	if !e.IntegrationInPhase(v1.IntegrationPhaseInitialization, v1.IntegrationPhaseBuildingKit) && !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}
	if !ptr.Deref(t.Auto, true) {
		return ptr.Deref(t.Enabled, false), nil, nil
	}

	enabled, err := e.ConsumeMeta(false, func(meta metadata.IntegrationMetadata) bool {
		found := false
	loop:
		for _, endpoint := range meta.FromURIs {
			if uri.GetComponent(endpoint) == masterComponent {
				found = true
				break loop
			}
		}
		if found {
			if ptr.Deref(t.IncludeDelegateDependencies, true) {
				t.delegateDependencies = findAdditionalDependencies(e, meta)
			}
		}

		return found
	})

	if err != nil {
		return false, nil, err
	}
	if enabled {
		t.Enabled = ptr.To(enabled)
		if t.ResourceName == nil {
			val := e.Integration.Name + "-lock"
			t.ResourceName = &val
		}
		if t.LabelValue == nil {
			t.LabelValue = &e.Integration.Name
		}
	}

	return enabled, nil, nil
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

		if e.CamelCatalog.Runtime.Capabilities["master"].RuntimeProperties != nil {
			t.setCatalogConfiguration(e)
		} else {
			t.setCustomizerConfiguration(e)
		}
	}

	return nil
}

// Deprecated: to be removed in future release in favor of func setCatalogConfiguration().
func (t *masterTrait) setCustomizerConfiguration(e *trait.Environment) {
	e.Integration.Status.Configuration = append(e.Integration.Status.Configuration,
		v1.ConfigurationSpec{Type: "property", Value: "customizer.master.enabled=true"},
	)
	e.Integration.Status.Configuration = append(e.Integration.Status.Configuration,
		v1.ConfigurationSpec{Type: "property", Value: "customizer.master.kubernetesResourceNames=" + *t.ResourceName},
	)
	e.Integration.Status.Configuration = append(e.Integration.Status.Configuration,
		v1.ConfigurationSpec{Type: "property", Value: "customizer.master.leaseResourceType" + t.getResourceKey()},
	)
	e.Integration.Status.Configuration = append(e.Integration.Status.Configuration,
		v1.ConfigurationSpec{Type: "property", Value: "customizer.master.labelKey=" + t.getLabelKey()},
	)
	e.Integration.Status.Configuration = append(e.Integration.Status.Configuration,
		v1.ConfigurationSpec{Type: "property", Value: "customizer.master.labelValue=" + *t.LabelValue},
	)
}

func (t *masterTrait) setCatalogConfiguration(e *trait.Environment) {
	if e.ApplicationProperties == nil {
		e.ApplicationProperties = make(map[string]string)
	}
	e.ApplicationProperties["camel.k.master.resourceName"] = *t.ResourceName
	e.ApplicationProperties["camel.k.master.resourceType"] = t.getResourceKey()
	e.ApplicationProperties["camel.k.master.labelKey"] = t.getLabelKey()
	e.ApplicationProperties["camel.k.master.labelValue"] = *t.LabelValue

	for _, cp := range e.CamelCatalog.Runtime.Capabilities["master"].RuntimeProperties {
		e.ApplicationProperties[trait.CapabilityPropertyKey(cp.Key, e.ApplicationProperties)] = cp.Value
	}
}

func (t *masterTrait) getResourceKey() string {
	if t.ResourceType == nil {
		return leaseResourceType
	}

	return *t.ResourceType
}

func (t *masterTrait) getLabelKey() string {
	if t.LabelKey == nil {
		return v1.IntegrationLabel
	}

	return *t.LabelKey
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
