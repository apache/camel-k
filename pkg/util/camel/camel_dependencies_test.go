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
	"strings"
	"testing"

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
