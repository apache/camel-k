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
	"fmt"
	"sort"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/builder"
)

const (
	gitTraitID    = "git"
	gitTraitOrder = 1700
)

type gitTrait struct {
	BaseTrait
}

func newGitTrait() Trait {
	return &gitTrait{
		BaseTrait: NewBaseTrait(gitTraitID, gitTraitOrder),
	}
}

func (t *gitTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	return e.IntegrationInPhase(v1.IntegrationPhaseBuildSubmitted), nil, nil
}

func (t *gitTrait) Apply(e *Environment) error {
	buildTask := getBuilderTask(e.Pipeline)
	if buildTask == nil {
		return fmt.Errorf("unable to find builder task: %s", e.Integration.Name)
	}
	packageTask := getPackageTask(e.Pipeline)
	if packageTask == nil {
		return fmt.Errorf("unable to find package task: %s", e.Integration.Name)
	}

	buildSteps, err := builder.StepsFrom(buildTask.Steps...)
	if err != nil {
		return err
	}
	buildSteps = append(buildSteps, builder.Git.CommonSteps...)
	packageSteps, err := builder.StepsFrom(packageTask.Steps...)
	if err != nil {
		return err
	}
	packageSteps = append(packageSteps, builder.Git.ComputeDependencies)
	packageSteps = append(packageSteps, builder.Image.StandardImageContext)
	// Create the dockerfile, regardless it's later used or not by the publish strategy
	packageSteps = append(packageSteps, builder.Image.JvmDockerfile)

	// Sort steps by phase
	sort.SliceStable(buildSteps, func(i, j int) bool {
		return buildSteps[i].Phase() < buildSteps[j].Phase()
	})
	sort.SliceStable(packageSteps, func(i, j int) bool {
		return packageSteps[i].Phase() < packageSteps[j].Phase()
	})

	buildTask.Steps = builder.StepIDsFor(buildSteps...)
	packageTask.Steps = builder.StepIDsFor(packageSteps...)

	return nil
}
