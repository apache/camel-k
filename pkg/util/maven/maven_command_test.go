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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMavenContext(t *testing.T) {
	mvnSimpleCompile := getMavenContext([]string{"compile", "-s", "my-settings.xml"}, "")
	mvnOptionsCompile := getMavenContext([]string{"compile", "-s", "my-settings.xml"}, "-DmyProperty=hello")
	mvnSimplePackage := getMavenContext([]string{"package", "-s", "my-settings.xml"}, "")
	mvnOptionsPackage := getMavenContext([]string{"package", "-s", "my-settings.xml"}, "-DmyProperty=hello")

	assert.Equal(t, "compile -s my-settings.xml", mvnSimpleCompile)
	assert.Equal(t, "compile -s my-settings.xml -DmyProperty=hello", mvnOptionsCompile)
	assert.Equal(t, "-s my-settings.xml", mvnSimplePackage)
	assert.Equal(t, "-s my-settings.xml -DmyProperty=hello", mvnOptionsPackage)
}

func TestCloneMavenQuarkus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-test-camel-k-resources")
	assert.Nil(t, err)
	mvnCommand := Command{
		context: Context{
			Path: tmpDir,
		},
	}
	err = mvnCommand.cloneCamelQuarkusArchetype()
	assert.Nil(t, err)
}
