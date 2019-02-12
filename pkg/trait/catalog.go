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
	"reflect"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/fatih/structs"
)

// Catalog collects all information about traits in one place
type Catalog struct {
	L               log.Logger
	tDebug          Trait
	tDependencies   Trait
	tDeployment     Trait
	tKnativeService Trait
	tKnative        Trait
	tService        Trait
	tRoute          Trait
	tIngress        Trait
	tJolokia        Trait
	tLabels         Trait
	tPrometheus     Trait
	tOwner          Trait
	tImages         Trait
	tBuilder        Trait
	tSpringBoot     Trait
	tIstio          Trait
	tEnvironment    Trait
	tClasspath      Trait
	tRest           Trait
}

// NewCatalog creates a new trait Catalog
func NewCatalog(ctx context.Context, c client.Client) *Catalog {
	catalog := Catalog{
		L:               log.Log.WithName("trait"),
		tDebug:          newDebugTrait(),
		tRest:           newRestTrait(),
		tKnative:        newKnativeTrait(),
		tDependencies:   newDependenciesTrait(),
		tDeployment:     newDeploymentTrait(),
		tKnativeService: newKnativeServiceTrait(),
		tService:        newServiceTrait(),
		tRoute:          newRouteTrait(),
		tIngress:        newIngressTrait(),
		tJolokia:        newJolokiaTrait(),
		tLabels:         newLabelsTrait(),
		tPrometheus:     newPrometheusTrait(),
		tOwner:          newOwnerTrait(),
		tImages:         newImagesTrait(),
		tBuilder:        newBuilderTrait(),
		tSpringBoot:     newSpringBootTrait(),
		tIstio:          newIstioTrait(),
		tEnvironment:    newEnvironmentTrait(),
		tClasspath:      newClasspathTrait(),
	}

	for _, t := range catalog.allTraits() {
		if ctx != nil {
			t.InjectContext(ctx)
		}
		if c != nil {
			t.InjectClient(c)
		}
	}
	return &catalog
}

func (c *Catalog) allTraits() []Trait {
	return []Trait{
		c.tDebug,
		c.tRest,
		c.tKnative,
		c.tDependencies,
		c.tDeployment,
		c.tKnativeService,
		c.tService,
		c.tRoute,
		c.tIngress,
		c.tJolokia,
		c.tLabels,
		c.tPrometheus,
		c.tOwner,
		c.tImages,
		c.tBuilder,
		c.tSpringBoot,
		c.tIstio,
		c.tEnvironment,
		c.tClasspath,
	}
}

// Traits may depend on the result of previously executed ones,
// so care must be taken while changing the lists order.
func (c *Catalog) traitsFor(environment *Environment) []Trait {
	switch environment.DetermineProfile() {
	case v1alpha1.TraitProfileOpenShift:
		return []Trait{
			c.tDebug,
			c.tRest,
			c.tDependencies,
			c.tImages,
			c.tBuilder,
			c.tEnvironment,
			c.tClasspath,
			c.tSpringBoot,
			c.tJolokia,
			c.tPrometheus,
			c.tDeployment,
			c.tService,
			c.tRoute,
			c.tOwner,
			c.tLabels,
		}
	case v1alpha1.TraitProfileKubernetes:
		return []Trait{
			c.tDebug,
			c.tRest,
			c.tDependencies,
			c.tImages,
			c.tBuilder,
			c.tEnvironment,
			c.tClasspath,
			c.tSpringBoot,
			c.tJolokia,
			c.tPrometheus,
			c.tDeployment,
			c.tService,
			c.tIngress,
			c.tOwner,
			c.tLabels,
		}
	case v1alpha1.TraitProfileKnative:
		return []Trait{
			c.tDebug,
			c.tRest,
			c.tKnative,
			c.tDependencies,
			c.tImages,
			c.tBuilder,
			c.tEnvironment,
			c.tClasspath,
			c.tSpringBoot,
			c.tDeployment,
			c.tKnativeService,
			c.tIstio,
			c.tOwner,
			c.tLabels,
		}
	}

	return nil
}

func (c *Catalog) apply(environment *Environment) error {
	if err := c.configure(environment); err != nil {
		return err
	}
	traits := c.traitsFor(environment)

	for _, trait := range traits {
		enabled, err := trait.Configure(environment)
		if err != nil {
			return err
		}

		if enabled {
			c.L.Infof("Apply trait: %s", trait.ID())

			err = trait.Apply(environment)
			if err != nil {
				return err
			}

			environment.ExecutedTraits = append(environment.ExecutedTraits, trait)
		}
	}

	for _, processor := range environment.PostProcessors {
		err := processor(environment)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetTrait returns the trait with the given ID
func (c *Catalog) GetTrait(id string) Trait {
	for _, t := range c.allTraits() {
		if t.ID() == ID(id) {
			return t
		}
	}
	return nil
}

func (c *Catalog) configure(env *Environment) error {
	if env.Context != nil && env.Context.Spec.Traits != nil {
		for id, traitSpec := range env.Context.Spec.Traits {
			catTrait := c.GetTrait(id)
			if catTrait != nil {
				if err := traitSpec.Decode(catTrait); err != nil {
					return err
				}
			}
		}
	}
	if env.Integration != nil && env.Integration.Spec.Traits != nil {
		for id, traitSpec := range env.Integration.Spec.Traits {
			catTrait := c.GetTrait(id)
			if catTrait != nil {
				if err := traitSpec.Decode(catTrait); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ComputeTraitsProperties returns all key/value configuration properties that can be used to configure traits
func (c *Catalog) ComputeTraitsProperties() []string {
	results := make([]string, 0)
	for _, trait := range c.allTraits() {
		trait := trait // pin
		c.processFields(structs.Fields(trait), func(name string) {
			results = append(results, string(trait.ID())+"."+name)
		})
	}

	return results
}

func (c *Catalog) processFields(fields []*structs.Field, processor func(string)) {
	for _, f := range fields {
		if f.IsEmbedded() && f.IsExported() && f.Kind() == reflect.Struct {
			c.processFields(f.Fields(), processor)
		}

		if f.IsEmbedded() {
			continue
		}

		property := f.Tag("property")

		if property != "" {
			items := strings.Split(property, ",")
			processor(items[0])
		}
	}
}
