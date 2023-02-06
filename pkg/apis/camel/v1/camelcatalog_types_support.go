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

package v1

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCamelCatalog --
func NewCamelCatalog(namespace string, name string) CamelCatalog {
	return CamelCatalog{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       CamelCatalogKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// NewCamelCatalogWithSpecs --
func NewCamelCatalogWithSpecs(namespace string, name string, spec CamelCatalogSpec) CamelCatalog {
	return CamelCatalog{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       CamelCatalogKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: spec,
	}
}

// NewCamelCatalogList --
func NewCamelCatalogList() CamelCatalogList {
	return CamelCatalogList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       CamelCatalogKind,
		},
	}
}

// GetRuntimeVersion returns the Camel K runtime version of the catalog.
func (c *CamelCatalogSpec) GetRuntimeVersion() string {
	return c.Runtime.Version
}

// GetCamelVersion returns the Camel version the runtime is based on.
func (c *CamelCatalogSpec) GetCamelVersion() string {
	return c.Runtime.Metadata["camel.version"]
}

// GetCamelQuarkusVersion returns the Camel Quarkus version the runtime is based on.
func (c *CamelCatalogSpec) GetCamelQuarkusVersion() string {
	return c.Runtime.Metadata["camel-quarkus.version"]
}

// GetQuarkusVersion returns the Quarkus version the runtime is based on.
func (c *CamelCatalogSpec) GetQuarkusVersion() string {
	return c.Runtime.Metadata["quarkus.version"]
}

// HasCapability checks if the given capability is present in the catalog.
func (c *CamelCatalogSpec) HasCapability(capability string) bool {
	_, ok := c.Runtime.Capabilities[capability]

	return ok
}

// GetDependencyID returns a Camel K recognizable maven dependency for the artifact
func (in *CamelArtifact) GetDependencyID() string {
	switch {
	case in.GroupID == "org.apache.camel.quarkus" && strings.HasPrefix(in.ArtifactID, "camel-quarkus-"):
		return "camel:" + in.ArtifactID[14:]
	case in.Version == "":
		return "mvn:" + in.GroupID + ":" + in.ArtifactID
	default:
		return "mvn:" + in.GroupID + ":" + in.ArtifactID + ":" + in.Version
	}
}

func (in *CamelArtifact) GetConsumerDependencyIDs(schemeID string) []string {
	return in.getDependencyIDs(schemeID, consumerScheme)
}

func (in *CamelArtifact) GetProducerDependencyIDs(schemeID string) []string {
	return in.getDependencyIDs(schemeID, producerScheme)
}

func (in *CamelArtifact) getDependencyIDs(schemeID string, scope func(CamelScheme) CamelSchemeScope) []string {
	ads := in.getDependencies(schemeID, scope)
	if ads == nil {
		return nil
	}
	deps := make([]string, 0, len(ads))
	for _, ad := range ads {
		deps = append(deps, ad.GetDependencyID())
	}
	return deps
}

func (in *CamelArtifact) GetConsumerDependencies(schemeID string) []CamelArtifactDependency {
	return in.getDependencies(schemeID, consumerScheme)
}

func (in *CamelArtifact) GetProducerDependencies(schemeID string) []CamelArtifactDependency {
	return in.getDependencies(schemeID, producerScheme)
}

func (in *CamelArtifact) getDependencies(schemeID string, scope func(CamelScheme) CamelSchemeScope) []CamelArtifactDependency {
	scheme := in.GetScheme(schemeID)
	if scheme == nil {
		return nil
	}
	return scope(*scheme).Dependencies
}

func (in *CamelArtifact) GetScheme(schemeID string) *CamelScheme {
	for _, scheme := range in.Schemes {
		if scheme.ID == schemeID {
			return &scheme
		}
	}
	return nil
}

func consumerScheme(scheme CamelScheme) CamelSchemeScope {
	return scheme.Consumer
}

func producerScheme(scheme CamelScheme) CamelSchemeScope {
	return scheme.Producer
}
