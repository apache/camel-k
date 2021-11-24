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
	"github.com/scylladb/go-set/strset"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// The Dependencies trait is internally used to automatically add runtime dependencies based on the
// integration that the user wants to run.
//
// +camel-k:trait=dependencies.
type dependenciesTrait struct {
	BaseTrait `property:",squash"`
}

func newDependenciesTrait() Trait {
	return &dependenciesTrait{
		BaseTrait: NewBaseTrait("dependencies", 500),
	}
}

func (t *dependenciesTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		return false, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseInitialization), nil
}

func (t *dependenciesTrait) Apply(e *Environment) error {
	if e.Integration.Status.Dependencies == nil {
		e.Integration.Status.Dependencies = make([]string, 0)
	}

	dependencies := strset.New()

	if e.Integration.Spec.Dependencies != nil {
		dependencies.Add(e.Integration.Spec.Dependencies...)
	}

	// Add runtime specific dependencies
	for _, d := range e.CamelCatalog.Runtime.Dependencies {
		dependencies.Add(d.GetDependencyID())
	}

	sources, err := kubernetes.ResolveIntegrationSources(e.Ctx, e.Client, e.Integration, e.Resources)
	if err != nil {
		return err
	}
	for _, s := range sources {
		// Add source-related dependencies
		dependencies.Merge(AddSourceDependencies(s, e.CamelCatalog))

		meta := metadata.Extract(e.CamelCatalog, s)
		meta.RequiredCapabilities.Each(func(item string) bool {
			util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, item)
			return true
		})
	}

	// Add dependencies back to integration
	dependencies.Each(func(item string) bool {
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, item)
		return true
	})

	return nil
}

// IsPlatformTrait overrides base class method.
func (t *dependenciesTrait) IsPlatformTrait() bool {
	return true
}
