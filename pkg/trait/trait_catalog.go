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
	"reflect"
	"sort"
	"strings"

	"github.com/fatih/structs"
	"github.com/pkg/errors"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/log"
)

// Catalog collects all information about traits in one place.
type Catalog struct {
	L      log.Logger
	traits []Trait
}

// NewCatalog creates a new trait Catalog.
func NewCatalog(c client.Client) *Catalog {
	traitList := make([]Trait, 0, len(FactoryList))
	for _, factory := range FactoryList {
		traitList = append(traitList, factory())
	}
	sort.Slice(traitList, func(i, j int) bool {
		if traitList[i].Order() != traitList[j].Order() {
			return traitList[i].Order() < traitList[j].Order()
		}
		return string(traitList[i].ID()) < string(traitList[j].ID())
	})

	catalog := Catalog{
		L:      log.Log.WithName("trait"),
		traits: traitList,
	}

	for _, t := range catalog.AllTraits() {
		if c != nil {
			t.InjectClient(c)
		}
	}
	return &catalog
}

func (c *Catalog) AllTraits() []Trait {
	return append([]Trait(nil), c.traits...)
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
func (c *Catalog) TraitsForProfile(profile v1.TraitProfile) []Trait {
	var res []Trait
	for _, t := range c.AllTraits() {
		if t.IsAllowedInProfile(profile) {
			res = append(res, t)
		}
	}
	return res
}

func (c *Catalog) apply(environment *Environment) error {
	if err := c.Configure(environment); err != nil {
		return err
	}
	traits := c.traitsFor(environment)
	environment.ConfiguredTraits = traits

	applicable := false
	for _, trait := range traits {
		if !environment.PlatformInPhase(v1.IntegrationPlatformPhaseReady) && trait.RequiresIntegrationPlatform() {
			c.L.Debugf("Skipping trait because of missing integration platform: %s", trait.ID())

			continue
		}
		applicable = true
		enabled, err := trait.Configure(environment)
		if err != nil {
			return err
		}

		if enabled {
			err = trait.Apply(environment)
			if err != nil {
				return err
			}

			environment.ExecutedTraits = append(environment.ExecutedTraits, trait)

			// execute post step processors
			for _, processor := range environment.PostStepProcessors {
				err := processor(environment)
				if err != nil {
					return errors.Wrap(err, "error executing post step action")
				}
			}
		}
	}

	traitIds := make([]string, 0)
	for _, trait := range environment.ExecutedTraits {
		traitIds = append(traitIds, string(trait.ID()))
	}
	c.L.Debugf("Applied traits: %s", strings.Join(traitIds, ","))

	if !applicable && environment.PlatformInPhase(v1.IntegrationPlatformPhaseReady) {
		return errors.New("no trait can be executed because of no ready platform found")
	}

	for _, processor := range environment.PostProcessors {
		err := processor(environment)
		if err != nil {
			return errors.Wrap(err, "error executing post processor")
		}
	}

	return nil
}

// GetTrait returns the trait with the given ID.
func (c *Catalog) GetTrait(id string) Trait {
	for _, t := range c.AllTraits() {
		if t.ID() == ID(id) {
			return t
		}
	}
	return nil
}

// ComputeTraitsProperties returns all key/value configuration properties that can be used to configure traits.
func (c *Catalog) ComputeTraitsProperties() []string {
	results := make([]string, 0)
	for _, trait := range c.AllTraits() {
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

type Finder interface {
	GetTrait(id string) Trait
}

var _ Finder = &Catalog{}
