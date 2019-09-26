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

package camel

import (
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// NewRuntimeCatalog --
func NewRuntimeCatalog(spec v1alpha1.CamelCatalogSpec) *RuntimeCatalog {
	catalog := RuntimeCatalog{}
	catalog.CamelCatalogSpec = spec
	catalog.artifactByScheme = make(map[string]string)
	catalog.schemesByID = make(map[string]v1alpha1.CamelScheme)
	catalog.languageDependencies = make(map[string]string)

	for id, artifact := range catalog.Artifacts {
		for _, scheme := range artifact.Schemes {
			scheme := scheme
			catalog.artifactByScheme[scheme.ID] = id
			catalog.schemesByID[scheme.ID] = scheme
		}
		for _, language := range artifact.Languages {
			// Skip languages in common dependencies since they are always available to integrations
			if artifact.ArtifactID != "camel-base" {
				catalog.languageDependencies[language] = strings.Replace(artifact.ArtifactID, "camel-", "camel:", 1)
			}
		}
	}

	return &catalog
}

// RuntimeCatalog --
type RuntimeCatalog struct {
	v1alpha1.CamelCatalogSpec

	artifactByScheme     map[string]string
	schemesByID          map[string]v1alpha1.CamelScheme
	languageDependencies map[string]string
}

// HasArtifact --
func (c *RuntimeCatalog) HasArtifact(artifact string) bool {
	if !strings.HasPrefix(artifact, "camel-") {
		artifact = "camel-" + artifact
	}

	_, ok := c.Artifacts[artifact]

	return ok
}

// GetArtifactByScheme returns the artifact corresponding to the given component scheme
func (c *RuntimeCatalog) GetArtifactByScheme(scheme string) *v1alpha1.CamelArtifact {
	if id, ok := c.artifactByScheme[scheme]; ok {
		if artifact, present := c.Artifacts[id]; present {
			return &artifact
		}
	}
	return nil
}

// GetScheme returns the scheme definition for the given scheme id
func (c *RuntimeCatalog) GetScheme(id string) (v1alpha1.CamelScheme, bool) {
	scheme, ok := c.schemesByID[id]
	return scheme, ok
}

// GetLanguageDependency returns the maven dependency for the given language name
func (c *RuntimeCatalog) GetLanguageDependency(language string) (string, bool) {
	language, ok := c.languageDependencies[language]
	return language, ok
}

// VisitArtifacts --
func (c *RuntimeCatalog) VisitArtifacts(visitor func(string, v1alpha1.CamelArtifact) bool) {
	for id, artifact := range c.Artifacts {
		if !visitor(id, artifact) {
			break
		}
	}
}

// VisitSchemes --
func (c *RuntimeCatalog) VisitSchemes(visitor func(string, v1alpha1.CamelScheme) bool) {
	for id, scheme := range c.schemesByID {
		if !visitor(id, scheme) {
			break
		}
	}
}
