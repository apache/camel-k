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

package maven

import (
	"encoding/xml"
	"strings"
)

// Project represent a maven project
type Project struct {
	XMLName              xml.Name
	XMLNs                string               `xml:"xmlns,attr"`
	XMLNsXsi             string               `xml:"xmlns:xsi,attr"`
	XsiSchemaLocation    string               `xml:"xsi:schemaLocation,attr"`
	ModelVersion         string               `xml:"modelVersion"`
	GroupID              string               `xml:"groupId"`
	ArtifactID           string               `xml:"artifactId"`
	Version              string               `xml:"version"`
	Properties           Properties           `xml:"properties,omitempty"`
	DependencyManagement DependencyManagement `xml:"dependencyManagement"`
	Dependencies         Dependencies         `xml:"dependencies"`
	Repositories         Repositories         `xml:"repositories"`
	PluginRepositories   PluginRepositories   `xml:"pluginRepositories"`
}

// DependencyManagement represent maven's dependency management block
type DependencyManagement struct {
	Dependencies Dependencies `xml:"dependencies"`
}

// Dependencies --
type Dependencies struct {
	Dependencies []Dependency `xml:"dependency"`
}

// Add a dependency to maven's dependencies
func (deps *Dependencies) Add(dep Dependency) {
	for _, d := range deps.Dependencies {
		// Check if the given dependency is already included in the dependency list
		if d == dep {
			return
		}
	}

	deps.Dependencies = append(deps.Dependencies, dep)
}

// AddGAV a dependency to maven's dependencies
func (deps *Dependencies) AddGAV(groupID string, artifactID string, version string) {
	deps.Add(NewDependency(groupID, artifactID, version))
}

// AddEncodedGAV a dependency to maven's dependencies
func (deps *Dependencies) AddEncodedGAV(gav string) {
	if d, err := ParseGAV(gav); err == nil {
		// TODO: error handling
		deps.Add(d)
	}
}

// Exclusion represent a maven's dependency exlucsion
type Exclusion struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
}

// Exclusions --
type Exclusions struct {
	Exclusions []Exclusion `xml:"exclusion"`
}

// Dependency represent a maven's dependency
type Dependency struct {
	GroupID    string      `xml:"groupId"`
	ArtifactID string      `xml:"artifactId"`
	Version    string      `xml:"version,omitempty"`
	Type       string      `xml:"type,omitempty"`
	Classifier string      `xml:"classifier,omitempty"`
	Scope      string      `xml:"scope,omitempty"`
	Exclusions *Exclusions `xml:"exclusions,omitempty"`
}

// NewDependency create an new dependency from the given gav info
func NewDependency(groupID string, artifactID string, version string) Dependency {
	return Dependency{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Version:    version,
		Type:       "jar",
		Classifier: "",
	}
}

// Repositories --
type Repositories struct {
	Repositories []Repository `xml:"repository"`
}

// PluginRepositories --
type PluginRepositories struct {
	Repositories []Repository `xml:"pluginRepository"`
}

// Repository --
type Repository struct {
	ID        string           `xml:"id"`
	Name      string           `xml:"name,omitempty"`
	URL       string           `xml:"url"`
	Snapshots RepositoryPolicy `xml:"snapshots,omitempty"`
	Releases  RepositoryPolicy `xml:"releases,omitempty"`
}

//
// NewRepository parse the given repo url ang generated the related struct.
//
// The repository can be customized by appending @instruction to the repository
// uri, as example:
//
//     http://my-nexus:8081/repository/publicc@id=my-repo@snapshots
//
// Will enable snapshots and sets the repo it to my-repo
//
func NewRepository(repo string) Repository {
	r := Repository{
		URL: repo,
		Releases: RepositoryPolicy{
			Enabled: true,
		},
		Snapshots: RepositoryPolicy{
			Enabled: false,
		},
	}

	if idx := strings.Index(repo, "@"); idx != -1 {
		r.URL = repo[:idx]

		for _, attribute := range strings.Split(repo[idx+1:], "@") {
			if attribute == "snapshots" {
				r.Snapshots.Enabled = true
			} else if attribute == "noreleases" {
				r.Releases.Enabled = false
			} else if strings.HasPrefix(attribute, "id=") {
				r.ID = attribute[3:]
			}
		}
	}

	return r
}

// RepositoryPolicy --
type RepositoryPolicy struct {
	Enabled      bool   `xml:"enabled"`
	UpdatePolicy string `xml:"updatePolicy,omitempty"`
}

// Build --
type Build struct {
	Plugins Plugins `xml:"plugins,omitempty"`
}

// Plugin --
type Plugin struct {
	GroupID    string     `xml:"groupId"`
	ArtifactID string     `xml:"artifactId"`
	Version    string     `xml:"version,omitempty"`
	Executions Executions `xml:"executions"`
}

// Plugins --
type Plugins struct {
	Plugins []Plugin `xml:"plugin"`
}

// Execution --
type Execution struct {
	ID    string `xml:"id"`
	Phase string `xml:"phase"`
	Goals Goals  `xml:"goals,omitempty"`
}

// Executions --
type Executions struct {
	Executions []Execution `xml:"execution"`
}

// Goals --
type Goals struct {
	Goals []string `xml:"goal"`
}

// Properties --
type Properties map[string]string

type propertiesEntry struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

// MarshalXML --
func (m Properties) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(m) == 0 {
		return nil
	}

	err := e.EncodeToken(start)
	if err != nil {
		return err
	}

	for k, v := range m {
		if err := e.Encode(propertiesEntry{XMLName: xml.Name{Local: k}, Value: v}); err != nil {
			return err
		}
	}

	return e.EncodeToken(start.End())
}
