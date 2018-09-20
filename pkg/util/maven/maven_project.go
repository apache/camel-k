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

// Dependency represent a maven's dependency
type Dependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version,omitempty"`
	Type       string `xml:"type,omitempty"`
	Classifier string `xml:"classifier,omitempty"`
	Scope      string `xml:"scope,omitempty"`
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
	ID        string    `xml:"id"`
	Name      string    `xml:"name"`
	URL       string    `xml:"url"`
	Snapshots Snapshots `xml:"snapshots"`
	Releases  Releases  `xml:"releases"`
}

// Snapshots --
type Snapshots struct {
	Enabled      bool   `xml:"enabled"`
	UpdatePolicy string `xml:"updatePolicy"`
}

// Releases --
type Releases struct {
	Enabled      bool   `xml:"enabled"`
	UpdatePolicy string `xml:"updatePolicy"`
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

/*
 <plugin>
        <groupId>org.apache.camel.k</groupId>
        <artifactId>camel-k-runtime-dependency-lister</artifactId>
        <version>0.0.3-SNAPSHOT</version>
        <executions>
          <execution>
            <id>generate-dependency-list</id>
            <phase>initialize</phase>
            <goals>
              <goal>generate-dependency-list</goal>
            </goals>
          </execution>
        </executions>
      </plugin>
*/
