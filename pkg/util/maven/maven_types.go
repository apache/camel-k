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
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"time"
)

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
	Enabled        bool   `xml:"enabled"`
	UpdatePolicy   string `xml:"updatePolicy,omitempty"`
	ChecksumPolicy string `xml:"checksumPolicy,omitempty"`
}

// Mirror --
type Mirror struct {
	ID       string `xml:"id"`
	Name     string `xml:"name,omitempty"`
	URL      string `xml:"url"`
	MirrorOf string `xml:"mirrorOf"`
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
	ID    string   `xml:"id,omitempty"`
	Phase string   `xml:"phase,omitempty"`
	Goals []string `xml:"goals>goal,omitempty"`
}

// Properties --
type Properties map[string]string

type propertiesEntry struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

// AddAll --
func (m Properties) AddAll(properties map[string]string) {
	for k, v := range properties {
		m[k] = v
	}
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

// NewContext --
func NewContext(buildDir string, project Project) Context {
	return Context{
		Path:                buildDir,
		Project:             project,
		AdditionalArguments: make([]string, 0),
		AdditionalEntries:   make(map[string]interface{}),
	}
}

// Context --
type Context struct {
	Path                string
	Project             Project
	ExtraMavenOpts      []string
	SettingsContent     []byte
	AdditionalArguments []string
	AdditionalEntries   map[string]interface{}
	Timeout             time.Duration
	LocalRepository     string
	Stdout              io.Writer
}

// AddEntry --
func (c *Context) AddEntry(id string, entry interface{}) {
	if c.AdditionalEntries == nil {
		c.AdditionalEntries = make(map[string]interface{})
	}

	c.AdditionalEntries[id] = entry
}

// AddArgument --
func (c *Context) AddArgument(argument string) {
	c.AdditionalArguments = append(c.AdditionalArguments, argument)
}

// AddArgumentf --
func (c *Context) AddArgumentf(format string, args ...interface{}) {
	c.AdditionalArguments = append(c.AdditionalArguments, fmt.Sprintf(format, args...))
}

// AddArguments --
func (c *Context) AddArguments(arguments ...string) {
	c.AdditionalArguments = append(c.AdditionalArguments, arguments...)
}

// AddSystemProperty --
func (c *Context) AddSystemProperty(name string, value string) {
	c.AddArgumentf("-D%s=%s", name, value)
}

// Settings represent a maven settings
type Settings struct {
	XMLName           xml.Name
	XMLNs             string    `xml:"xmlns,attr"`
	XMLNsXsi          string    `xml:"xmlns:xsi,attr"`
	XsiSchemaLocation string    `xml:"xsi:schemaLocation,attr"`
	LocalRepository   string    `xml:"localRepository"`
	Profiles          []Profile `xml:"profiles>profile,omitempty"`
	Mirrors           []Mirror  `xml:"mirrors>mirror,omitempty"`
}

// MarshalBytes --
func (s Settings) MarshalBytes() ([]byte, error) {
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)

	e := xml.NewEncoder(w)
	e.Indent("", "  ")

	err := e.Encode(s)
	if err != nil {
		return []byte{}, err
	}

	return w.Bytes(), nil
}

// Project represent a maven project
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
	Repositories         []Repository          `xml:"repositories>repository,omitempty"`
	PluginRepositories   []Repository          `xml:"pluginRepositories>pluginRepository,omitempty"`
	Build                *Build                `xml:"build,omitempty"`
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

// Profile --
type Profile struct {
	ID                 string       `xml:"id"`
	Activation         Activation   `xml:"activation,omitempty"`
	Properties         Properties   `xml:"properties,omitempty"`
	Repositories       []Repository `xml:"repositories>repository,omitempty"`
	PluginRepositories []Repository `xml:"pluginRepositories>pluginRepository,omitempty"`
}

// Activation --
type Activation struct {
	ActiveByDefault bool                `xml:"activeByDefault"`
	Property        *PropertyActivation `xml:"property,omitempty"`
}

// PropertyActivation --
type PropertyActivation struct {
	Name  string `xml:"name"`
	Value string `xml:"value"`
}
