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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/scylladb/go-set/strset"
)

// The Dependencies trait is internally used to automatically add runtime dependencies based on the
// integration that the user wants to run.
//
// +camel-k:trait=dependencies
type dependenciesTrait struct {
	BaseTrait `property:",squash"`
}

func newDependenciesTrait() Trait {
	return &dependenciesTrait{
		BaseTrait: NewBaseTrait("dependencies", 500),
	}
}

func (t *dependenciesTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
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

	// add runtime specific dependencies
	for _, d := range e.CamelCatalog.Runtime.Dependencies {
		dependencies.Add(fmt.Sprintf("mvn:%s/%s", d.GroupID, d.ArtifactID))
	}

	sources, err := kubernetes.ResolveIntegrationSources(e.C, e.Client, e.Integration, e.Resources)
	if err != nil {
		return err
	}
	for _, s := range sources {
		meta := metadata.Extract(e.CamelCatalog, s)
		lang := s.InferLanguage()

		// add auto-detected dependencies
		dependencies.Merge(meta.Dependencies)

		for loader, v := range e.CamelCatalog.Loaders {
			// add loader specific dependencies
			if s.Loader != "" && s.Loader == loader {
				dependencies.Add(fmt.Sprintf("mvn:%s/%s", v.GroupID, v.ArtifactID))

				for _, d := range v.Dependencies {
					dependencies.Add(fmt.Sprintf("mvn:%s/%s", d.GroupID, d.ArtifactID))
				}
			} else if s.Loader == "" {
				// add language specific dependencies
				if util.StringSliceExists(v.Languages, string(lang)) {
					dependencies.Add(fmt.Sprintf("mvn:%s/%s", v.GroupID, v.ArtifactID))

					for _, d := range v.Dependencies {
						dependencies.Add(fmt.Sprintf("mvn:%s/%s", d.GroupID, d.ArtifactID))
					}
				}
			}
		}

		meta.RequiredCapabilities.Each(func(item string) bool {
			util.StringSliceUniqueAdd(&e.Integration.Status.Capabilities, item)
			return true
		})
	}

	// add dependencies back to integration
	dependencies.Each(func(item string) bool {
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, item)
		return true
	})

	return nil
}

// IsPlatformTrait overrides base class method
func (t *dependenciesTrait) IsPlatformTrait() bool {
	return true
}
