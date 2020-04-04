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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	yaml2 "gopkg.in/yaml.v2"
)

// YAMLInspector --
type YAMLInspector struct {
	baseInspector
}

// Extract --
func (inspector YAMLInspector) Extract(source v1.SourceSpec, meta *Metadata) error {
	definitions := make([]map[string]interface{}, 0)

	if err := yaml2.Unmarshal([]byte(source.Content), &definitions); err != nil {
		return err
	}

	for i := range definitions {
		definition := definitions[i]

		for k, v := range definition {
			if err := inspector.parseStep(k, v, meta); err != nil {
				return err
			}
		}
	}

	inspector.discoverDependencies(source, meta)

	meta.ExposesHTTPServices = meta.ExposesHTTPServices || inspector.containsHTTPURIs(meta.FromURIs)
	meta.PassiveEndpoints = inspector.hasOnlyPassiveEndpoints(meta.FromURIs)

	return nil
}

func (inspector YAMLInspector) parseStep(key string, content interface{}, meta *Metadata) error {
	switch key {
	case "rest":
		meta.ExposesHTTPServices = true
		meta.RequiredCapabilities.Add(v1.CapabilityRest)
	case "circuitBreaker":
		inspector.addDependency("camel:hystrix", meta)
	}

	var maybeURI string

	switch t := content.(type) {
	case string:
		maybeURI = t
	case map[interface{}]interface{}:
		if u, ok := t["rest"]; ok {
			return inspector.parseStep("rest", u, meta)
		} else if u, ok := t["from"]; ok {
			return inspector.parseStep("from", u, meta)
		} else if u, ok := t["steps"]; ok {
			steps := u.([]interface{})

			for i := range steps {
				step := steps[i].(map[interface{}]interface{})

				if len(step) != 1 {
					return fmt.Errorf("unable to parse step: %v", step)
				}

				for k, v := range step {
					switch kt := k.(type) {
					case fmt.Stringer:
						if err := inspector.parseStep(kt.String(), v, meta); err != nil {
							return err
						}
					case string:
						if err := inspector.parseStep(kt, v, meta); err != nil {
							return err
						}
					default:
						return fmt.Errorf("unknown key type: %v, step: %v", k, step)
					}
				}
			}
		}

		if u, ok := t["uri"]; ok {
			maybeURI = u.(string)
		}

		if _, ok := t["language"]; ok {
			if s, ok := t["language"].(string); ok {
				if dependency, ok := inspector.catalog.GetLanguageDependency(s); ok {
					inspector.addDependency(dependency, meta)
				}
			} else if m, ok := t["language"].(map[interface{}]interface{}); ok {
				if err := inspector.parseStep("language", m, meta); err != nil {
					return err
				}
			}
		}

		for k := range t {
			if s, ok := k.(string); ok {
				if dependency, ok := inspector.catalog.GetLanguageDependency(s); ok {
					inspector.addDependency(dependency, meta)
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
