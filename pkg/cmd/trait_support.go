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

func configureTraits(options []string, traits interface{}, catalog trait.Finder) error {
	config, err := optionsToMap(options)
	if err != nil {
		return err
	}

	md := mapstructure.Metadata{}
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Metadata:         &md,
			WeaklyTypedInput: true,
			TagName:          "property",
			Result:           &traits,
		},
	)
	if err != nil {
		return err
	}

	if err = decoder.Decode(config); err != nil {
		return err
	}

	if len(md.Unused) == 0 {
		// No addons found
		return nil
	}

	addons := make(map[string]map[string]interface{})
	for _, id := range md.Unused {
		addons[id] = config[id]
	}

	return configureAddons(addons, traits, catalog)
}

func optionsToMap(options []string) (map[string]map[string]interface{}, error) {
	optionMap := make(map[string]map[string]interface{})

	for _, option := range options {
		parts := traitConfigRegexp.FindStringSubmatch(option)
		if len(parts) < 4 {
			return nil, errors.New("unrecognized config format (expected \"<trait>.<prop>=<value>\"): " + option)
		}
		id := parts[1]
		fullProp := parts[2][1:]
		value := parts[3]
		if _, ok := optionMap[id]; !ok {
			optionMap[id] = make(map[string]interface{})
		}

		propParts := util.ConfigTreePropertySplit(fullProp)
		var current = optionMap[id]
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

	return optionMap, nil
}

func configureAddons(config map[string]map[string]interface{}, traits interface{}, catalog trait.Finder) error {
	// Addon traits require raw message mapping
	addons := make(map[string]v1.AddonTrait)
	for id, props := range config {
		t := catalog.GetTrait(id)
		if t != nil {
			// let's take a clone to prevent default values set at runtime from being serialized
			zero := reflect.New(reflect.TypeOf(t)).Interface()
			if err := configureAddon(props, zero); err != nil {
				return err
			}
			data, err := json.Marshal(zero)
			if err != nil {
				return err
			}
			addon := v1.AddonTrait{}
			if err = json.Unmarshal(data, &addon); err != nil {
				return err
			}
			addons[id] = addon
		}
	}
	if len(addons) > 0 {
		if ts, ok := traits.(*v1.Traits); ok {
			ts.Addons = addons
		}
		if ikts, ok := traits.(*v1.IntegrationKitTraits); ok {
			ikts.Addons = addons
		}
	}

	return nil
}

func configureAddon(props map[string]interface{}, trait interface{}) error {
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

	return decoder.Decode(props)
}
