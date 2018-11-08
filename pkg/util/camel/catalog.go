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
	"github.com/apache/camel-k/deploy"
	"gopkg.in/yaml.v2"
)

// Catalog --
type Catalog struct {
	Version          string              `yaml:"version"`
	Artifacts        map[string]Artifact `yaml:"artifacts"`
	artifactByScheme map[string]string   `yaml:"-"`
}

// Artifact --
type Artifact struct {
	GroupID     string   `yaml:"groupId"`
	ArtifactID  string   `yaml:"artifactId"`
	Version     string   `yaml:"version"`
	Schemes     []string `yaml:"schemes"`
	Languages   []string `yaml:"languages"`
	DataFormats []string `yaml:"dataformats"`
}

func init() {
	data := deploy.Resources["camel-catalog.yaml"]
	if err := yaml.Unmarshal([]byte(data), &Runtime); err != nil {
		panic(err)
	}
	// Adding embedded artifacts
	for k, v := range EmbeddedArtifacts() {
		Runtime.Artifacts[k] = v
	}

	Runtime.artifactByScheme = make(map[string]string)
	for id, artifact := range Runtime.Artifacts {
		for _, scheme := range artifact.Schemes {
			Runtime.artifactByScheme[scheme] = id
		}
	}
}

// GetArtifactByScheme returns the artifact corresponding to the given component scheme
func (c Catalog) GetArtifactByScheme(scheme string) *Artifact {
	if id, ok := c.artifactByScheme[scheme]; ok {
		if artifact, present := c.Artifacts[id]; present {
			return &artifact
		}
	}
	return nil
}

// Runtime --
var Runtime Catalog
