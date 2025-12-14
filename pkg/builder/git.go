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
	"path/filepath"

	util "github.com/apache/camel-k/v2/pkg/util/gitops"
)

func init() {
	registerSteps(Git)

	Git.CommonSteps = []Step{
		Git.CloneProject,
		Git.InjectJibProfile,
		Git.BuildMavenContext,
		Git.ExecuteMavenContext,
		Git.ComputeDependencies,
	}
}

type gitSteps struct {
	CloneProject        Step
	InjectJibProfile    Step
	BuildMavenContext   Step
	ExecuteMavenContext Step
	ComputeDependencies Step

	CommonSteps []Step
}

//nolint:mnd
var Git = gitSteps{
	CloneProject:        NewStep(ProjectGenerationPhase, cloneProject),
	InjectJibProfile:    NewStep(ProjectGenerationPhase+1, injectJibProfile),
	BuildMavenContext:   NewStep(ProjectGenerationPhase+2, buildMavenContextSettings),
	ExecuteMavenContext: NewStep(ProjectGenerationPhase+3, executeMavenPackageCommand),
	ComputeDependencies: NewStep(ProjectBuildPhase+1, computeFatJarDependency),
}

func cloneProject(ctx *builderContext) error {
	secretToken := ""
	if ctx.Build.Git.Secret != "" {
		var err error
		secretToken, err = util.GitToken(ctx.C, ctx.Client, ctx.Namespace, ctx.Build.Git.Secret)
		if err != nil {
			return err
		}
	}
	_, err := util.CloneGitProject(*ctx.Build.Git, filepath.Join(ctx.Path, "maven"), secretToken)

	return err
}
