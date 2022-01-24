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

package trait

import (
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// The Registry trait sets up Maven to use the Image registry
// as a Maven repository
//
// +camel-k:trait=registry
type registryTrait struct {
	BaseTrait `property:",squash"`
}

func newRegistryTrait() Trait {
	return &registryTrait{
		BaseTrait: NewBaseTrait("registry", 1650),
	}
}

// InfluencesKit overrides base class method
func (t *registryTrait) InfluencesKit() bool {
	return true
}

func (t *registryTrait) Configure(e *Environment) (bool, error) {
	// disabled by default
	if IsNilOrFalse(t.Enabled) {
		return false, nil
	}

	return e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted), nil
}

func (t *registryTrait) Apply(e *Environment) error {
	build := getBuilderTask(e.BuildTasks)
	ext := v1.MavenArtifact{
		GroupID:    "com.github.johnpoth",
		ArtifactID: "wagon-docker-registry",
		Version:    "0.2.0-SNAPSHOT",
	}
	policy := v1.RepositoryPolicy{
		Enabled:        true,
		ChecksumPolicy: "fail",
	}
	// TODO: If we are running on openshift then then fetch the credentials
	// needed to lookup the image registry
	// append namespace to the repository URL
	// current workaround is to setup Maven manually during installation
	if e.Platform.Spec.Cluster != v1.IntegrationPlatformClusterOpenShift {
		repo := v1.Repository{
			ID:        "image-registry",
			URL:       "docker://" + e.Platform.Spec.Build.Registry.Address,
			Snapshots: policy,
			Releases:  policy,
		}
		// configure Maven to lookup dependencies in the Image registry
		build.Maven.Repositories = append(build.Maven.Repositories, repo)
	}

	build.Maven.Extension = append(build.Maven.Extension, ext)
	return nil
}
