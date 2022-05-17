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

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/mitchellh/mapstructure"
)

func validateTraits(catalog *trait.Catalog, traits []string) error {
	tp := catalog.ComputeTraitsProperties()
	for _, t := range traits {
		kv := strings.SplitN(t, "=", 2)
		prefix := kv[0]
		if strings.Contains(prefix, "[") {
			prefix = prefix[0:strings.Index(prefix, "[")]
		}
		if !util.StringSliceExists(tp, prefix) {
			return fmt.Errorf("%s is not a valid trait property", t)
		}
	}

	return nil
}

func configureTraits(options []string, catalog trait.Finder) (map[string]v1.TraitSpec, error) {
	traits := make(map[string]map[string]interface{})

	for _, option := range options {
		parts := traitConfigRegexp.FindStringSubmatch(option)
		if len(parts) < 4 {
			return nil, errors.New("unrecognized config format (expected \"<trait>.<prop>=<value>\"): " + option)
		}
		id := parts[1]
		fullProp := parts[2][1:]
		value := parts[3]
		if _, ok := traits[id]; !ok {
			traits[id] = make(map[string]interface{})
		}

		propParts := util.ConfigTreePropertySplit(fullProp)
		var current = traits[id]
		if len(propParts) > 1 {
			c, err := util.NavigateConfigTree(current, propParts[0:len(propParts)-1])
			if err != nil {
				return nil, err
			}
			if cc, ok := c.(map[string]interface{}); ok {
				current = cc
			} else {
				return nil, errors.New("trait configuration cannot end with a slice")
			}
		}

		prop := propParts[len(propParts)-1]
		switch v := current[prop].(type) {
		case []string:
			current[prop] = append(v, value)
		case string:
			// Aggregate multiple occurrences of the same option into a string array, to emulate POSIX conventions.
			// This enables executing:
			// $ kamel run -t <trait>.<property>=<value_1> ... -t <trait>.<property>=<value_N>
			// Or:
			// $ kamel run --trait <trait>.<property>=<value_1>,...,<trait>.<property>=<value_N>
			current[prop] = []string{v, value}
		case nil:
			current[prop] = value
		}
	}

	specs := make(map[string]v1.TraitSpec)
	for id, config := range traits {
		t := catalog.GetTrait(id)
		if t != nil {
			// let's take a clone to prevent default values set at runtime from being serialized
			zero := reflect.New(reflect.TypeOf(t)).Interface()
			err := configureTrait(config, zero)
			if err != nil {
				return nil, err
			}
			data, err := json.Marshal(zero)
			if err != nil {
				return nil, err
			}
			var spec v1.TraitSpec
			err = json.Unmarshal(data, &spec.Configuration)
			if err != nil {
				return nil, err
			}
			specs[id] = spec
		}
	}

	return specs, nil
}

func configureTrait(config map[string]interface{}, trait interface{}) error {
	md := mapstructure.Metadata{}

	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Metadata:         &md,
			WeaklyTypedInput: true,
			TagName:          "property",
			Result:           &trait,
		},
	)
	if err != nil {
		return err
	}

	return decoder.Decode(config)
}
