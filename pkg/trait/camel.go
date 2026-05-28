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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/property"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	CamelPropertiesType = "camel-properties"
	camelTraitID        = "camel"
	camelTraitOrder     = 200
)

type camelTrait struct {
	BasePlatformTrait
	traitv1.CamelTrait `property:",squash"`

	// private configuration used only internally
	runtimeVersion  string
	runtimeProvider v1.RuntimeProvider
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

	if t.RuntimeProvider == "" {
		t.runtimeProvider = determineRuntimeProvider(e)
	} else {
		t.runtimeProvider = v1.RuntimeProvider(t.RuntimeProvider)
	}
	if t.RuntimeVersion == "" {
		t.runtimeVersion = determineRuntimeVersion(e)
	} else {
		t.runtimeVersion = t.RuntimeVersion
	}

	var cond *TraitCondition
	//nolint: staticcheck
	if (e.Integration != nil && (!e.Integration.IsManagedBuild() || e.Integration.IsGitBuild())) ||
		(e.IntegrationKit != nil && e.IntegrationKit.IsSynthetic()) {
		// We set a condition to warn the user the catalog used to run the Integration
		// may differ from the runtime version which we don't control
		cond = NewIntegrationCondition(
			"Camel",
			v1.IntegrationConditionTraitInfo,
			corev1.ConditionTrue,
			TraitConfigurationReason,
			fmt.Sprintf(
				"Operated with CamelCatalog version %s which may be different from the runtime used in the container",
				t.runtimeVersion,
			),
		)
	}

	return !e.IntegrationInPhase(v1.IntegrationPhaseBuildComplete), cond, nil
}

func (t *camelTrait) Apply(e *Environment) error {
	// This is an important action to do as most of the traits
	// expects a CamelCatalog to be loaded regardless it's a managed or
	// non managed build Integration
	if e.CamelCatalog == nil {
		if err := t.loadOrCreateCatalog(e); err != nil {
			return err
		}
	}

	if e.Integration != nil {
		if e.Integration.IsManagedBuild() && !e.Integration.IsGitBuild() {
			// If it's not managed we don't know which is the runtime running
			e.Integration.Status.RuntimeVersion = t.runtimeVersion
			e.Integration.Status.RuntimeProvider = t.runtimeProvider
		}
		e.Integration.Status.Catalog = &v1.Catalog{
			Version:  e.CamelCatalog.Runtime.Version,
			Provider: e.CamelCatalog.Runtime.Provider,
		}
	}
	if e.IntegrationKit != nil {
		//nolint: staticcheck
		if !e.IntegrationKit.IsSynthetic() {
			e.IntegrationKit.Status.RuntimeVersion = t.runtimeVersion
			e.IntegrationKit.Status.RuntimeProvider = t.runtimeProvider
		}
		e.IntegrationKit.Status.Catalog = &v1.Catalog{
			Version:  e.CamelCatalog.Runtime.Version,
			Provider: e.CamelCatalog.Runtime.Provider,
		}
	}

	if e.IntegrationInRunningPhases() {
		e.Resources.AddAll(t.computeUserProperties(e))
	}

	return nil
}

