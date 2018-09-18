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

package catalog

import (
	"github.com/apache/camel-k/deploy"
	"gopkg.in/yaml.v2"
)

// Catalog --
type Catalog struct {
	Version    string               `yaml:"version"`
	Components map[string]Component `yaml:"components"`
}

// Dependency --
type Dependency struct {
	GroupID    string `yaml:"groupId"`
	ArtifactID string `yaml:"artifactId"`
	Version    string `yaml:"version"`
}

// Component --
type Component struct {
	Dependency Dependency `yaml:"dependency"`
	Schemes    []string   `yaml:"schemes"`
}

func init() {
	data := deploy.Resources["camel-catalog.yaml"]
	if err := yaml.Unmarshal([]byte(data), &Runtime); err != nil {
		panic(err)
	}
}

// Runtime --
var Runtime Catalog
