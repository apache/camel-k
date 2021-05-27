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
	"fmt"
	"reflect"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

func (c *Catalog) configure(env *Environment) error {
	if env.Platform != nil {
		if env.Platform.Status.Traits != nil {
			if err := c.configureTraits(env.Platform.Status.Traits); err != nil {
				return err
			}
		}
		if err := c.configureTraitsFromAnnotations(env.Platform.Annotations); err != nil {
			return err
		}
	}
	if env.IntegrationKit != nil {
		if env.IntegrationKit.Spec.Traits != nil {
			if err := c.configureTraits(env.IntegrationKit.Spec.Traits); err != nil {
				return err
			}
		}
		if err := c.configureTraitsFromAnnotations(env.IntegrationKit.Annotations); err != nil {
			return err
		}
	}
	if env.Integration != nil {
		if env.Integration.Spec.Traits != nil {
			if err := c.configureTraits(env.Integration.Spec.Traits); err != nil {
				return err
			}
		}
		if err := c.configureTraitsFromAnnotations(env.Integration.Annotations); err != nil {
			return err
		}
	}

	return nil
}

func (c *Catalog) configureTraits(traits map[string]v1.TraitSpec) error {
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

func decodeTraitSpec(in *v1.TraitSpec, target interface{}) error {
	data, err := json.Marshal(&in.Configuration)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &target)
}

func (c *Catalog) configureTraitsFromAnnotations(annotations map[string]string) error {
	options := make(map[string]map[string]string, len(annotations))
	for k, v := range annotations {
		if strings.HasPrefix(k, v1.TraitAnnotationPrefix) {
			configKey := strings.TrimPrefix(k, v1.TraitAnnotationPrefix)
			if strings.Contains(configKey, ".") {
				parts := strings.SplitN(configKey, ".", 2)
				id := parts[0]
				prop := parts[1]
				if _, ok := options[id]; !ok {
					options[id] = make(map[string]string)
				}
				options[id][prop] = v
			} else {
				return fmt.Errorf("wrong format for trait annotation %q: missing trait ID", k)
			}
		}
	}
	return c.configureFromOptions(options)
}

func (c *Catalog) configureFromOptions(traits map[string]map[string]string) error {
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

func configureTrait(id string, config map[string]string, trait interface{}) error {
	md := mapstructure.Metadata{}

	var valueConverter mapstructure.DecodeHookFuncKind = func(sourceKind reflect.Kind, targetKind reflect.Kind, data interface{}) (interface{}, error) {
		// Allow JSON encoded arrays to set slices
		if sourceKind == reflect.String && targetKind == reflect.Slice {
			if v, ok := data.(string); ok && strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
				var value interface{}
				if err := json.Unmarshal([]byte(v), &value); err != nil {
					return nil, errors.Wrap(err, "could not decode JSON array for configuring trait property")
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
		return errors.Wrapf(err, "error while decoding trait configuration from annotations on trait %q", id)
	}

	return decoder.Decode(config)
}
