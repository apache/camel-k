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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/uri"
)

// YAMLInspector --.
type YAMLInspector struct {
	baseInspector
}

// Extract --.
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
	i.discoverKamelets(source, meta)

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
				dataFormatID := defaultJSONDataFormat
				if jsContent, jsContentOk := js.(map[interface{}]interface{}); jsContentOk {
					if lib, libOk := jsContent["library"]; libOk {
						dataFormatID = strings.ToLower(fmt.Sprintf("%s", lib))
					}
				}
				if dfDep := i.catalog.GetArtifactByDataFormat(dataFormatID); dfDep != nil {
					i.addDependency(dfDep.GetDependencyID(), meta)
				}
			}
		}
	case "kamelet":
		switch t := content.(type) {
		case string:
			AddKamelet(meta, "kamelet:"+t)
		case map[interface{}]interface{}:
			AddKamelet(meta, "kamelet:"+t["name"].(string))
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
					i.addDependency(dependency, meta)
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
						i.addDependency(dependency, meta)
					}
				} else if m, ok := v.(map[interface{}]interface{}); ok {
					if err := i.parseStep("language", m, meta); err != nil {
						return err
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
		case "to", "to-d", "toD", "wireTap":
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
