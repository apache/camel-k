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
	"strings"

	"github.com/pkg/errors"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/maven"
)

// The Camel trait can be used to configure versions of Apache Camel K runtime and related libraries, it cannot be disabled.
//
// +camel-k:trait=camel
type camelTrait struct {
	BaseTrait `property:",squash"`
	// The camel-k-runtime version to use for the integration. It overrides the default version set in the Integration Platform.
	RuntimeVersion string `property:"runtime-version" json:"runtimeVersion,omitempty"`
}

func newCamelTrait() Trait {
	return &camelTrait{
		BaseTrait: NewBaseTrait("camel", 200),
	}
}

func (t *camelTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		return false, errors.New("trait camel cannot be disabled")
	}

	return true, nil
}

func (t *camelTrait) Apply(e *Environment) error {
	rv := t.determineRuntimeVersion(e)

	if e.CamelCatalog == nil {
		err := t.loadOrCreateCatalog(e, rv)
		if err != nil {
			return err
		}
	}

	e.RuntimeVersion = rv

	if e.Integration != nil {
		e.Integration.Status.RuntimeVersion = e.CamelCatalog.Runtime.Version
		e.Integration.Status.RuntimeProvider = e.CamelCatalog.Runtime.Provider
	}
	if e.IntegrationKit != nil {
		e.IntegrationKit.Status.RuntimeVersion = e.CamelCatalog.Runtime.Version
		e.IntegrationKit.Status.RuntimeProvider = e.CamelCatalog.Runtime.Provider
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
			catalog, err = camel.GenerateCatalog(ctx, e.Client, ns, e.Platform.Status.Build.Maven, runtime, []maven.Dependency{})
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

			err = e.Client.Create(e.Ctx, &cx)
			if err != nil {
				return errors.Wrapf(err, "unable to create catalog runtime=%s, provider=%s, name=%s",
					runtime.Version,
					runtime.Provider,
					catalogName)
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
	return e.Platform.Status.Build.RuntimeVersion
}

// IsPlatformTrait overrides base class method
func (t *camelTrait) IsPlatformTrait() bool {
	return true
}
