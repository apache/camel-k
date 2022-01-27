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

package dsl

import (
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestTemplateReadWrite(t *testing.T) {
	templateJSON := v1.Template{
		RawMessage: []byte(`
		{
			"beans": [
				{
					"name": "myBean",
					"type": "com.acme.MyBean"
				}
			],
			"from": {
				"uri": "kamelet:source",
				"steps": [
					{
						"uri": "log:info"
					}
				]
			}
		}
		`),
	}

	yamlBytes, err := TemplateToYamlDSL(templateJSON, "myid")
	assert.NoError(t, err)
	yaml := string(yamlBytes)
	expected := `- template:
    beans:
    - name: myBean
      type: com.acme.MyBean
    from:
      steps:
      - uri: log:info
      uri: kamelet:source
    id: myid
`
	assert.Equal(t, expected, yaml)
}
