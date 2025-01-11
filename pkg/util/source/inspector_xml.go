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
	"encoding/xml"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

const (
	language = "language"
	URI      = "uri"
)

// XMLInspector inspects XML DSL spec.
type XMLInspector struct {
	baseInspector
}

// Extract extracts all metadata from source spec.
func (i XMLInspector) Extract(source v1.SourceSpec, meta *Metadata) error {
	content := strings.NewReader(source.Content)
	decoder := xml.NewDecoder(content)

	//nolint: nestif
	for {
		// Read tokens from the XML document in a stream.
		t, _ := decoder.Token()
		if t == nil {
			break
		}

		if se, ok := t.(xml.StartElement); ok {
			switch se.Name.Local {
			//nolint: goconst
			case "rest", "restConfiguration":
				meta.ExposesHTTPServices = true
				meta.RequiredCapabilities.Add(v1.CapabilityRest)
			case "openApi":
				if dfDep := i.catalog.GetArtifactByScheme("rest-openapi"); dfDep != nil {
					meta.AddDependency(dfDep.GetDependencyID())
				}
			case "circuitBreaker":
				meta.RequiredCapabilities.Add(v1.CapabilityCircuitBreaker)
			case "json":
				dataFormatID := defaultJSONDataFormat
				for _, a := range se.Attr {
					if a.Name.Local == "library" {
						dataFormatID = strings.ToLower(a.Value)
					}
				}
				if dfDep := i.catalog.GetArtifactByDataFormat(dataFormatID); dfDep != nil {
					meta.AddDependency(dfDep.GetDependencyID())
				}
			case language:
				for _, a := range se.Attr {
					if a.Name.Local == language {
						if dependency, ok := i.catalog.GetLanguageDependency(a.Value); ok {
							meta.AddDependency(dependency)
						}
					}
				}
			case "deadLetterChannel":
				for _, a := range se.Attr {
					if a.Name.Local == "deadLetterUri" {
						_, scheme := i.catalog.DecodeComponent(a.Value)
						if dfDep := i.catalog.GetArtifactByScheme(scheme.ID); dfDep != nil {
							meta.AddDependency(dfDep.GetDependencyID())
						}
						if scheme.ID == kamelet {
							AddKamelet(meta, a.Value)
						}
					}
				}
			case "from", "fromF":
				for _, a := range se.Attr {
					if a.Name.Local == URI {
						meta.FromURIs = append(meta.FromURIs, a.Value)
					}
				}
			case "to", "toD", "toF", "wireTap":
				for _, a := range se.Attr {
					if a.Name.Local == URI {
						meta.ToURIs = append(meta.ToURIs, a.Value)
					}
				}
			case kamelet:
				for _, a := range se.Attr {
					if a.Name.Local == "name" {
						AddKamelet(meta, kamelet+":"+a.Value)
					}
				}
			}

			if dependency, ok := i.catalog.GetLanguageDependency(se.Name.Local); ok {
				meta.AddDependency(dependency)
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

// ReplaceFromURI parses the source content and replace the `from` URI configuration with the a new URI. Returns true if it applies a replacement.
func (i XMLInspector) ReplaceFromURI(source *v1.SourceSpec, newFromURI string) (bool, error) {
	metadata := NewMetadata()
	if err := i.Extract(*source, &metadata); err != nil {
		return false, err
	}
	newContent := source.Content
	if metadata.FromURIs == nil {
		return false, nil
	}
	for _, from := range metadata.FromURIs {
		newContent = strings.ReplaceAll(newContent, from, newFromURI)
	}
	replaced := newContent != source.Content

	if replaced {
		source.Content = newContent
	}

	return replaced, nil
}
