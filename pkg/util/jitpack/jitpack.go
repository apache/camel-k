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
	"strings"

	"github.com/apache/camel-k/pkg/util/maven"
)

const RepoURL = "https://jitpack.io"
const LatestVersion = "master-SNAPSHOT"

func ToDependency(dependencyID string) *maven.Dependency {
	gav := ""

	switch {
	case strings.HasPrefix(dependencyID, "github:"):
		gav = strings.TrimPrefix(dependencyID, "github:")
		gav = "com.github." + gav
	case strings.HasPrefix(dependencyID, "gitlab:"):
		gav = strings.TrimPrefix(dependencyID, "gitlab:")
		gav = "com.gitlab." + gav
	case strings.HasPrefix(dependencyID, "bitbucket:"):
		gav = strings.TrimPrefix(dependencyID, "bitbucket:")
		gav = "org.bitbucket." + gav
	case strings.HasPrefix(dependencyID, "gitee:"):
		gav = strings.TrimPrefix(dependencyID, "gitee:")
		gav = "com.gitee." + gav
	case strings.HasPrefix(dependencyID, "azure:"):
		gav = strings.TrimPrefix(dependencyID, "azure:")
		gav = "com.azure." + gav
	}

	if gav == "" {
		return nil
	}

	gav = strings.Replace(gav, "/", ":", -1)
	dep, err := maven.ParseGAV(gav)
	if err != nil {
		return nil
	}

	// if no version is set, then use master-SNAPSHOT which
	// targets the latest snapshot from the master branch
	if dep.Version == "" {
		dep.Version = LatestVersion
	}

	return &dep
}
