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
	"strings"
)

// AddDependency a dependency to maven's dependencies
func (p *Project) AddDependency(dep Dependency) {
	for _, d := range p.Dependencies {
		// Check if the given dependency is already included in the dependency list
		if d == dep {
			return
		}
	}

	p.Dependencies = append(p.Dependencies, dep)
}

// AddDependencyGAV a dependency to maven's dependencies
func (p *Project) AddDependencyGAV(groupID string, artifactID string, version string) {
	p.AddDependency(NewDependency(groupID, artifactID, version))
}

// AddEncodedDependencyGAV a dependency to maven's dependencies
func (p *Project) AddEncodedDependencyGAV(gav string) {
	if d, err := ParseGAV(gav); err == nil {
		// TODO: error handling
		p.AddDependency(d)
	}
}

// NewDependency create an new dependency from the given gav info
func NewDependency(groupID string, artifactID string, version string) Dependency {
	return Dependency{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Version:    version,
		Type:       "jar",
		Classifier: "",
	}
}

//
// NewRepository parse the given repo url ang generated the related struct.
//
// The repository can be customized by appending @instruction to the repository
// uri, as example:
//
//     http://my-nexus:8081/repository/publicc@id=my-repo@snapshots
//
// Will enable snapshots and sets the repo it to my-repo
//
func NewRepository(repo string) Repository {
	r := Repository{
		URL: repo,
		Releases: RepositoryPolicy{
			Enabled: true,
		},
		Snapshots: RepositoryPolicy{
			Enabled: false,
		},
	}

	if idx := strings.Index(repo, "@"); idx != -1 {
		r.URL = repo[:idx]

		for _, attribute := range strings.Split(repo[idx+1:], "@") {
			switch {
			case attribute == "snapshots":
				r.Snapshots.Enabled = true
			case attribute == "noreleases":
				r.Releases.Enabled = false
			case strings.HasPrefix(attribute, "id="):
				r.ID = attribute[3:]
			}
		}
	}

	return r
}
