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
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util"
)

// Configure reads trait configurations from environment and applies them to catalog.
func (c *Catalog) Configure(env *Environment) error {
	if env.Platform != nil {
		if err := c.configureTraits(env.Platform.Status.Traits); err != nil {
			return err
		}
		// Deprecated: to be removed in future version
		if err := c.configureTraitsFromAnnotations(env.Platform.Annotations); err != nil {
			return err
		}
	}
	if env.IntegrationProfile != nil {
		if err := c.configureTraits(env.IntegrationProfile.Spec.Traits); err != nil {
			return err
		}
	}
	if env.IntegrationKit != nil {
		if err := c.configureTraits(env.IntegrationKit.Spec.Traits); err != nil {
			return err
		}
		// Deprecated: to be removed in future version
		if err := c.configureTraitsFromAnnotations(env.IntegrationKit.Annotations); err != nil {
			return err
		}
	}
	if env.Integration != nil {
		if err := c.configureTraits(env.Integration.Spec.Traits); err != nil {
			return err
		}
		// Deprecated: to be removed in future version
		if err := c.configureTraitsFromAnnotations(env.Integration.Annotations); err != nil {
			return err
		}
	}

	return nil
}

func (c *Catalog) configureTraits(traits interface{}) error {
	traitMap, err := ToTraitMap(traits)
	if err != nil {
		return err
	}

	for id, trait := range traitMap {
		if id == "addons" {
			// Handle addons later so that the configurations on the new API
			// take precedence over the legacy addon configurations
			continue
		}
		if err := c.configureTrait(id, trait); err != nil {
			return err
		}
	}
	// Addons
	for id, trait := range traitMap["addons"] {
		if addons, ok := trait.(map[string]interface{}); ok {
			if err := c.configureTrait(id, addons); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Catalog) configureTrait(id string, trait map[string]interface{}) error {
	if catTrait := c.GetTrait(id); catTrait != nil {
		if err := decodeTrait(trait, catTrait); err != nil {
			return err
		}
	}

	return nil
}

func decodeTrait(in map[string]interface{}, target Trait) error {
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

// Deprecated: to be removed in future versions.
func (c *Catalog) configureTraitsFromAnnotations(annotations map[string]string) error {
	options := make(map[string]map[string]interface{}, len(annotations))
	for k, v := range annotations {
		if !strings.HasPrefix(k, v1.TraitAnnotationPrefix) {
			continue
		}

		configKey := strings.TrimPrefix(k, v1.TraitAnnotationPrefix)
		if !strings.Contains(configKey, ".") {
			return fmt.Errorf("wrong format for trait annotation %q: missing trait ID", k)
		}

		parts := strings.SplitN(configKey, ".", 2)
		id := parts[0]
		prop := parts[1]
		if _, ok := options[id]; !ok {
			options[id] = make(map[string]interface{})
		}

		propParts := util.ConfigTreePropertySplit(prop)
		var current = options[id]
		if len(propParts) > 1 {
			c, err := util.NavigateConfigTree(current, propParts[0:len(propParts)-1])
			if err != nil {
				return err
			}
			if cc, ok := c.(map[string]interface{}); ok {
				current = cc
			} else {
				return errors.New(`invalid array specification: to set an array value use the ["v1", "v2"] format`)
			}
		}
		current[prop] = v
	}
	return c.configureFromOptions(options)
}

// Deprecated: to be removed in future versions.
func (c *Catalog) configureFromOptions(traits map[string]map[string]interface{}) error {
	for id, config := range traits {
		t := c.GetTrait(id)
		if t != nil {
			err := configureTrait(id, config, t)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Deprecated: to be removed in future versions.
func configureTrait(id string, config map[string]interface{}, trait interface{}) error {
	md := mapstructure.Metadata{}

	var valueConverter mapstructure.DecodeHookFuncKind = func(sourceKind reflect.Kind, targetKind reflect.Kind, data interface{}) (interface{}, error) {
		// Allow JSON encoded arrays to set slices
		if sourceKind == reflect.String && targetKind == reflect.Slice {
			if v, ok := data.(string); ok && strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
				var value interface{}
				if err := json.Unmarshal([]byte(v), &value); err != nil {
					return nil, fmt.Errorf("could not decode JSON array for configuring trait property: %w", err)
				}
				return value, nil
			}
		}
		return data, nil
	}

	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Metadata:         &md,
			DecodeHook:       valueConverter,
			WeaklyTypedInput: true,
			TagName:          "property",
			Result:           &trait,
			ErrorUnused:      true,
		},
	)
	if err != nil {
		return fmt.Errorf("error while decoding trait configuration %q: %w", id, err)
	}

	return decoder.Decode(config)
}
