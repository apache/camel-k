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
	"sort"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	src "github.com/apache/camel-k/pkg/util/source"
)

// ExtractAll returns metadata information from all listed source codes
func ExtractAll(sources []v1alpha1.SourceSpec) IntegrationMetadata {
	// neutral metadata
	meta := IntegrationMetadata{
		Dependencies:        []string{},
		FromURIs:            []string{},
		ToURIs:              []string{},
		PassiveEndpoints:    true,
		RequiresHTTPService: false,
	}
	for _, source := range sources {
		meta = merge(meta, Extract(source))
	}
	return meta
}

func merge(m1 IntegrationMetadata, m2 IntegrationMetadata) IntegrationMetadata {
	deps := make(map[string]bool)
	for _, d := range m1.Dependencies {
		deps[d] = true
	}
	for _, d := range m2.Dependencies {
		deps[d] = true
	}
	allDependencies := make([]string, 0)
	for k := range deps {
		allDependencies = append(allDependencies, k)
	}
	sort.Strings(allDependencies)
	return IntegrationMetadata{
		FromURIs:            append(m1.FromURIs, m2.FromURIs...),
		ToURIs:              append(m1.ToURIs, m2.ToURIs...),
		Dependencies:        allDependencies,
		RequiresHTTPService: m1.RequiresHTTPService || m2.RequiresHTTPService,
		PassiveEndpoints:    m1.PassiveEndpoints && m2.PassiveEndpoints,
	}
}

// Extract returns metadata information from the source code
func Extract(source v1alpha1.SourceSpec) IntegrationMetadata {
	language := source.InferLanguage()
	// TODO: handle error
	fromURIs, _ := src.InspectorForLanguage(language).FromURIs(source)
	// TODO:: handle error
	toURIs, _ := src.InspectorForLanguage(language).ToURIs(source)
	dependencies := discoverDependencies(source, fromURIs, toURIs)
	requiresHTTPService := requiresHTTPService(source, fromURIs)
	passiveEndpoints := hasOnlyPassiveEndpoints(source, fromURIs)
	return IntegrationMetadata{
		FromURIs:            fromURIs,
		ToURIs:              toURIs,
		Dependencies:        dependencies,
		RequiresHTTPService: requiresHTTPService,
		PassiveEndpoints:    passiveEndpoints,
	}
}

// Each --
func Each(sources []v1alpha1.SourceSpec, consumer func(int, IntegrationMetadata) bool) {
	for i, s := range sources {
		meta := Extract(s)

		if !consumer(i, meta) {
			break
		}
	}
}
