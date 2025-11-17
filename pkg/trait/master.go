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
	"fmt"
	"strings"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/metadata"
	"github.com/apache/camel-k/v2/pkg/resources"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/uri"
)

type masterTrait struct {
	BaseTrait
	traitv1.MasterTrait `property:",squash"`

	delegateDependencies []string
}

// NewMasterTrait --.
func NewMasterTrait() Trait {
	return &masterTrait{
		BaseTrait: NewBaseTrait("master", TraitOrderBeforeControllerCreation),
	}
}

const (
	masterComponent   = "master"
	leaseResourceType = "Lease"
)

func (t *masterTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}
	if !ptr.Deref(t.Enabled, true) {
		return false, NewIntegrationConditionUserDisabled(masterComponent), nil
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

func (t *masterTrait) Apply(e *Environment) error {
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
		}
	}

	return nil
}

func (t *masterTrait) setCatalogConfiguration(e *Environment) {
	if e.ApplicationProperties == nil {
		e.ApplicationProperties = make(map[string]string)
	}
	e.ApplicationProperties["camel.k.master.resourceName"] = *t.ResourceName
	e.ApplicationProperties["camel.k.master.resourceType"] = t.getResourceKey()
	e.ApplicationProperties["camel.k.master.labelKey"] = t.getLabelKey()
	e.ApplicationProperties["camel.k.master.labelValue"] = *t.LabelValue

	for _, cp := range e.CamelCatalog.Runtime.Capabilities["master"].RuntimeProperties {
		e.ApplicationProperties[CapabilityPropertyKey(cp.Key, e.ApplicationProperties)] = cp.Value
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

func findAdditionalDependencies(e *Environment, meta metadata.IntegrationMetadata) []string {
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
	data, err := resources.TemplateResource("resources/addons/master/"+name, params)
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
		Name:           itName + "-master",
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
