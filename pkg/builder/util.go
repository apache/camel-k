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

package builder

import (
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func artifactIDs(artifacts []v1.Artifact) []string {
	result := make([]string, 0, len(artifacts))

	for _, a := range artifacts {
		result = append(result, a.ID)
	}

	return result
}

// initializeStatusFrom helps creating a BuildStatus from scratch filling with base and root images.
func initializeStatusFrom(buildStatus v1.BuildStatus, taskBaseImage string) *v1.BuildStatus {
	status := v1.BuildStatus{}
	baseImage := buildStatus.BaseImage
	if baseImage == "" {
		baseImage = taskBaseImage
	}
	status.BaseImage = baseImage
	rootImage := buildStatus.RootImage
	if rootImage == "" {
		rootImage = taskBaseImage
	}
	status.RootImage = rootImage

	return &status
}
