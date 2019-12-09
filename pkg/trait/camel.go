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

	"github.com/pkg/errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/camel"
)

// The Camel trait can be used to configure versions of Apache Camel and related libraries.
//
// +camel-k:trait=camel
type camelTrait struct {
	BaseTrait `property:",squash"`
	// The camel version to use for the integration. It overrides the default version set in the Integration Platform.
	Version string `property:"version"`
	// The camel-k-runtime version to use for the integration. It overrides the default version set in the Integration Platform.
	RuntimeVersion string `property:"runtime-version"`
}

func newCamelTrait() *camelTrait {
	return &camelTrait{
		BaseTrait: newBaseTrait("camel"),
	}
}

func (t *camelTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return true, nil
}

func (t *camelTrait) Apply(e *Environment) error {
	cv := t.determineCamelVersion(e)
	rv := t.determineRuntimeVersion(e)

	if e.CamelCatalog == nil {
		quarkus := e.Catalog.GetTrait("quarkus").(*quarkusTrait)
		if quarkus.isEnabled() {
			err := quarkus.loadOrCreateCatalog(e, cv, rv)
			if err != nil {
				return err
			}
		} else {
			err := t.loadOrCreateCatalog(e, cv, rv)
			if err != nil {
				return err
			}
		}
	}

	e.RuntimeVersion = rv

	if e.Integration != nil {
		e.Integration.Status.CamelVersion = e.CamelCatalog.Version
		e.Integration.Status.RuntimeVersion = e.CamelCatalog.RuntimeVersion
		e.Integration.Status.RuntimeProvider = e.CamelCatalog.RuntimeProvider
	}
	if e.IntegrationKit != nil {
		e.IntegrationKit.Status.CamelVersion = e.CamelCatalog.Version
		e.IntegrationKit.Status.RuntimeVersion = e.CamelCatalog.RuntimeVersion
		e.IntegrationKit.Status.RuntimeProvider = e.CamelCatalog.RuntimeProvider
	}

	return nil
}

func (t *camelTrait) loadOrCreateCatalog(e *Environment, camelVersion string, runtimeVersion string) error {
	ns := e.DetermineNamespace()
	if ns == "" {
		return errors.New("unable to determine namespace")
	}

	catalog, err := camel.LoadCatalog(e.C, e.Client, ns, camelVersion, runtimeVersion, nil)
	if err != nil {
		return err
	}

	if catalog == nil {
		// if the catalog is not found in the cluster, try to create it if
		// the required versions (camel and runtime) are not expressed as
		// semver constraints
		if exactVersionRegexp.MatchString(camelVersion) && exactVersionRegexp.MatchString(runtimeVersion) {
			catalog, err = camel.GenerateCatalog(e.C, e.Client, ns, e.Platform.Status.FullConfig.Build.Maven, camelVersion, runtimeVersion)
			if err != nil {
				return err
			}

			// sanitize catalog name
			catalogName := "camel-catalog-" + strings.ToLower(camelVersion+"-"+runtimeVersion)

			cx := v1alpha1.NewCamelCatalogWithSpecs(ns, catalogName, catalog.CamelCatalogSpec)
			cx.Labels = make(map[string]string)
			cx.Labels["app"] = "camel-k"
			cx.Labels["camel.apache.org/catalog.version"] = camelVersion
			cx.Labels["camel.apache.org/catalog.loader.version"] = camelVersion
			cx.Labels["camel.apache.org/runtime.version"] = runtimeVersion
			cx.Labels["camel.apache.org/catalog.generated"] = True

			err = e.Client.Create(e.C, &cx)
			if err != nil && !k8serrors.IsAlreadyExists(err) {
				return err
			}
		}
	}

	if catalog == nil {
		return fmt.Errorf("unable to find catalog matching version requirement: camel=%s, runtime=%s",
			camelVersion, runtimeVersion)
	}

	e.CamelCatalog = catalog

	return nil
}

func (t *camelTrait) determineCamelVersion(e *Environment) string {
	if t.Version != "" {
		return t.Version
	}
	if e.Integration != nil && e.Integration.Status.CamelVersion != "" {
		return e.Integration.Status.CamelVersion
	}
	if e.IntegrationKit != nil && e.IntegrationKit.Status.CamelVersion != "" {
		return e.IntegrationKit.Status.CamelVersion
	}
	return e.Platform.Status.FullConfig.Build.CamelVersion
}

func (t *camelTrait) determineRuntimeVersion(e *Environment) string {
	if t.RuntimeVersion != "" {
		return t.RuntimeVersion
	}
	if e.Integration != nil && e.Integration.Status.RuntimeVersion != "" {
		return e.Integration.Status.RuntimeVersion
	}
	if e.IntegrationKit != nil && e.IntegrationKit.Status.RuntimeVersion != "" {
		return e.IntegrationKit.Status.RuntimeVersion
	}
	return e.Platform.Status.FullConfig.Build.RuntimeVersion
}

// IsPlatformTrait overrides base class method
func (t *camelTrait) IsPlatformTrait() bool {
	return true
}
