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
	"github.com/scylladb/go-set/strset"

	"github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/gzip"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/log"

	src "github.com/apache/camel-k/pkg/util/source"
)

// ExtractAll returns metadata information from all listed source codes
func ExtractAll(catalog *camel.RuntimeCatalog, sources []v1.SourceSpec) IntegrationMetadata {
	// neutral metadata
	meta := NewIntegrationMetadata()
	meta.PassiveEndpoints = true
	meta.RequiresHTTPService = false

	for _, source := range sources {
		meta = merge(meta, Extract(catalog, source))
	}
	return meta
}

func merge(m1 IntegrationMetadata, m2 IntegrationMetadata) IntegrationMetadata {
	d := strset.Union(m1.Dependencies, m2.Dependencies)

	f := make([]string, 0, len(m1.FromURIs)+len(m2.FromURIs))
	f = append(f, m1.FromURIs...)
	f = append(f, m2.FromURIs...)

	t := make([]string, 0, len(m1.ToURIs)+len(m2.ToURIs))
	t = append(t, m1.ToURIs...)
	t = append(t, m2.ToURIs...)

	return IntegrationMetadata{
		Metadata: src.Metadata{
			FromURIs:     f,
			ToURIs:       t,
			Dependencies: d,
		},
		RequiresHTTPService: m1.RequiresHTTPService || m2.RequiresHTTPService,
		PassiveEndpoints:    m1.PassiveEndpoints && m2.PassiveEndpoints,
	}
}

// Extract returns metadata information from the source code
func Extract(catalog *camel.RuntimeCatalog, source v1.SourceSpec) IntegrationMetadata {
	var err error
	source, err = uncompress(source)
	if err != nil {
		log.Errorf(err, "unable to uncompress source %s: %v", source.Name, err)
	}

	language := source.InferLanguage()

	m := NewIntegrationMetadata()

	// TODO: handle error
	_ = src.InspectorForLanguage(catalog, language).Extract(source, &m.Metadata)

	m.RequiresHTTPService = requiresHTTPService(catalog, source, m.FromURIs)
	m.PassiveEndpoints = hasOnlyPassiveEndpoints(catalog, source, m.FromURIs)

	return m
}

// Each --
func Each(catalog *camel.RuntimeCatalog, sources []v1.SourceSpec, consumer func(int, IntegrationMetadata) bool) {
	for i, s := range sources {
		meta := Extract(catalog, s)

		if !consumer(i, meta) {
			break
		}
	}
}

func uncompress(spec v1.SourceSpec) (v1.SourceSpec, error) {
	if spec.Compression {
		data := []byte(spec.Content)
		var uncompressed []byte
		var err error
		if uncompressed, err = gzip.UncompressBase64(data); err != nil {
			return spec, err
		}
		newSpec := spec
		newSpec.Compression = false
		newSpec.Content = string(uncompressed)
		return newSpec, nil
	}
	return spec, nil
}
