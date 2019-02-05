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
	"fmt"
	"sync"

	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/coreos/go-semver/semver"
	"gopkg.in/yaml.v2"
)

// Artifact --
type Artifact struct {
	maven.Dependency `yaml:",inline"`
	Schemes          []Scheme           `yaml:"schemes"`
	Languages        []string           `yaml:"languages"`
	DataFormats      []string           `yaml:"dataformats"`
	Dependencies     []maven.Dependency `yaml:"dependencies"`
}

// Scheme --
type Scheme struct {
	ID      string `yaml:"id"`
	Passive bool   `yaml:"passive"`
	HTTP    bool   `yaml:"http"`
}

// RuntimeCatalog --
type RuntimeCatalog struct {
	Version   string              `yaml:"version"`
	Artifacts map[string]Artifact `yaml:"artifacts"`

	artifactByScheme map[string]string
	schemesByID      map[string]Scheme
}

// HasArtifact --
func (c RuntimeCatalog) HasArtifact(artifact string) bool {
	_, ok := c.Artifacts[artifact]
	if !ok {
		_, ok = c.Artifacts["camel-"+artifact]
	}

	return ok
}

// GetArtifactByScheme returns the artifact corresponding to the given component scheme
func (c RuntimeCatalog) GetArtifactByScheme(scheme string) *Artifact {
	if id, ok := c.artifactByScheme[scheme]; ok {
		if artifact, present := c.Artifacts[id]; present {
			return &artifact
		}
	}
	return nil
}

// GetScheme returns the scheme definition for the given scheme id
func (c RuntimeCatalog) GetScheme(id string) (Scheme, bool) {
	scheme, ok := c.schemesByID[id]
	return scheme, ok
}

// VisitArtifacts --
func (c RuntimeCatalog) VisitArtifacts(visitor func(string, Artifact) bool) {
	for id, artifact := range c.Artifacts {
		if !visitor(id, artifact) {
			break
		}
	}
}

// VisitSchemes --
func (c RuntimeCatalog) VisitSchemes(visitor func(string, Scheme) bool) {
	for id, scheme := range c.schemesByID {
		if !visitor(id, scheme) {
			break
		}
	}
}

// ******************************
//
//
//
// ******************************

var defaultCatalog RuntimeCatalog
var catalogs map[string]RuntimeCatalog
var catalogsLock sync.Mutex

func init() {
	c, err := loadCatalog("camel-catalog.yaml")
	if err != nil {
		panic(err)
	}

	defaultCatalog = *c

	catalogs = make(map[string]RuntimeCatalog)
}

func loadCatalog(resourceName string) (*RuntimeCatalog, error) {
	var catalog RuntimeCatalog

	data, ok := deploy.Resources[resourceName]
	if !ok {
		return nil, nil
	}

	if err := yaml.Unmarshal([]byte(data), &catalog); err != nil {
		return nil, err
	}

	catalog.artifactByScheme = make(map[string]string)
	catalog.schemesByID = make(map[string]Scheme)

	for id, artifact := range catalog.Artifacts {
		for _, scheme := range artifact.Schemes {
			scheme := scheme
			catalog.artifactByScheme[scheme.ID] = id
			catalog.schemesByID[scheme.ID] = scheme
		}
	}

	return &catalog, nil
}

// Catalog --
func Catalog(camelVersion string) *RuntimeCatalog {
	catalogsLock.Lock()
	defer catalogsLock.Unlock()

	if c, ok := catalogs[camelVersion]; ok {
		return &c
	}

	var c *RuntimeCatalog
	var r string
	var err error

	// try with the exact match
	r = fmt.Sprintf("camel-catalog-%s.yaml", camelVersion)
	c, err = loadCatalog(r)
	if err != nil {
		panic(err)
	}
	if c != nil {
		catalogs[camelVersion] = *c
		return c
	}

	// try with ${major}.${minor}
	sv := semver.New(camelVersion)
	r = fmt.Sprintf("camel-catalog-%d.%d.yaml", sv.Major, sv.Minor)
	c, err = loadCatalog(r)
	if err != nil {
		panic(err)
	}
	if c != nil {
		catalogs[camelVersion] = *c
		return c
	}

	// try with ${major}
	r = fmt.Sprintf("camel-catalog-%d.yaml", sv.Major)
	c, err = loadCatalog(r)
	if err != nil {
		panic(err)
	}
	if c != nil {
		catalogs[camelVersion] = *c
		return c
	}

	// return default
	return &defaultCatalog
}
