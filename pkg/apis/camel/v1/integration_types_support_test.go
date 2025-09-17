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

package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllLanguages(t *testing.T) {
	assert.Contains(t, Languages, LanguageJavaSource)
	assert.Contains(t, Languages, LanguageXML)
	assert.Contains(t, Languages, LanguageYaml)
}

func TestLanguageFromName(t *testing.T) {
	for _, l := range Languages {
		language := l
		t.Run(string(language), func(t *testing.T) {
			code := SourceSpec{
				DataSpec: DataSpec{
					Name: fmt.Sprintf("code.%s", language),
				},
			}

			if language != code.InferLanguage() {
				t.Errorf("got %s, want %s", code.InferLanguage(), language)
			}
		})
	}
}

func TestLanguageAlreadySet(t *testing.T) {
	code := SourceSpec{
		DataSpec: DataSpec{
			Name: "Request.java",
		},
		Language: LanguageJavaSource,
	}
	assert.Equal(t, LanguageJavaSource, code.InferLanguage())
}

func TestAddDependency(t *testing.T) {
	integration := IntegrationSpec{}
	integration.AddDependency("camel:file")
	assert.Equal(t, []string{"camel:file"}, integration.Dependencies)
	// adding the same dependency twice won't duplicate it in the list
	integration.AddDependency("camel:file")
	assert.Equal(t, []string{"camel:file"}, integration.Dependencies)

	integration = IntegrationSpec{}
	integration.AddDependency("mvn:com.my:company")
	assert.Equal(t, integration.Dependencies, []string{"mvn:com.my:company"})

	integration = IntegrationSpec{}
	integration.AddDependency("file:dep")
	assert.Equal(t, integration.Dependencies, []string{"file:dep"})
}

func TestGetConfigurationProperty(t *testing.T) {
	integration := IntegrationSpec{}
	integration.AddConfiguration("property", "key1=value1")
	integration.AddConfiguration("property", "key2 = value2")
	integration.AddConfiguration("property", "key3 = value with trailing space ")
	integration.AddConfiguration("property", "key4 =  value with leading space")
	integration.AddConfiguration("property", "key5 = ")
	integration.AddConfiguration("property", "key6=")

	missing := integration.GetConfigurationProperty("missing")
	assert.Equal(t, "", missing)
	v1 := integration.GetConfigurationProperty("key")
	assert.Equal(t, "value1", v1)
	v2 := integration.GetConfigurationProperty("key2")
	assert.Equal(t, "value2", v2)
	v3 := integration.GetConfigurationProperty("key3")
	assert.Equal(t, "value with trailing space ", v3)
	v4 := integration.GetConfigurationProperty("key4")
	assert.Equal(t, " value with leading space", v4)
	v5 := integration.GetConfigurationProperty("key5")
	assert.Equal(t, "", v5)
	v6 := integration.GetConfigurationProperty("key6")
	assert.Equal(t, "", v6)
}

func TestManagedBuild(t *testing.T) {
	integration := Integration{
		Spec: IntegrationSpec{},
	}
	assert.True(t, integration.IsManagedBuild())
	integration.Spec.Traits = Traits{
		Container: &trait.ContainerTrait{},
	}
	assert.True(t, integration.IsManagedBuild())
	integration.Spec.Traits = Traits{
		Container: &trait.ContainerTrait{
			Image: "registry.io/my-org/my-image",
		},
	}
	assert.False(t, integration.IsManagedBuild())
	integration.Spec.Traits = Traits{
		Container: &trait.ContainerTrait{
			Image: "10.100.107.57/camel-k/camel-k-kit-cr82ehho23os73cgua70@sha256:13e5a67d37665710c0bdd89701c7ae10aee393b00f5e4e09dc8ecc234763e7c2",
		},
	}
	assert.True(t, integration.IsManagedBuild())
}

func TestReadWriteYaml(t *testing.T) {
	// yaml in conventional form as marshalled by the go runtime
	yaml := `- from:
    parameters:
      period: 3600001
    steps:
    - to: log:info
    uri: timer:tick
`

	yamlReader := bytes.NewReader([]byte(yaml))
	flows, err := FromYamlDSL(yamlReader)
	require.NoError(t, err)
	assert.NotNil(t, flows)
	assert.Len(t, flows, 1)

	flow := map[string]interface{}{}
	err = json.Unmarshal(flows[0].RawMessage, &flow)
	require.NoError(t, err)

	assert.NotNil(t, flow["from"])
	assert.Nil(t, flow["xx"])

	data, err := ToYamlDSL(flows)
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, yaml, string(data))
}
