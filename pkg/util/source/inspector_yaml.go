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

	yaml2 "gopkg.in/yaml.v2"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/uri"
)

// YAMLInspector inspects YAML DSL spec.
type YAMLInspector struct {
	baseInspector
}

// Extract extracts all metadata from source spec.
func (i YAMLInspector) Extract(source v1.SourceSpec, meta *Metadata) error {
	definitions := make([]map[string]interface{}, 0)

	if err := yaml2.Unmarshal([]byte(source.Content), &definitions); err != nil {
		return err
	}

	for _, definition := range definitions {
		if err := i.parseDefinition(definition, meta); err != nil {
			return err
		}
		for k, v := range definition {
			if err := i.parseStep(k, v, meta); err != nil {
				return err
			}
		}
	}

	if err := i.discoverCapabilities(source, meta); err != nil {
		return err
	}
	if err := i.discoverDependencies(source, meta); err != nil {
		return err
	}
	i.discoverKamelets(meta)

	meta.ExposesHTTPServices = meta.ExposesHTTPServices || i.containsHTTPURIs(meta.FromURIs)
	meta.PassiveEndpoints = i.hasOnlyPassiveEndpoints(meta.FromURIs)

	return nil
}

//nolint:nestif
func (i YAMLInspector) parseDefinition(def map[string]interface{}, meta *Metadata) error {
	for k, v := range def {
		if k == "rest" {
			meta.ExposesHTTPServices = true
			meta.RequiredCapabilities.Add(v1.CapabilityRest)
			// support contract first openapi
			if oa, ok := v.(map[interface{}]interface{}); ok {
				if _, oaOk := oa["openApi"]; oaOk {
					if dfDep := i.catalog.GetArtifactByScheme("rest-openapi"); dfDep != nil {
						meta.AddDependency(dfDep.GetDependencyID())
					}
				}
			}
		}
	}

	return nil
}

//nolint:nestif
func (i YAMLInspector) parseStep(key string, content interface{}, meta *Metadata) error {
	switch key {
	case "bean":
		if bean := i.catalog.GetArtifactByScheme("bean"); bean != nil {
			meta.AddDependency(bean.GetDependencyID())
		}
	case "rest":
		meta.ExposesHTTPServices = true
		meta.RequiredCapabilities.Add(v1.CapabilityRest)
	case "circuitBreaker":
		meta.RequiredCapabilities.Add(v1.CapabilityCircuitBreaker)
	case "marshal", "unmarshal":
		if cm, ok := content.(map[interface{}]interface{}); ok {
			if js, jsOk := cm["json"]; jsOk {
				dataFormatID := defaultJSONDataFormat
				if jsContent, jsContentOk := js.(map[interface{}]interface{}); jsContentOk {
					if lib, libOk := jsContent["library"]; libOk {
						dataFormatID = strings.ToLower(fmt.Sprintf("%s", lib))
					}
				}
				if dfDep := i.catalog.GetArtifactByDataFormat(dataFormatID); dfDep != nil {
					meta.AddDependency(dfDep.GetDependencyID())
				}
			}
		}
	case kamelet:
		switch t := content.(type) {
		case string:
			AddKamelet(meta, kamelet+":"+t)
		case map[interface{}]interface{}:
			if name, ok := t["name"].(string); ok {
				AddKamelet(meta, kamelet+":"+name)
			}
		}
	}

	var maybeURI string

	switch t := content.(type) {
	case string:
		maybeURI = t
	case map[interface{}]interface{}:
		for k, v := range t {

			if s, ok := k.(string); ok {
				if dependency, ok := i.catalog.GetLanguageDependency(s); ok {
					meta.AddDependency(dependency)
				}
			}

			switch k {
			case "steps":
				if steps, stepsFormatOk := v.([]interface{}); stepsFormatOk {
					if err := i.parseStepsParam(steps, meta); err != nil {
						return err
					}
				}
			case "uri":
				if vv, isString := v.(string); isString {
					builtURI := vv
					// Inject parameters into URIs to allow other parts of the operator to inspect them
					if params, pok := t["parameters"]; pok {
						if paramMap, pmok := params.(map[interface{}]interface{}); pmok {
							params := make(map[string]string, len(paramMap))
							for k, v := range paramMap {
								ks := fmt.Sprintf("%v", k)
								vs := fmt.Sprintf("%v", v)
								params[ks] = vs
							}
							builtURI = uri.AppendParameters(builtURI, params)
						}
					}
					maybeURI = builtURI
				}
			case "language":
				if s, ok := v.(string); ok {
					if dependency, ok := i.catalog.GetLanguageDependency(s); ok {
						meta.AddDependency(dependency)
					}
				} else if m, ok := v.(map[interface{}]interface{}); ok {
					if err := i.parseStep("language", m, meta); err != nil {
						return err
					}
				}
			case "deadLetterUri":
				if s, ok := v.(string); ok {
					_, scheme := i.catalog.DecodeComponent(s)
					if dfDep := i.catalog.GetArtifactByScheme(scheme.ID); dfDep != nil {
						meta.AddDependency(dfDep.GetDependencyID())
					}
					if scheme.ID == kamelet {
						AddKamelet(meta, s)
					}
				}
			default:
				// Always follow children because from/to uris can be nested
				if ks, ok := k.(string); ok {
					if _, ok := v.(map[interface{}]interface{}); ok {
						if err := i.parseStep(ks, v, meta); err != nil {
							return err
						}
					} else if ls, ok := v.([]interface{}); ok {
						for _, el := range ls {
							if err := i.parseStep(ks, el, meta); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}

	if maybeURI != "" {
		switch key {
		case "from":
			meta.FromURIs = append(meta.FromURIs, maybeURI)
		case "to", "to-d", "toD", "wire-tap", "wireTap":
			meta.ToURIs = append(meta.ToURIs, maybeURI)
		}
	}
	return nil
}

// TODO nolint: gocyclo.
func (i YAMLInspector) parseStepsParam(steps []interface{}, meta *Metadata) error {
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
	return nil
}

// ReplaceFromURI parses the source content and replace the `from` URI configuration with the a new URI.
// Returns true if it applies a replacement.
func (i YAMLInspector) ReplaceFromURI(source *v1.SourceSpec, newFromURI string) (bool, error) {
	definitions := make([]map[string]interface{}, 0)

	if err := yaml2.Unmarshal([]byte(source.Content), &definitions); err != nil {
		return false, err
	}

	// We expect the from in .route.from or .from location
	for _, routeRaw := range definitions {
		var from map[interface{}]interface{}
		var fromOk bool
		route, routeOk := routeRaw["route"].(map[interface{}]interface{})
		if routeOk {
			from, fromOk = route["from"].(map[interface{}]interface{})
			if !fromOk {
				return false, nil
			}
		}
		if from == nil {
			from, fromOk = routeRaw["from"].(map[interface{}]interface{})
			if !fromOk {
				return false, nil
			}
		}
		delete(from, "parameters")
		oldURI, ok := from["uri"].(string)
		if ok && (strings.HasPrefix(oldURI, "timer") || strings.HasPrefix(oldURI, "cron") || strings.HasPrefix(oldURI, "quartz")) {
			from["uri"] = newFromURI
		}
	}

	newContentRaw, err := yaml2.Marshal(definitions)
	if err != nil {
		return false, err
	}

	newContent := string(newContentRaw)
	if newContent != source.Content {
		source.Content = newContent
		return true, nil
	}

	return false, nil
}
