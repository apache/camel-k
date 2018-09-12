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

package local

import (
	"testing"

	"github.com/apache/camel-k/pkg/build/api"
	"github.com/stretchr/testify/assert"
)

func TestProjectGeneration(t *testing.T) {
	source := api.BuildSource{
		Identifier: api.BuildIdentifier{
			Name:      "my-integration",
			Qualifier: "",
		},
		Code: api.Code{
			Name:     "my-code.js",
			Content:  `from("timer:start").to("log:end")`,
			Language: "",
		},
		Dependencies: []string{
			"camel:mail",
			"camel:camel-netty4",
			"mvn:org.apache.camel/camel-servicenow/2.21.1",
			"mvn:org.apache.camel/camel-salesforce",
		},
	}

	prj, err := generateProjectDefinition(source)
	assert.Nil(t, err)
	assert.NotNil(t, prj)
	assert.Equal(t, len(prj.Project.Dependencies.Dependencies), 5)
	assert.Equal(t, prj.Project.Dependencies.Dependencies[0].ArtifactId, "camel-k-runtime-jvm")
	assert.Equal(t, prj.Project.Dependencies.Dependencies[1].ArtifactId, "camel-mail")
	assert.Equal(t, prj.Project.Dependencies.Dependencies[2].ArtifactId, "camel-netty4")
	assert.Equal(t, prj.Project.Dependencies.Dependencies[3].ArtifactId, "camel-servicenow")
	assert.Equal(t, prj.Project.Dependencies.Dependencies[3].Version, "2.21.1")
	assert.Equal(t, prj.Project.Dependencies.Dependencies[4].ArtifactId, "camel-salesforce")
	assert.Equal(t, prj.Project.Dependencies.Dependencies[4].Version, "")
}

func TestProjectGenerationWithFailure(t *testing.T) {
	source := api.BuildSource{
		Identifier: api.BuildIdentifier{
			Name:      "my-integration",
			Qualifier: "",
		},
		Code: api.Code{
			Name:     "my-code.js",
			Content:  `from("timer:start").to("log:end")`,
			Language: "",
		},
		Dependencies: []string{
			"other:mail",
			"camel:camel-netty4",
			"mvn:org.apache.camel/camel-servicenow/2.21.1",
			"mvn:org.apache.camel/camel-salesforce",
		},
	}

	_, err := generateProjectDefinition(source)
	assert.NotNil(t, err)
}
