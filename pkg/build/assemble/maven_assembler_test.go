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

package assemble

import (
	"testing"

	"github.com/apache/camel-k/pkg/build"
	"github.com/stretchr/testify/assert"
)

func TestProjectGeneration(t *testing.T) {
	request := build.Request{
		Identifier: build.Identifier{
			Name:      "my-integration",
			Qualifier: "",
		},
		Code: build.Source{
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

	prj, err := generateProject(&request)

	assert.Nil(t, err)
	assert.NotNil(t, prj)
	assert.Equal(t, len(prj.Dependencies.Dependencies), 5)
	assert.Equal(t, prj.Dependencies.Dependencies[0].ArtifactID, "camel-k-runtime-jvm")
	assert.Equal(t, prj.Dependencies.Dependencies[1].ArtifactID, "camel-mail")
	assert.Equal(t, prj.Dependencies.Dependencies[2].ArtifactID, "camel-netty4")
	assert.Equal(t, prj.Dependencies.Dependencies[3].ArtifactID, "camel-servicenow")
	assert.Equal(t, prj.Dependencies.Dependencies[3].Version, "2.21.1")
	assert.Equal(t, prj.Dependencies.Dependencies[4].ArtifactID, "camel-salesforce")
	assert.Equal(t, prj.Dependencies.Dependencies[4].Version, "")
}

func TestProjectGenerationWithFailure(t *testing.T) {
	request := build.Request{
		Identifier: build.Identifier{
			Name:      "my-integration",
			Qualifier: "",
		},
		Code: build.Source{
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

	_, err := generateProject(&request)

	assert.NotNil(t, err)
}
