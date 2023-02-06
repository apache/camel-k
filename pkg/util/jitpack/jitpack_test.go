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

package jitpack

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/camel-k/pkg/util/maven"
)

func TestConversion(t *testing.T) {
	vals := []struct {
		prefixID  string
		prefixGav string
	}{
		{"github", "com.github"},
		{"gitlab", "com.gitlab"},
		{"bitbucket", "org.bitbucket"},
		{"gitee", "com.gitee"},
		{"azure", "com.azure"},
	}

	for _, tt := range vals {
		val := tt
		t.Run(val.prefixID, func(t *testing.T) {
			var d *maven.Dependency

			d = ToDependency(val.prefixID + ":u")
			assert.Nil(t, d)

			d = ToDependency(val.prefixID + ":u/r/v")
			assert.NotNil(t, d)
			assert.Equal(t, val.prefixGav+".u", d.GroupID)
			assert.Equal(t, "r", d.ArtifactID)
			assert.Equal(t, "v", d.Version)

			d = ToDependency(val.prefixID + ":u/r")
			assert.NotNil(t, d)
			assert.Equal(t, val.prefixGav+".u", d.GroupID)
			assert.Equal(t, "r", d.ArtifactID)
			assert.Equal(t, DefaultVersion, d.Version)
		})
	}
}
