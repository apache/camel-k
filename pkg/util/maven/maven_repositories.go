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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

var DefaultRepositories = &defaultRepositories{}

type defaultRepositories struct{}

func (o defaultRepositories) apply(settings *Settings) error {
	for _, repository := range defaultMavenRepositories() {
		upsertRepository(repository, &settings.Profiles[0].Repositories)
		upsertRepository(repository, &settings.Profiles[0].PluginRepositories)
	}
	return nil
}

func defaultMavenRepositories() (repositories []v1.Repository) {
	for _, repository := range strings.Split(DefaultMavenRepositories, ",") {
		repositories = append(repositories, NewRepository(repository))
	}
	return
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
