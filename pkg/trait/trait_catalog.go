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
	L                 log.Logger
	tAffinity         Trait
	tCamel            Trait
	tDebug            Trait
	tDependencies     Trait
	tDeployer         Trait
	tDeployment       Trait
	tGarbageCollector Trait
	tKnativeService   Trait
	tKnative          Trait
	tService          Trait
	tRoute            Trait
	tIngress          Trait
	tJolokia          Trait
	tPrometheus       Trait
	tOwner            Trait
	tBuilder          Trait
	tIstio            Trait
	tEnvironment      Trait
	tClasspath        Trait
	tRestDsl          Trait
	tProbes           Trait
	tQuarkus          Trait
	tContainer        Trait
	tPullSecret		  Trait
}

// NewCatalog creates a new trait Catalog
func NewCatalog(ctx context.Context, c client.Client) *Catalog {
	catalog := Catalog{
		L:                 log.Log.WithName("trait"),
		tAffinity:         newAffinityTrait(),
		tCamel:            newCamelTrait(),
		tDebug:            newDebugTrait(),
		tRestDsl:          newRestDslTrait(),
		tKnative:          newKnativeTrait(),
		tDependencies:     newDependenciesTrait(),
		tDeployer:         newDeployerTrait(),
		tDeployment:       newDeploymentTrait(),
		tGarbageCollector: newGarbageCollectorTrait(),
		tKnativeService:   newKnativeServiceTrait(),
		tService:          newServiceTrait(),
		tRoute:            newRouteTrait(),
		tIngress:          newIngressTrait(),
		tJolokia:          newJolokiaTrait(),
		tPrometheus:       newPrometheusTrait(),
		tOwner:            newOwnerTrait(),
		tBuilder:          newBuilderTrait(),
		tIstio:            newIstioTrait(),
		tEnvironment:      newEnvironmentTrait(),
		tClasspath:        newClasspathTrait(),
		tProbes:           newProbesTrait(),
		tQuarkus:          newQuarkusTrait(),
		tContainer:        newContainerTrait(),
		tPullSecret:       newPullSecretTrait(),
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
		c.tAffinity,
		c.tCamel,
		c.tDebug,
		c.tRestDsl,
		c.tKnative,
		c.tDependencies,
		c.tDeployer,
		c.tDeployment,
		c.tGarbageCollector,
		c.tKnativeService,
		c.tService,
		c.tRoute,
		c.tIngress,
		c.tJolokia,
		c.tPrometheus,
		c.tOwner,
		c.tBuilder,
		c.tIstio,
		c.tEnvironment,
		c.tClasspath,
		c.tProbes,
		c.tQuarkus,
		c.tContainer,
		c.tPullSecret,
	}
}

// Traits may depend on the result of previously executed ones,
// so care must be taken while changing the lists order.
func (c *Catalog) traitsFor(environment *Environment) []Trait {
	profile := environment.DetermineProfile()
	return c.TraitsForProfile(profile)
}

// TraitsForProfile returns all traits associated with a given profile.
//
// Traits may depend on the result of previously executed ones,
// so care must be taken while changing the lists order.
func (c *Catalog) TraitsForProfile(profile v1alpha1.TraitProfile) []Trait {
	switch profile {
	case v1alpha1.TraitProfileOpenShift:
		return []Trait{
			c.tCamel,
			c.tGarbageCollector,
			c.tDebug,
			c.tRestDsl,
			c.tDependencies,
			c.tBuilder,
			c.tQuarkus,
			c.tEnvironment,
			c.tDeployer,
			c.tDeployment,
			c.tAffinity,
			c.tService,
			c.tContainer,
			c.tPullSecret,
			c.tJolokia,
			c.tPrometheus,
			c.tClasspath,
			c.tProbes,
			c.tRoute,
			c.tIstio,
			c.tOwner,
		}
	case v1alpha1.TraitProfileKubernetes:
		return []Trait{
			c.tCamel,
			c.tGarbageCollector,
			c.tDebug,
			c.tRestDsl,
			c.tDependencies,
			c.tBuilder,
			c.tQuarkus,
			c.tEnvironment,
			c.tDeployer,
			c.tDeployment,
			c.tAffinity,
			c.tService,
			c.tContainer,
			c.tPullSecret,
			c.tJolokia,
			c.tPrometheus,
			c.tClasspath,
			c.tProbes,
			c.tIngress,
			c.tIstio,
			c.tOwner,
		}
	case v1alpha1.TraitProfileKnative:
		return []Trait{
			c.tCamel,
			c.tGarbageCollector,
			c.tDebug,
			c.tRestDsl,
			c.tKnative,
			c.tDependencies,
			c.tBuilder,
			c.tQuarkus,
			c.tEnvironment,
			c.tDeployer,
			c.tDeployment,
			c.tAffinity,
			c.tKnativeService,
			c.tContainer,
			c.tPullSecret,
			c.tJolokia,
			c.tPrometheus,
			c.tClasspath,
			c.tProbes,
			c.tIstio,
			c.tOwner,
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
	if env.Platform != nil && env.Platform.Spec.Traits != nil {
		if err := c.configureTraits(env.Platform.Spec.Traits); err != nil {
			return err
		}
	}
	if env.IntegrationKit != nil && env.IntegrationKit.Spec.Traits != nil {
		if err := c.configureTraits(env.IntegrationKit.Spec.Traits); err != nil {
			return err
		}
	}
	if env.Integration != nil && env.Integration.Spec.Traits != nil {
		if err := c.configureTraits(env.Integration.Spec.Traits); err != nil {
			return err
		}
	}

	return nil
}

func (c *Catalog) configureTraits(traits map[string]v1alpha1.TraitSpec) error {
	for id, traitSpec := range traits {
		catTrait := c.GetTrait(id)
		if catTrait != nil {
			trait := traitSpec
			if err := decodeTraitSpec(&trait, catTrait); err != nil {
				return err
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
