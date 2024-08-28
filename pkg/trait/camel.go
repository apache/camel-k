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
	"errors"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/property"
)

const (
	CamelPropertiesType = "camel-properties"
	camelTraitID        = "camel"
	camelTraitOrder     = 200
)

type camelTrait struct {
	BasePlatformTrait
	traitv1.CamelTrait `property:",squash"`
}

func newCamelTrait() Trait {
	return &camelTrait{
		BasePlatformTrait: NewBasePlatformTrait(camelTraitID, camelTraitOrder),
	}
}

// InfluencesKit overrides base class method.
func (t *camelTrait) InfluencesKit() bool {
	return true
}

func (t *camelTrait) Matches(trait Trait) bool {
	otherTrait, ok := trait.(*camelTrait)
	if !ok {
		return false
	}

	return otherTrait.RuntimeVersion == t.RuntimeVersion
}

func (t *camelTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration != nil && e.Integration.IsSynthetic() {
		return false, NewIntegrationConditionPlatformDisabledWithMessage("Camel", "synthetic integration"), nil
	}

	if t.RuntimeVersion == "" {
		if runtimeVersion, err := determineRuntimeVersion(e); err != nil {
			return false, nil, err
		} else {
			t.RuntimeVersion = runtimeVersion
		}
	}

	var cond *TraitCondition
	if (e.Integration != nil && !e.Integration.IsManagedBuild()) || (e.IntegrationKit != nil && e.IntegrationKit.IsSynthetic()) {
		// We set a condition to warn the user the catalog used to run the Integration
		// may differ from the runtime version which we don't control
		cond = NewIntegrationCondition(
			"Camel",
			v1.IntegrationConditionTraitInfo,
			corev1.ConditionTrue,
			traitConfigurationReason,
			fmt.Sprintf(
				"Operated with CamelCatalog version %s which may be different from the runtime used in the container",
				t.RuntimeVersion,
			),
		)
	}

	return true, cond, nil
}

func (t *camelTrait) Apply(e *Environment) error {
	// This is an important action to do as most of the traits
	// expects a CamelCatalog to be loaded regardless it's a managed or
	// non managed build Integration
	if e.CamelCatalog == nil {
		if err := t.loadOrCreateCatalog(e, t.RuntimeVersion); err != nil {
			return err
		}
	}

	if e.Integration != nil {
		if e.Integration.IsManagedBuild() {
			// If it's not managed we don't know which is the runtime running
			e.Integration.Status.RuntimeVersion = e.CamelCatalog.Runtime.Version
			e.Integration.Status.RuntimeProvider = e.CamelCatalog.Runtime.Provider
		}
		e.Integration.Status.Catalog = &v1.Catalog{
			Version:  e.CamelCatalog.Runtime.Version,
			Provider: e.CamelCatalog.Runtime.Provider,
		}
	}
	if e.IntegrationKit != nil {
		if !e.IntegrationKit.IsSynthetic() {
			e.IntegrationKit.Status.RuntimeVersion = e.CamelCatalog.Runtime.Version
			e.IntegrationKit.Status.RuntimeProvider = e.CamelCatalog.Runtime.Provider
		}
		e.IntegrationKit.Status.Catalog = &v1.Catalog{
			Version:  e.CamelCatalog.Runtime.Version,
			Provider: e.CamelCatalog.Runtime.Provider,
		}
	}

	if e.IntegrationInRunningPhases() {
		// Get all resources
		maps := t.computeConfigMaps(e)
		e.Resources.AddAll(maps)
	}
	return nil
}

func (t *camelTrait) loadOrCreateCatalog(e *Environment, runtimeVersion string) error {
	catalogNamespace := e.DetermineCatalogNamespace()
	if catalogNamespace == "" {
		return errors.New("unable to determine namespace")
	}

	runtime := v1.RuntimeSpec{
		Version:  runtimeVersion,
		Provider: v1.RuntimeProviderQuarkus,
	}

	catalog, err := camel.LoadCatalog(e.Ctx, e.Client, catalogNamespace, runtime)
	if err != nil {
		return err
	}

	if catalog == nil {
		// if the catalog is not found in the cluster, try to create it if
		// the required versions (camel and runtime) are not expressed as
		// semver constraints
		if exactVersionRegexp.MatchString(runtimeVersion) {
			catalog, err = camel.CreateCatalog(e.Ctx, e.Client, catalogNamespace, e.Platform, runtime)
			if err != nil {
				return err
			}
		}
	}

	if catalog == nil {
		return fmt.Errorf("unable to find catalog matching version requirement: runtime=%s, provider=%s",
			runtime.Version,
			runtime.Provider)
	}

	e.CamelCatalog = catalog

	return nil
}

func (t *camelTrait) computeConfigMaps(e *Environment) []ctrl.Object {
	sources := e.Integration.AllSources()
	maps := make([]ctrl.Object, 0, len(sources)+1)

	// combine properties of integration with kit, integration
	// properties have the priority
	userProperties := ""

	for _, prop := range e.collectConfigurationPairs("property") {
		// properties in resource configuration are expected to be pre-encoded using properties format
		userProperties += fmt.Sprintf("%s=%s\n", prop.Name, prop.Value)
	}

	if t.Properties != nil {
		// Merge with properties set in the trait
		for _, prop := range t.Properties {
			k, v := property.SplitPropertyFileEntry(prop)
			userProperties += fmt.Sprintf("%s=%s\n", k, v)
		}
	}

	if userProperties != "" {
		maps = append(
			maps,
			&corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      e.Integration.Name + "-user-properties",
					Namespace: e.Integration.Namespace,
					Labels: map[string]string{
						v1.IntegrationLabel:                e.Integration.Name,
						"camel.apache.org/properties.type": "user",
						kubernetes.ConfigMapTypeLabel:      CamelPropertiesType,
					},
				},
				Data: map[string]string{
					"application.properties": userProperties,
				},
			},
		)
	}

	i := 0
	for _, s := range sources {
		if s.ContentRef != "" || e.isEmbedded(s) || s.IsGeneratedFromKamelet() {
			continue
		}

		cm := corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-source-%03d", e.Integration.Name, i),
				Namespace: e.Integration.Namespace,
				Labels: map[string]string{
					v1.IntegrationLabel: e.Integration.Name,
				},
				Annotations: map[string]string{
					sourceLanguageAnnotation:    string(s.InferLanguage()),
					sourceLoaderAnnotation:      s.Loader,
					sourceNameAnnotation:        s.Name,
					sourceCompressionAnnotation: strconv.FormatBool(s.Compression),
				},
			},
			Data: map[string]string{
				"content": s.Content,
			},
		}

		maps = append(maps, &cm)
		i++
	}

	return maps
}

func determineRuntimeVersion(e *Environment) (string, error) {
	if e.Integration != nil && e.Integration.Status.RuntimeVersion != "" {
		return e.Integration.Status.RuntimeVersion, nil
	}
	if e.IntegrationKit != nil && e.IntegrationKit.Status.RuntimeVersion != "" {
		return e.IntegrationKit.Status.RuntimeVersion, nil
	}
	if e.IntegrationProfile != nil && e.IntegrationProfile.Status.Build.RuntimeVersion != "" {
		return e.IntegrationProfile.Status.Build.RuntimeVersion, nil
	}
	if e.Platform != nil && e.Platform.Status.Build.RuntimeVersion != "" {
		return e.Platform.Status.Build.RuntimeVersion, nil
	}
	return "", errors.New("unable to determine runtime version")
}
