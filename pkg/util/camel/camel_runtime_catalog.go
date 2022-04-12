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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// NewRuntimeCatalog --.
func NewRuntimeCatalog(spec v1.CamelCatalogSpec) *RuntimeCatalog {
	catalog := RuntimeCatalog{}
	catalog.CamelCatalogSpec = spec
	catalog.artifactByScheme = make(map[string]string)
	catalog.artifactByDataFormat = make(map[string]string)
	catalog.schemesByID = make(map[string]v1.CamelScheme)
	catalog.languageDependencies = make(map[string]string)
	catalog.javaTypeDependencies = make(map[string]string)

	for id, artifact := range catalog.Artifacts {
		for _, scheme := range artifact.Schemes {
			scheme := scheme

			// In case of duplicate only, choose the "org.apache.camel.quarkus" artifact (if present).
			// Workaround for https://github.com/apache/camel-k-runtime/issues/592
			if _, duplicate := catalog.artifactByScheme[scheme.ID]; duplicate {
				if artifact.GroupID != "org.apache.camel.quarkus" {
					continue
				}
			}

			catalog.artifactByScheme[scheme.ID] = id
			catalog.schemesByID[scheme.ID] = scheme
		}
		for _, dataFormat := range artifact.DataFormats {
			dataFormat := dataFormat
			catalog.artifactByDataFormat[dataFormat] = id
		}
		for _, language := range artifact.Languages {
			// Skip languages in common dependencies since they are always available to integrations
			if artifact.ArtifactID != "camel-base" {
				catalog.languageDependencies[language] = getDependency(artifact, catalog.Runtime.Provider)
			}
		}
		for _, javaType := range artifact.JavaTypes {
			// Skip types in common dependencies since they are always available to integrations
			if artifact.ArtifactID != "camel-base" {
				catalog.javaTypeDependencies[javaType] = getDependency(artifact, catalog.Runtime.Provider)
			}
		}
	}

	return &catalog
}

// RuntimeCatalog --.
type RuntimeCatalog struct {
	v1.CamelCatalogSpec

	artifactByScheme     map[string]string
	artifactByDataFormat map[string]string
	schemesByID          map[string]v1.CamelScheme
	languageDependencies map[string]string
	javaTypeDependencies map[string]string
}

// HasArtifact --.
func (c *RuntimeCatalog) HasArtifact(artifact string) bool {
	if !strings.HasPrefix(artifact, "camel-") {
		artifact = "camel-" + artifact
	}

	_, ok := c.Artifacts[artifact]

	return ok
}

// GetArtifactByScheme returns the artifact corresponding to the given component scheme.
func (c *RuntimeCatalog) GetArtifactByScheme(scheme string) *v1.CamelArtifact {
	if id, ok := c.artifactByScheme[scheme]; ok {
		if artifact, present := c.Artifacts[id]; present {
			return &artifact
		}
	}
	return nil
}

// GetArtifactByDataFormat returns the artifact corresponding to the given data format.
func (c *RuntimeCatalog) GetArtifactByDataFormat(dataFormat string) *v1.CamelArtifact {
	if id, ok := c.artifactByDataFormat[dataFormat]; ok {
		if artifact, present := c.Artifacts[id]; present {
			return &artifact
		}
	}
	return nil
}

// GetScheme returns the scheme definition for the given scheme id.
func (c *RuntimeCatalog) GetScheme(id string) (v1.CamelScheme, bool) {
	scheme, ok := c.schemesByID[id]
	return scheme, ok
}

// GetLanguageDependency returns the maven dependency for the given language name.
func (c *RuntimeCatalog) GetLanguageDependency(language string) (string, bool) {
	language, ok := c.languageDependencies[language]
	return language, ok
}

// GetJavaTypeDependency returns the maven dependency for the given type name.
func (c *RuntimeCatalog) GetJavaTypeDependency(camelType string) (string, bool) {
	javaType, ok := c.javaTypeDependencies[camelType]
	return javaType, ok
}

// GetCamelVersion returns the Camel version the runtime is based on.
func (c *RuntimeCatalog) GetCamelVersion() string {
	return c.Runtime.Metadata["camel.version"]
}

// VisitArtifacts --.
func (c *RuntimeCatalog) VisitArtifacts(visitor func(string, v1.CamelArtifact) bool) {
	for id, artifact := range c.Artifacts {
		if !visitor(id, artifact) {
			break
		}
	}
}

// VisitSchemes --.
func (c *RuntimeCatalog) VisitSchemes(visitor func(string, v1.CamelScheme) bool) {
	for id, scheme := range c.schemesByID {
		if !visitor(id, scheme) {
			break
		}
	}
}

// DecodeComponent parses an URI and return a camel artifact and a scheme.
func (c *RuntimeCatalog) DecodeComponent(uri string) (*v1.CamelArtifact, *v1.CamelScheme) {
	uriSplit := strings.SplitN(uri, ":", 2)
	if len(uriSplit) < 2 {
		return nil, nil
	}
	uriStart := uriSplit[0]
	scheme, ok := c.GetScheme(uriStart)
	var schemeRef *v1.CamelScheme
	if ok {
		schemeRef = &scheme
	}
	return c.GetArtifactByScheme(uriStart), schemeRef
}