//nolint:nestif
func (t *camelTrait) loadOrCreateCatalog(e *Environment) error {
	catalogNamespace := e.DetermineCatalogNamespace()
	if catalogNamespace == "" {
		return errors.New("unable to determine namespace")
	}

	runtime := v1.RuntimeSpec{
		Version:  t.runtimeVersion,
		Provider: t.runtimeProvider,
	}
	if runtime.Provider == v1.RuntimeProviderPlainQuarkus {
		// We need this workaround to load the last existing catalog
		// TODO: this part will be subject to future refactoring
		runtime.Version = defaults.CamelKRuntimeCatalogVersion
	}

	catalog, err := camel.LoadCatalog(e.Ctx, e.Client, catalogNamespace, runtime)
	if err != nil {
		return err
	}

	if catalog == nil {
		// if the catalog is not found in the cluster, try to create it if
		// the required versions (camel and runtime) are not expressed as
		// semver constraints
		if exactVersionRegexp.MatchString(t.runtimeVersion) {
			mavenSpec := e.Platform.Maven.MavenSpec
			var extraRepositories []string
			// If the resource is targeting a specific operator, then, we must
			// provide the same setting to the catalog
			operatorId := ""
			if e.Integration != nil {
				if e.Integration.GetOperatorID() != "" {
					operatorId = e.Integration.GetOperatorID()
				}
				if e.Integration.Spec.Repositories != nil {
					extraRepositories = append(extraRepositories, e.Integration.Spec.Repositories...)
				}
			}
			if e.IntegrationKit != nil {
				if e.IntegrationKit.GetOperatorID() != "" {
					operatorId = e.IntegrationKit.GetOperatorID()
				}
				if e.IntegrationKit.Spec.Repositories != nil {
					extraRepositories = append(extraRepositories, e.IntegrationKit.Spec.Repositories...)
				}
			}
			catalog, err = camel.CreateCatalog(e.Ctx, e.Client, catalogNamespace,
				mavenSpec, platform.DefaultBuildTimeout, runtime, extraRepositories, operatorId)
			if err != nil {
				return err
			}
		}

		// If the catalog is a plain-quarkus one, then, we need to wait for it to be available
		// as the logic is that it is cloned after a legacy camel k runtime catalog.
		// NOTE: the CamelCatalog is meant to disappear anytime soon, so we can maintain this workaround
		// as long as the CamelCatalog is supported
		if runtime.Provider == v1.RuntimeProviderPlainQuarkus {
			err = wait.PollUntilContextTimeout(
				e.Ctx,
				//nolint:mnd
				2*time.Second,
				1*time.Minute,
				true,
				func(ctx context.Context) (bool, error) {
					c, err := camel.LoadCatalogFixedRuntime(
						ctx,
						e.Client,
						catalogNamespace,
						runtime,
					)

					if apierrors.IsNotFound(err) {
						return false, nil
					}

					if err != nil {
						return false, err
					}

					if c == nil {
						return false, nil
					}

					catalog = c

					return true, nil
				},
			)
		}

		if err != nil {
			return err
		}
	}

	if catalog == nil {
		return fmt.Errorf("unable to find catalog matching version requirement: runtime=%s, provider=%s",
			runtime.Version,
			runtime.Provider)
	}

	if runtime.Provider == v1.RuntimeProviderPlainQuarkus {
		// We need this workaround to load the last existing catalog
		// TODO: this part will be subject to future refactoring
		catalog.Runtime.Version = t.runtimeVersion
	}
	e.CamelCatalog = catalog

	return nil
}

func determineRuntimeVersion(e *Environment) string {
	if e.Integration != nil && e.Integration.Status.RuntimeVersion != "" {
		return e.Integration.Status.RuntimeVersion
	}
	if e.IntegrationKit != nil && e.IntegrationKit.Status.RuntimeVersion != "" {
		return e.IntegrationKit.Status.RuntimeVersion
	}
	if e.IntegrationProfile != nil && e.IntegrationProfile.Spec.Build.RuntimeVersion != "" {
		return e.IntegrationProfile.Spec.Build.RuntimeVersion
	}
	if e.Platform.BuildRuntimeVersion != "" {
		return e.Platform.BuildRuntimeVersion
	}

	return defaults.DefaultRuntimeVersion
}

func determineRuntimeProvider(e *Environment) v1.RuntimeProvider {
	if e.Integration != nil && e.Integration.Status.RuntimeProvider != "" {
		return e.Integration.Status.RuntimeProvider
	}
	if e.IntegrationKit != nil && e.IntegrationKit.Status.RuntimeProvider != "" {
		return e.IntegrationKit.Status.RuntimeProvider
	}
	if e.IntegrationProfile != nil && e.IntegrationProfile.Spec.Build.RuntimeProvider != "" {
		return e.IntegrationProfile.Spec.Build.RuntimeProvider
	}
	if e.Platform.BuildRuntimeProvider != "" {
		return e.Platform.BuildRuntimeProvider
	}

	return v1.RuntimeProvider(defaults.DefaultRuntimeProvider)
}

func (t *camelTrait) computeUserProperties(e *Environment) []ctrl.Object {
	sources := e.Integration.AllSources()
	maps := make([]ctrl.Object, 0, len(sources)+1)

	// combine properties of integration with kit, integration
	// properties have the priority
	userProperties := ""

	var userPropertiesSb238 strings.Builder
	for _, prop := range e.collectConfigurationPairs("property") {
		// properties in resource configuration are expected to be pre-encoded using properties format
		fmt.Fprintf(&userPropertiesSb238, "%s=%s\n", prop.Name, prop.Value)
	}
	userProperties += userPropertiesSb238.String()

	if t.Properties != nil {
		// Merge with properties set in the trait
		var userPropertiesSb245 strings.Builder
		for _, prop := range t.Properties {
			k, v := property.SplitPropertyFileEntry(prop)
			fmt.Fprintf(&userPropertiesSb245, "%s=%s\n", k, v)
		}
		userProperties += userPropertiesSb245.String()
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
