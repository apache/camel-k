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
	"strings"
	"testing"

	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeDependency(t *testing.T) {
	assert.Equal(t, "camel:file", NormalizeDependency("camel-file"))
	assert.Equal(t, "camel:file", NormalizeDependency("camel:file"))
	assert.Equal(t, "camel:file", NormalizeDependency("camel-quarkus-file"))
	assert.Equal(t, "camel:file", NormalizeDependency("camel-quarkus:file"))
	assert.Equal(t, "camel-k:knative", NormalizeDependency("camel-k-knative"))
	assert.Equal(t, "camel-k:knative", NormalizeDependency("camel-k:knative"))
	assert.Equal(t, "mvn:org.apache.camel:camel-file", NormalizeDependency("mvn:org.apache.camel:camel-file"))
	assert.Equal(t, "mvn:org.apache.camel.quarkus:camel-quarkus-file", NormalizeDependency("mvn:org.apache.camel.quarkus:camel-quarkus-file"))
	assert.Equal(t, "mvn:org.apache.camel:camel-k-knative", NormalizeDependency("mvn:org.apache.camel:camel-k-knative"))
}

func TestValidateDependency(t *testing.T) {
	catalog, err := DefaultCatalog()
	assert.Nil(t, err)

	output := strings.Builder{}
	ValidateDependency(catalog, "", &output)
	assert.Equal(t, "", output.String())

	output.Reset()
	ValidateDependency(catalog, "camel:file", &output)
	assert.Equal(t, "", output.String())

	output.Reset()
	ValidateDependency(catalog, "camel-quarkus-file", &output)
	assert.Equal(t, "", output.String())

	output.Reset()
	ValidateDependency(catalog, "camel-quarkus:file", &output)
	assert.Equal(t, "", output.String())

	output.Reset()
	ValidateDependency(catalog, "camel:unknown", &output)
	assert.Equal(t, "Warning: dependency camel:unknown not found in Camel catalog\n", output.String())

	output.Reset()
	ValidateDependency(catalog, "mvn:org.apache.camel:camel-foo", &output)
	assert.Equal(t, "Warning: do not use mvn:org.apache.camel:camel-foo. Use camel:foo instead\n", output.String())

	output.Reset()
	ValidateDependency(catalog, "mvn:org.apache.camel.quarkus:camel-quarkus-foo", &output)
	assert.Equal(t, "Warning: do not use mvn:org.apache.camel.quarkus:camel-quarkus-foo. Use camel:foo instead\n", output.String())
}

func TestManageIntegrationDependencies(t *testing.T) {
	catalog, err := DefaultCatalog()
	assert.Nil(t, err)

	tests := []struct {
		name         string
		dependencies []string
		coordinates  string
	}{
		{
			name: "basic_camel",
			dependencies: []string{
				"camel:direct",
				"camel:log",
				"camel:core",
			},
			coordinates: "org.apache.camel.quarkus:camel-quarkus-direct," +
				"org.apache.camel.quarkus:camel-quarkus-log," +
				"org.apache.camel.quarkus:camel-quarkus-core",
		},
		{
			name: "camel_quarkus",
			dependencies: []string{
				"camel:direct",
				"camel-quarkus:log",
				"camel:camel-quarkus-core",
			},
			coordinates: "org.apache.camel.quarkus:camel-quarkus-direct," +
				"org.apache.camel.quarkus:camel-quarkus-log," +
				"org.apache.camel.quarkus:camel-quarkus-core",
		},
		{
			name: "camel_k",
			dependencies: []string{
				"camel:direct",
				"camel-k:webhook",
			},
			coordinates: "org.apache.camel.quarkus:camel-quarkus-direct," +
				"org.apache.camel.k:camel-k-webhook",
		},
		{
			name: "not_in_catalog",
			dependencies: []string{
				"camel:direct",
				"camel:resiliance4j",
			},
			coordinates: "org.apache.camel.quarkus:camel-quarkus-direct," +
				"org.apache.camel.quarkus:camel-quarkus-resiliance4j",
		},
		{
			name: "mvn",
			dependencies: []string{
				"mvn:org.foo:bar",
				"mvn:org.apache.camel:camel-resiliance4j",
			},
			coordinates: "org.foo:bar," +
				"org.apache.camel:camel-resiliance4j",
		},
		{
			name: "jitpack",
			dependencies: []string{
				"github:apache/camel-sample/1.0",
			},
			coordinates: "com.github.apache:camel-sample",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project := maven.Project{}

			err = ManageIntegrationDependencies(&project, test.dependencies, catalog)
			assert.Nil(t, err)

			coordinates := strings.Builder{}
			for i, d := range project.Dependencies {
				if i == 0 {
					_, err = fmt.Fprintf(&coordinates, "%s:%s", d.GroupID, d.ArtifactID)
					assert.Nil(t, err)
				} else {
					_, err = fmt.Fprintf(&coordinates, ",%s:%s", d.GroupID, d.ArtifactID)
					assert.Nil(t, err)
				}
			}
			assert.Equal(t, test.coordinates, coordinates.String(), coordinates)
		})
	}
}
