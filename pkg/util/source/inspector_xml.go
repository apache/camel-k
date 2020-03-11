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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// XMLInspector --
type XMLInspector struct {
	baseInspector
}

// Extract --
func (i XMLInspector) Extract(source v1.SourceSpec, meta *Metadata) error {
	content := strings.NewReader(source.Content)
	decoder := xml.NewDecoder(content)

	for {
		// Read tokens from the XML document in a stream.
		t, _ := decoder.Token()
		if t == nil {
			break
		}

		if se, ok := t.(xml.StartElement); ok {
			switch se.Name.Local {
			case "rest", "restConfiguration":
				i.addDependency("camel:rest", meta)
				meta.ExposesHTTPServices = true
			case "circuitBreaker":
				i.addDependency("camel:hystrix", meta)
			case "language":
				for _, a := range se.Attr {
					if a.Name.Local == "language" {
						if dependency, ok := i.catalog.GetLanguageDependency(a.Value); ok {
							i.addDependency(dependency, meta)
						}
					}
				}
			case "from", "fromF":
				for _, a := range se.Attr {
					if a.Name.Local == "uri" {
						meta.FromURIs = append(meta.FromURIs, a.Value)
					}
				}
			case "to", "toD", "toF":
				for _, a := range se.Attr {
					if a.Name.Local == "uri" {
						meta.ToURIs = append(meta.ToURIs, a.Value)
					}
				}
			}

			if dependency, ok := i.catalog.GetLanguageDependency(se.Name.Local); ok {
				i.addDependency(dependency, meta)
			}
		}
	}

	i.discoverDependencies(source, meta)

	meta.ExposesHTTPServices = meta.ExposesHTTPServices || i.containsHTTPURIs(meta.FromURIs)
	meta.PassiveEndpoints = i.hasOnlyPassiveEndpoints(meta.FromURIs)

	return nil
}
