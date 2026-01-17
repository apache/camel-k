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
	"errors"
	"fmt"
	"regexp"

	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/go-viper/mapstructure/v2"
)

type optionMap map[string]map[string]any

var traitConfigRegexp = regexp.MustCompile(`^([a-z0-9-]+)((?:\.[a-z0-9-]+)(?:\[[0-9]+\]|\..+)*)=(.*)$`)

func ValidateTrait(catalog *Catalog, trait string) error {
	tr := catalog.GetTrait(trait)
	if tr == nil {
		return fmt.Errorf("trait %s does not exist in catalog", trait)
	}

	return nil
}

func ValidateTraits(catalog *Catalog, traits []string) error {
	for _, t := range traits {
		if err := ValidateTrait(catalog, t); err != nil {
			return err
		}
	}

	return nil
}

func ConfigureTraits(options []string, traits any, catalog Finder) error {
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

	return nil
}

func optionsToMap(options []string) (optionMap, error) {
	optionMap := make(optionMap)

	for _, option := range options {
		parts := traitConfigRegexp.FindStringSubmatch(option)
		if len(parts) < 4 {
			return nil, errors.New("unrecognized config format (expected \"<trait>.<prop>=<value>\"): " + option)
		}
		id := parts[1]
		fullProp := parts[2][1:]
		value := parts[3]
		if _, ok := optionMap[id]; !ok {
			optionMap[id] = make(map[string]any)
		}

		propParts := util.ConfigTreePropertySplit(fullProp)
		var current = optionMap[id]
		if len(propParts) > 1 {
			c, err := util.NavigateConfigTree(current, propParts[0:len(propParts)-1])
			if err != nil {
				return nil, err
			}
			if cc, ok := c.(map[string]any); ok {
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
