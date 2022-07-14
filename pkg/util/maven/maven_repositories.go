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
	"fmt"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

var DefaultRepositories = defaultRepositories{}

type defaultRepositories struct{}

func (o defaultRepositories) apply(settings *Settings) error {
	for _, repository := range defaultMavenRepositories() {
		upsertRepository(repository, &settings.Profiles[0].Repositories)
		upsertRepository(repository, &settings.Profiles[0].PluginRepositories)
	}
	return nil
}

func defaultMavenRepositories() []v1.Repository {
	var repositories []v1.Repository
	for _, repository := range strings.Split(DefaultMavenRepositories, ",") {
		repositories = append(repositories, NewRepository(repository))
	}
	return repositories
}

func Repositories(repositories ...string) SettingsOption {
	return extraRepositories{
		repositories: repositories,
	}
}

type extraRepositories struct {
	repositories []string
}

func (o extraRepositories) apply(settings *Settings) error {
	for i, r := range o.repositories {
		if strings.Contains(r, "@mirrorOf=") {
			mirror := NewMirror(r)
			if mirror.ID == "" {
				mirror.ID = fmt.Sprintf("mirror-%03d", i)
			}
			upsertMirror(mirror, &settings.Mirrors)
		} else {
			repository := NewRepository(r)
			if repository.ID == "" {
				repository.ID = fmt.Sprintf("repository-%03d", i)
			}
			upsertRepository(repository, &settings.Profiles[0].Repositories)
			upsertRepository(repository, &settings.Profiles[0].PluginRepositories)
		}
	}
	return nil
}

func upsertRepository(repository v1.Repository, repositories *[]v1.Repository) {
	for i, r := range *repositories {
		if r.ID == repository.ID {
			(*repositories)[i] = repository
			return
		}
	}
	*repositories = append(*repositories, repository)
}

func upsertMirror(mirror Mirror, mirrors *[]Mirror) {
	for i, r := range *mirrors {
		if r.ID == mirror.ID {
			(*mirrors)[i] = mirror
			return
		}
	}
	*mirrors = append(*mirrors, mirror)
}
