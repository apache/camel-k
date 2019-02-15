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
	Properties           Properties           `xml:"properties,omitempty"`
	DependencyManagement DependencyManagement `xml:"dependencyManagement"`
	Dependencies         []Dependency         `xml:"dependencies>dependency,omitempty"`
	Repositories         []Repository         `xml:"repositories>repository,omitempty"`
	PluginRepositories   []Repository         `xml:"pluginRepositories>pluginRepository,omitempty"`
	Build                Build                `xml:"build,omitempty"`
}

// Exclusion represent a maven's dependency exlucsion
type Exclusion struct {
	GroupID    string `xml:"groupId" yaml:"groupId"`
	ArtifactID string `xml:"artifactId" yaml:"artifactId"`
}

// DependencyManagement represent maven's dependency management block
type DependencyManagement struct {
	Dependencies []Dependency `xml:"dependencies>dependency,omitempty"`
}

// Dependency represent a maven's dependency
type Dependency struct {
	GroupID    string       `xml:"groupId" yaml:"groupId"`
	ArtifactID string       `xml:"artifactId" yaml:"artifactId"`
	Version    string       `xml:"version,omitempty" yaml:"version,omitempty"`
	Type       string       `xml:"type,omitempty" yaml:"type,omitempty"`
	Classifier string       `xml:"classifier,omitempty" yaml:"classifier,omitempty"`
	Scope      string       `xml:"scope,omitempty" yaml:"scope,omitempty"`
	Exclusions *[]Exclusion `xml:"exclusions>exclusion,omitempty" yaml:"exclusions,omitempty"`
}

// Repository --
type Repository struct {
	ID        string           `xml:"id"`
	Name      string           `xml:"name,omitempty"`
	URL       string           `xml:"url"`
	Snapshots RepositoryPolicy `xml:"snapshots,omitempty"`
	Releases  RepositoryPolicy `xml:"releases,omitempty"`
}

// RepositoryPolicy --
type RepositoryPolicy struct {
	Enabled      bool   `xml:"enabled"`
	UpdatePolicy string `xml:"updatePolicy,omitempty"`
}

// Build --
type Build struct {
	DefaultGoal string   `xml:"defaultGoal,omitempty"`
	Plugins     []Plugin `xml:"plugins>plugin,omitempty"`
}

// Plugin --
type Plugin struct {
	GroupID      string       `xml:"groupId"`
	ArtifactID   string       `xml:"artifactId"`
	Version      string       `xml:"version,omitempty"`
	Executions   []Execution  `xml:"executions>execution,omitempty"`
	Dependencies []Dependency `xml:"dependencies>dependency,omitempty"`
}

// Execution --
type Execution struct {
	ID    string   `xml:"id"`
	Phase string   `xml:"phase"`
	Goals []string `xml:"goals>goal,omitempty"`
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
