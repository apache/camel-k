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
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetKitDependenciesDirectories(t *testing.T) {
	kit := &IntegrationKit{
		Status: IntegrationKitStatus{
			Artifacts: []Artifact{
				{Target: "my-dir1/lib/mytest.jar"},
				{Target: "my-dir1/lib/mytest2.jar"},
				{Target: "my-dir1/lib/mytest3.jar"},
				{Target: "my-dir2/lib/mytest4.jar"},
				{Target: "my-dir1/lib2/mytest5.jar"},
				{Target: "my-dir/mytest6.jar"},
				{Target: "my-dir/mytest7.jar"},
			},
			Phase: IntegrationKitPhaseReady,
		},
	}
	paths := kit.Status.GetDependenciesPaths()
	pathsArray := paths.List()
	sort.Strings(pathsArray)
	assert.Len(t, pathsArray, 4)
	assert.Equal(t, "my-dir/*", pathsArray[0])
	assert.Equal(t, "my-dir1/lib/*", pathsArray[1])
	assert.Equal(t, "my-dir1/lib2/*", pathsArray[2])
	assert.Equal(t, "my-dir2/lib/*", pathsArray[3])
}
