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

package source

import (
	"fmt"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	yaml2 "gopkg.in/yaml.v2"
)

// YAMLInspector --
type YAMLInspector struct {
	baseInspector
}

// Extract --
func (i YAMLInspector) Extract(source v1.SourceSpec, meta *Metadata) error {
	definitions := make([]map[string]interface{}, 0)

	if err := yaml2.Unmarshal([]byte(source.Content), &definitions); err != nil {
		return err
	}

	for _, definition := range definitions {
		for k, v := range definition {
			if err := i.parseStep(k, v, meta); err != nil {
				return err
			}
		}
	}

	i.discoverCapabilities(source, meta)
	i.discoverDependencies(source, meta)

	meta.ExposesHTTPServices = meta.ExposesHTTPServices || i.containsHTTPURIs(meta.FromURIs)
	meta.PassiveEndpoints = i.hasOnlyPassiveEndpoints(meta.FromURIs)

	return nil
}

func (i YAMLInspector) parseStep(key string, content interface{}, meta *Metadata) error {
	switch key {
	case "rest":
		meta.ExposesHTTPServices = true
		meta.RequiredCapabilities.Add(v1.CapabilityRest)
	case "circuitBreaker":
		meta.RequiredCapabilities.Add(v1.CapabilityCircuitBreaker)
	case "unmarshal":
		fallthrough
	case "marshal":
		if cm, ok := content.(map[interface{}]interface{}); ok {
			if js, jsOk := cm["json"]; jsOk {
				dataFormatID := defaultJsonDataformat
				if jsContent, jsContentOk := js.(map[interface{}]interface{}); jsContentOk {
					if lib, libOk := jsContent["library"]; libOk {
						dataFormatID = strings.ToLower(fmt.Sprintf("json-%s", lib))
					}
				}
				if dfDep := i.catalog.GetArtifactByDataFormat(dataFormatID); dfDep != nil {
					i.addDependency(dfDep.GetDependencyID(), meta)
				}
			}
		}
	}

	var maybeURI string

	switch t := content.(type) {
	case string:
		maybeURI = t
	case map[interface{}]interface{}:
		if u, ok := t["rest"]; ok {
			return i.parseStep("rest", u, meta)
		} else if u, ok := t["from"]; ok {
			return i.parseStep("from", u, meta)
		} else if u, ok := t["steps"]; ok {
			if steps, stepsFormatOk := u.([]interface{}); stepsFormatOk {
				for _, raw := range steps {
					if step, stepFormatOk := raw.(map[interface{}]interface{}); stepFormatOk {

						if len(step) != 1 {
							return fmt.Errorf("unable to parse step: %v", step)
						}

						for k, v := range step {
							switch kt := k.(type) {
							case fmt.Stringer:
								if err := i.parseStep(kt.String(), v, meta); err != nil {
									return err
								}
							case string:
								if err := i.parseStep(kt, v, meta); err != nil {
									return err
								}
							default:
								return fmt.Errorf("unknown key type: %v, step: %v", k, step)
							}
						}
					}
				}
			}
		}

		if u, ok := t["uri"]; ok {
			if v, isString := u.(string); isString {
				maybeURI = v
			}
		}

		if _, ok := t["language"]; ok {
			if s, ok := t["language"].(string); ok {
				if dependency, ok := i.catalog.GetLanguageDependency(s); ok {
					i.addDependency(dependency, meta)
				}
			} else if m, ok := t["language"].(map[interface{}]interface{}); ok {
				if err := i.parseStep("language", m, meta); err != nil {
					return err
				}
			}
		}

		for k := range t {
			if s, ok := k.(string); ok {
				if dependency, ok := i.catalog.GetLanguageDependency(s); ok {
					i.addDependency(dependency, meta)
				}
			}
		}
	}

	if maybeURI != "" {
		switch key {
		case "from":
			meta.FromURIs = append(meta.FromURIs, maybeURI)
		case "to":
			meta.ToURIs = append(meta.ToURIs, maybeURI)
		}
	}
	return nil
}
