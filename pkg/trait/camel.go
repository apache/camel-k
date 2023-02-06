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
	"strconv"
	"strings"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/property"
)

type camelTrait struct {
	BaseTrait
	traitv1.CamelTrait `property:",squash"`
}

func newCamelTrait() Trait {
	return &camelTrait{
		BaseTrait: NewBaseTrait("camel", 200),
	}
}

// IsPlatformTrait overrides base class method.
func (t *camelTrait) IsPlatformTrait() bool {
	return true
}

// InfluencesKit overrides base class method.
func (t *camelTrait) InfluencesKit() bool {
	return true
}

func (t *camelTrait) Configure(e *Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, errors.New("trait camel cannot be disabled")
	}

	if t.RuntimeVersion == "" {
		t.RuntimeVersion = determineRuntimeVersion(e)
	}

	return true, nil
}

func (t *camelTrait) Apply(e *Environment) error {
	if t.RuntimeVersion == "" {
		return errors.New("unable to determine runtime version")
	}

	if e.CamelCatalog == nil {
		if err := t.loadOrCreateCatalog(e, t.RuntimeVersion); err != nil {
			return err
		}
	}

	e.RuntimeVersion = t.RuntimeVersion

	if e.Integration != nil {
		e.Integration.Status.RuntimeVersion = e.CamelCatalog.Runtime.Version
		e.Integration.Status.RuntimeProvider = e.CamelCatalog.Runtime.Provider
	}
	if e.IntegrationKit != nil {
		e.IntegrationKit.Status.RuntimeVersion = e.CamelCatalog.Runtime.Version
		e.IntegrationKit.Status.RuntimeProvider = e.CamelCatalog.Runtime.Provider
	}

	if e.IntegrationKitInPhase(v1.IntegrationKitPhaseReady) && e.IntegrationInRunningPhases() {
		// Get all resources
		maps := t.computeConfigMaps(e)
		e.Resources.AddAll(maps)
	}

	return nil
}

func (t *camelTrait) loadOrCreateCatalog(e *Environment, runtimeVersion string) error {
	ns := e.DetermineCatalogNamespace()
	if ns == "" {
		return errors.New("unable to determine namespace")
	}

	runtime := v1.RuntimeSpec{
		Version:  runtimeVersion,
		Provider: v1.RuntimeProviderQuarkus,
	}

	catalog, err := camel.LoadCatalog(e.Ctx, e.Client, ns, runtime)
	if err != nil {
		return err
	}

	if catalog == nil {
		// if the catalog is not found in the cluster, try to create it if
		// the required versions (camel and runtime) are not expressed as
		// semver constraints
		if exactVersionRegexp.MatchString(runtimeVersion) {
			ctx, cancel := context.WithTimeout(e.Ctx, e.Platform.Status.Build.GetTimeout().Duration)
			defer cancel()
			catalog, err = camel.GenerateCatalog(ctx, e.Client,
				ns, e.Platform.Status.Build.Maven, runtime, []maven.Dependency{})
			if err != nil {
				return err
			}

			// sanitize catalog name
			catalogName := "camel-catalog-" + strings.ToLower(runtimeVersion) + "-" + string(runtime.Provider)

			cx := v1.NewCamelCatalogWithSpecs(ns, catalogName, catalog.CamelCatalogSpec)
			cx.Labels = make(map[string]string)
			cx.Labels["app"] = "camel-k"
			cx.Labels["camel.apache.org/runtime.version"] = runtime.Version
			cx.Labels["camel.apache.org/runtime.provider"] = string(runtime.Provider)
			cx.Labels["camel.apache.org/catalog.generated"] = True

			if err := e.Client.Create(e.Ctx, &cx); err != nil {
				if k8serrors.IsAlreadyExists(err) {
					// It's still possible that catalog wasn't yet found at the time of loading
					// but then created in the background before the client tries to create it.
					// In this case, simply try loading again and reuse the existing catalog.
					catalog, err = camel.LoadCatalog(e.Ctx, e.Client, ns, runtime)
					if err != nil {
						// unexpected error
						return errors.Wrapf(err, "catalog %q already exists but unable to load", catalogName)
					}
				} else {
					return errors.Wrapf(err, "unable to create catalog runtime=%s, provider=%s, name=%s",
						runtime.Version,
						runtime.Provider,
						catalogName)
				}
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
	sources := e.Integration.Sources()
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
					},
				},
				Data: map[string]string{
					"application.properties": userProperties,
				},
			},
		)
	}

	for i, s := range sources {
		if s.ContentRef != "" {
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
					"camel.apache.org/source.language":    string(s.InferLanguage()),
					"camel.apache.org/source.loader":      s.Loader,
					"camel.apache.org/source.name":        s.Name,
					"camel.apache.org/source.compression": strconv.FormatBool(s.Compression),
				},
			},
			Data: map[string]string{
				"content": s.Content,
			},
		}

		maps = append(maps, &cm)
	}

	return maps
}

func determineRuntimeVersion(e *Environment) string {
	if e.Integration != nil && e.Integration.Status.RuntimeVersion != "" {
		return e.Integration.Status.RuntimeVersion
	}
	if e.IntegrationKit != nil && e.IntegrationKit.Status.RuntimeVersion != "" {
		return e.IntegrationKit.Status.RuntimeVersion
	}
	if e.Platform != nil && e.Platform.Status.Build.RuntimeVersion != "" {
		return e.Platform.Status.Build.RuntimeVersion
	}
	return ""
}
