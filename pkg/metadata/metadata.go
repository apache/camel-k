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

package metadata

import (
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/sets"
	src "github.com/apache/camel-k/v2/pkg/util/source"
)

// ExtractAll returns metadata information from all listed source codes.
func ExtractAll(catalog *camel.RuntimeCatalog, sources []v1.SourceSpec) (IntegrationMetadata, error) {
	// neutral metadata
	meta := src.NewMetadata()
	meta.PassiveEndpoints = true
	meta.ExposesHTTPServices = false

	for _, source := range sources {
		m, err := extract(catalog, source)
		if err != nil {
			return IntegrationMetadata{}, err
		}
		meta = merge(meta, m.Metadata)
	}

	return IntegrationMetadata{
		Metadata: meta,
	}, nil
}

func merge(m1 src.Metadata, m2 src.Metadata) src.Metadata {
	f := make([]string, 0, len(m1.FromURIs)+len(m2.FromURIs))
	f = append(f, m1.FromURIs...)
	f = append(f, m2.FromURIs...)

	t := make([]string, 0, len(m1.ToURIs)+len(m2.ToURIs))
	t = append(t, m1.ToURIs...)
	t = append(t, m2.ToURIs...)

	k := make([]string, 0, len(m1.Kamelets)+len(m2.Kamelets))
	k = append(k, m1.Kamelets...)
	k = append(k, m2.Kamelets...)

	return src.Metadata{
		FromURIs:             f,
		ToURIs:               t,
		Dependencies:         sets.Union(m1.Dependencies, m2.Dependencies),
		RequiredCapabilities: sets.Union(m1.RequiredCapabilities, m2.RequiredCapabilities),
		ExposesHTTPServices:  m1.ExposesHTTPServices || m2.ExposesHTTPServices,
		PassiveEndpoints:     m1.PassiveEndpoints && m2.PassiveEndpoints,
		Kamelets:             k,
	}
}

// extract returns metadata information from the source code.
func extract(catalog *camel.RuntimeCatalog, source v1.SourceSpec) (IntegrationMetadata, error) {
	if source.ContentRef != "" {
		panic("source must be dereferenced before calling this method")
	}
	if source.Compression {
		panic("source must be uncompressed before calling this method")
	}

	language := source.InferLanguage()

	meta := src.NewMetadata()
	meta.PassiveEndpoints = true
	meta.ExposesHTTPServices = false

	if err := src.InspectorForLanguage(catalog, language).Extract(source, &meta); err != nil {
		return IntegrationMetadata{}, err
	}

	return IntegrationMetadata{
		Metadata: meta,
	}, nil
}
