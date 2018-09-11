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

type ProjectDefinition struct {
	Project     Project
	JavaSources map[string]string
	Resources   map[string]string
	Env         map[string]string
}

type Project struct {
	XMLName           xml.Name
	XmlNs             string       `xml:"xmlns,attr"`
	XmlNsXsi          string       `xml:"xmlns:xsi,attr"`
	XsiSchemaLocation string       `xml:"xsi:schemaLocation,attr"`
	ModelVersion      string       `xml:"modelVersion"`
	GroupId           string       `xml:"groupId"`
	ArtifactId        string       `xml:"artifactId"`
	Version           string       `xml:"version"`
	Dependencies      Dependencies `xml:"dependencies"`
}

type Dependencies struct {
	Dependencies []Dependency `xml:"dependency"`
}

type Dependency struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version,omitempty"`
	Type       string `xml:"type,omitempty"`
	Classifier string `xml:"classifier,omitempty"`
}

func NewDependency(groupId string, artifactId string, version string) Dependency {
	return Dependency{
		GroupId:    groupId,
		ArtifactId: artifactId,
		Version:    version,
		Type:       "jar",
		Classifier: "",
	}
}
