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
	"sort"

	"github.com/pkg/errors"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/dsl"
)

const flowsInternalSourceName = "camel-k-embedded-flow.yaml"

// Internal trait
type initTrait struct {
	BaseTrait `property:",squash"`
}

func newInitTrait() Trait {
	return &initTrait{
		BaseTrait: NewBaseTrait("init", 1),
	}
}

func (t *initTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, errors.New("trait init cannot be disabled")
	}

	return true, nil
}

func (t *initTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {

		// Flows need to be turned into a generated source
		if len(e.Integration.Spec.Flows) > 0 {
			content, err := dsl.ToYamlDSL(e.Integration.Spec.Flows)
			if err != nil {
				return err
			}
			e.Integration.Status.AddOrReplaceGeneratedSources(v1.SourceSpec{
				DataSpec: v1.DataSpec{
					Name:    flowsInternalSourceName,
					Content: string(content),
				},
			})
		}

		//
		// Dependencies need to be recomputed in case of a trait declares a capability but as
		// the dependencies trait runs earlier than some task such as the cron one, we need to
		// register a post step processor that recompute the dependencies based on the declared
		// capabilities
		//
		e.PostStepProcessors = append(e.PostStepProcessors, func(environment *Environment) error {
			//
			// The camel catalog is set-up by the camel trait so it may not be available for
			// traits executed before that one
			//
			if e.CamelCatalog != nil {
				// add runtime specific dependencies
				for _, capability := range e.Integration.Status.Capabilities {
					for _, dependency := range e.CamelCatalog.Runtime.CapabilityDependencies(capability) {
						util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, dependency.GetDependencyID())
					}
				}
			}

			if e.Integration.Status.Dependencies != nil {
				// sort the dependencies to get always the same list if they don't change
				sort.Strings(e.Integration.Status.Dependencies)
			}

			return nil
		})
	}

	return nil
}
