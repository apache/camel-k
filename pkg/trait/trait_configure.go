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
	"encoding/json"

	"github.com/apache/camel-k/v2/pkg/platform"
)

// Configure reads trait configurations from environment and applies them to catalog.
func (c *Catalog) Configure(env *Environment) error {
	if env.IntegrationProfile != nil {
		if err := c.configureTraits(env.IntegrationProfile.Spec.Traits); err != nil {
			return err
		}
	}
	if env.IntegrationKit != nil {
		if err := c.configureTraits(env.IntegrationKit.Spec.Traits); err != nil {
			return err
		}
	}

	if env.Integration != nil {
		ip, err := platform.GetForResource(env.Ctx, env.Client, env.Integration)
		if ip != nil && err == nil {
			if err := c.configureTraits(ip.Status.Traits); err != nil {
				return err
			}
		}

		if err := c.configureTraits(env.Integration.Spec.Traits); err != nil {
			return err
		}
	}

	return nil
}

func (c *Catalog) configureTraits(traits any) error {
	traitMap, err := ToTraitMap(traits)
	if err != nil {
		return err
	}

	for id, trait := range traitMap {
		if err := c.configureTrait(id, trait); err != nil {
			return err
		}
	}

	return nil
}

func (c *Catalog) configureTrait(id string, trait map[string]any) error {
	if catTrait := c.GetTrait(id); catTrait != nil {
		if err := decodeTrait(trait, catTrait); err != nil {
			return err
		}
	}

	return nil
}

func decodeTrait(in map[string]any, target Trait) error {
	// Migrate legacy configuration properties before applying to catalog
	if err := MigrateLegacyConfiguration(in); err != nil {
		return err
	}

	data, err := json.Marshal(&in)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}
