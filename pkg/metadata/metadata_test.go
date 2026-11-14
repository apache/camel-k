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

package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"
)

func TestExtractAllMergesBeanDefinitionsFromYAMLAndXML(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	sources := []v1.SourceSpec{
		{
			DataSpec: v1.DataSpec{
				Name: "clients.yaml",
				Content: `
- beans:
    - name: yamlClient
      type: com.example.YamlClient
      properties:
        endpoint: https://yaml.example
        timeout: 30
`,
			},
			Language: v1.LanguageYaml,
		},
		{
			DataSpec: v1.DataSpec{
				Name: "clients.xml",
				Content: `
<camel>
  <bean name="xmlClient" type="com.example.XMLClient">
    <properties>
      <property key="endpoint" value="https://xml.example"/>
      <property key="timeout" value="60"/>
    </properties>
  </bean>
</camel>
`,
			},
			Language: v1.LanguageXML,
		},
	}

	meta, err := ExtractAll(catalog, sources)
	require.NoError(t, err)

	require.Len(t, meta.Beans, 2)
	assert.Equal(t, "yamlClient", meta.Beans[0].Name)
	assert.Equal(t, "com.example.YamlClient", meta.Beans[0].Type)
	assert.Equal(t, map[string]string{
		"endpoint": "https://yaml.example",
		"timeout":  "30",
	}, meta.Beans[0].Properties)

	assert.Equal(t, "xmlClient", meta.Beans[1].Name)
	assert.Equal(t, "com.example.XMLClient", meta.Beans[1].Type)
	assert.Equal(t, map[string]string{
		"endpoint": "https://xml.example",
		"timeout":  "60",
	}, meta.Beans[1].Properties)
}
