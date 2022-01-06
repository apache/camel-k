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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

type Mirror struct {
	ID       string `xml:"id"`
	Name     string `xml:"name,omitempty"`
	URL      string `xml:"url"`
	MirrorOf string `xml:"mirrorOf"`
}

type Build struct {
	DefaultGoal string             `xml:"defaultGoal,omitempty"`
	Plugins     []Plugin           `xml:"plugins>plugin,omitempty"`
	Extensions  []v1.MavenArtifact `xml:"extensions>extension,omitempty"`
}

type Plugin struct {
	GroupID      string       `xml:"groupId"`
	ArtifactID   string       `xml:"artifactId"`
	Version      string       `xml:"version,omitempty"`
	Executions   []Execution  `xml:"executions>execution,omitempty"`
	Dependencies []Dependency `xml:"dependencies>dependency,omitempty"`
}

type Execution struct {
	ID    string   `xml:"id,omitempty"`
	Phase string   `xml:"phase,omitempty"`
	Goals []string `xml:"goals>goal,omitempty"`
}

type Properties map[string]string

// Settings models a Maven settings.
type Settings struct {
	XMLName           xml.Name
	XMLNs             string    `xml:"xmlns,attr"`
	XMLNsXsi          string    `xml:"xmlns:xsi,attr"`
	XsiSchemaLocation string    `xml:"xsi:schemaLocation,attr"`
	LocalRepository   string    `xml:"localRepository"`
	Profiles          []Profile `xml:"profiles>profile,omitempty"`
	Proxies           []Proxy   `xml:"proxies>proxy,omitempty"`
	Mirrors           []Mirror  `xml:"mirrors>mirror,omitempty"`
}

// Project models a Maven project.
type Project struct {
	XMLName              xml.Name
	XMLNs                string                `xml:"xmlns,attr"`
	XMLNsXsi             string                `xml:"xmlns:xsi,attr"`
	XsiSchemaLocation    string                `xml:"xsi:schemaLocation,attr"`
	ModelVersion         string                `xml:"modelVersion"`
	GroupID              string                `xml:"groupId"`
	ArtifactID           string                `xml:"artifactId"`
	Version              string                `xml:"version"`
	Properties           Properties            `xml:"properties,omitempty"`
	DependencyManagement *DependencyManagement `xml:"dependencyManagement"`
	Dependencies         []Dependency          `xml:"dependencies>dependency,omitempty"`
	Repositories         []v1.Repository       `xml:"repositories>repository,omitempty"`
	PluginRepositories   []v1.Repository       `xml:"pluginRepositories>pluginRepository,omitempty"`
	Build                *Build                `xml:"build,omitempty"`
}

// Exclusion models a dependency exclusion.
type Exclusion struct {
	GroupID    string `xml:"groupId" yaml:"groupId"`
	ArtifactID string `xml:"artifactId" yaml:"artifactId"`
}

// DependencyManagement models dependency management.
type DependencyManagement struct {
	Dependencies []Dependency `xml:"dependencies>dependency,omitempty"`
}

// Dependency models a dependency.
type Dependency struct {
	GroupID    string       `xml:"groupId" yaml:"groupId"`
	ArtifactID string       `xml:"artifactId" yaml:"artifactId"`
	Version    string       `xml:"version,omitempty" yaml:"version,omitempty"`
	Type       string       `xml:"type,omitempty" yaml:"type,omitempty"`
	Classifier string       `xml:"classifier,omitempty" yaml:"classifier,omitempty"`
	Scope      string       `xml:"scope,omitempty" yaml:"scope,omitempty"`
	Exclusions *[]Exclusion `xml:"exclusions>exclusion,omitempty" yaml:"exclusions,omitempty"`
}

type Profile struct {
	ID                 string          `xml:"id"`
	Activation         Activation      `xml:"activation,omitempty"`
	Properties         Properties      `xml:"properties,omitempty"`
	Repositories       []v1.Repository `xml:"repositories>repository,omitempty"`
	PluginRepositories []v1.Repository `xml:"pluginRepositories>pluginRepository,omitempty"`
}

type Activation struct {
	ActiveByDefault bool                `xml:"activeByDefault"`
	Property        *PropertyActivation `xml:"property,omitempty"`
}

type PropertyActivation struct {
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

type Proxy struct {
	ID            string `xml:"id"`
	Active        bool   `xml:"active"`
	Protocol      string `xml:"protocol"`
	Host          string `xml:"host"`
	Port          string `xml:"port,omitempty"`
	Username      string `xml:"username,omitempty"`
	Password      string `xml:"password,omitempty"`
	NonProxyHosts string `xml:"nonProxyHosts,omitempty"`
}
