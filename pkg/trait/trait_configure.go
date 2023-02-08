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
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

var traitCm = regexp.MustCompile("{{configmap:([a-z0-9-]+)/([a-z0-9-]+)}}")

// Configure reads trait configurations from environment and applies them to catalog.
func (c *Catalog) Configure(env *Environment) error {
	if env.Platform != nil {
		if err := c.configureTraits(env.Ctx, env.Client, env.Platform.Namespace, env.Platform.Status.Traits); err != nil {
			return err
		}
		if err := c.configureTraitsFromAnnotations(env.Ctx, env.Client, env.Platform.Namespace, env.Platform.Annotations); err != nil {
			return err
		}
	}
	if env.IntegrationKit != nil {
		if err := c.configureTraits(env.Ctx, env.Client, env.IntegrationKit.Namespace, env.IntegrationKit.Spec.Traits); err != nil {
			return err
		}
		if err := c.configureTraitsFromAnnotations(env.Ctx, env.Client, env.IntegrationKit.Namespace, env.IntegrationKit.Annotations); err != nil {
			return err
		}
	}
	if env.Integration != nil {
		if err := c.configureTraits(env.Ctx, env.Client, env.Integration.Namespace, env.Integration.Spec.Traits); err != nil {
			return err
		}
		if err := c.configureTraitsFromAnnotations(env.Ctx, env.Client, env.Integration.Namespace, env.Integration.Annotations); err != nil {
			return err
		}
	}

	return nil
}

func (c *Catalog) configureTraits(ctx context.Context, cl client.Client, ns string, traits interface{}) error {
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
		if err := c.configureTrait(ctx, cl, ns, id, trait); err != nil {
			return err
		}
	}
	// Addons
	for id, trait := range traitMap["addons"] {
		if addons, ok := trait.(map[string]interface{}); ok {
			if err := c.configureTrait(ctx, cl, ns, id, addons); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Catalog) configureTrait(ctx context.Context, cl client.Client, ns string, id string, trait map[string]interface{}) error {
	if catTrait := c.GetTrait(id); catTrait != nil {
		if err := decodeTrait(ctx, cl, ns, trait, catTrait); err != nil {
			return err
		}
	}

	return nil
}

func decodeTrait(ctx context.Context, cl client.Client, ns string, in map[string]interface{}, target Trait) error {
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

func (c *Catalog) configureTraitsFromAnnotations(ctx context.Context, cl client.Client, ns string, annotations map[string]string) error {
	options := make(map[string]map[string]interface{}, len(annotations))
	for k, v := range annotations {
		if strings.HasPrefix(k, v1.TraitAnnotationPrefix) {
			configKey := strings.TrimPrefix(k, v1.TraitAnnotationPrefix)
			if strings.Contains(configKey, ".") {
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

			} else {
				return fmt.Errorf("wrong format for trait annotation %q: missing trait ID", k)
			}
		}
	}
	return c.configureFromOptions(ctx, cl, ns, options)
}

func (c *Catalog) configureFromOptions(ctx context.Context, cl client.Client, ns string, traits map[string]map[string]interface{}) error {
	for id, config := range traits {
		t := c.GetTrait(id)
		if t != nil {
			err := configureTrait(ctx, cl, ns, id, config, t)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func configureTrait(ctx context.Context, c client.Client, ns string, id string, config map[string]interface{}, trait interface{}) error {
	err := parse(ctx, c, ns, config)
	if err != nil {
		return err
	}

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
		return errors.Wrapf(err, "error while decoding trait configuration %q", id)
	}

	return decoder.Decode(config)
}

func parse(ctx context.Context, c client.Client, ns string, config map[string]interface{}) error {
	for prop, val := range config {
		switch x := val.(type) {
		// if the value is an array, we need to recursively parse it as well
		case []interface{}:
			traitStr := val.([]interface{})
			for i, k := range traitStr {
				valFromCm, err := getFromConfigmap(ctx, c, ns, k)
				if err != nil {
					return err
				}
				// Replace the value with the one loaded dynamically
				if valFromCm != "" && valFromCm != k {
					traitStr[i] = valFromCm
				}
			}
		case interface{}:
			traitStr := val.(interface{})
			valFromCm, err := getFromConfigmap(ctx, c, ns, traitStr)
			if err != nil {
				return err
			}
			// Replace the value with the one loaded dynamically
			if valFromCm != "" && valFromCm != traitStr {
				config[prop] = valFromCm
			}

		default:
			return fmt.Errorf("unable to parse type %T", x)
		}
	}

	return nil
}

func getFromConfigmap(ctx context.Context, c client.Client, ns string, value interface{}) (interface{}, error) {
	strVal := fmt.Sprintf("%v", value)
	if !traitCm.MatchString(strVal) {
		// Nothing to parse, it's not a dynamic value
		return "", nil
	}
	matches := traitCm.FindStringSubmatch(strVal)
	// The regexp returns also the whole string, reason why we expect 3 values
	if len(matches) != 3 {
		return "", fmt.Errorf("unable to extract a value for %s configmap, syntax must be {{configmap:my-cm/my-prop}}", strVal)
	}
	// TODO we may cache locally the contents of the configmap
	cm := kubernetes.LookupConfigmap(ctx, c, ns, matches[1])
	if cm == nil {
		return "", fmt.Errorf("%v configmap not found in %s namespace, make sure to provide it before the Integration can run", matches[1], ns)
	}
	if cm.Data[matches[2]] == "" {
		return "", fmt.Errorf("Empty value for configmap property %s", matches[2])
	}

	if intValue, err := strconv.Atoi(cm.Data[matches[2]]); err == nil {
		return intValue, nil
	}
	if boolValue, err := strconv.ParseBool(cm.Data[matches[2]]); err == nil {
		return boolValue, nil
	}
	return cm.Data[matches[2]], nil
}
