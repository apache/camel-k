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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/builder/kaniko"
	"github.com/apache/camel-k/pkg/builder/s2i"
	"github.com/apache/camel-k/pkg/platform"
)

// TODO: we should add a way to label a trait as platform so it cannot be disabled/removed
type builderTrait struct {
	BaseTrait `property:",squash"`
}

func newBuilderTrait() *builderTrait {
	return &builderTrait{
		BaseTrait: BaseTrait{
			id: ID("builder"),
		},
	}
}

func (t *builderTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if e.IntegrationContextInPhase(v1alpha1.IntegrationContextPhaseBuilding) {
		return true, nil
	}

	if e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseBuildingImage) {
		return true, nil
	}

	return false, nil
}

func (t *builderTrait) Apply(e *Environment) error {
	if e.IntegrationContextInPhase(v1alpha1.IntegrationContextPhaseBuilding) {
		if platform.SupportsS2iPublishStrategy(e.Platform) {
			e.Steps = s2i.DefaultSteps
		} else if platform.SupportsKanikoPublishStrategy(e.Platform) {
			e.Steps = kaniko.DefaultSteps
			e.BuildDir = kaniko.BuildDir
		}
	}

	if e.InPhase(v1alpha1.IntegrationContextPhaseReady, v1alpha1.IntegrationPhaseBuildingImage) {
		if platform.SupportsS2iPublishStrategy(e.Platform) {
			e.Steps = []builder.Step{
				builder.NewStep("packager", builder.ApplicationPackagePhase, builder.StandardPackager),
				builder.NewStep("publisher/s2i", builder.ApplicationPublishPhase, s2i.Publisher),
				builder.NewStep("notify/integration", builder.NotifyPhase, builder.NotifyIntegration),
			}
		} else if platform.SupportsKanikoPublishStrategy(e.Platform) {
			e.Steps = []builder.Step{
				builder.NewStep("packager", builder.ApplicationPackagePhase, builder.StandardPackager),
				builder.NewStep("publisher/kaniko", builder.ApplicationPublishPhase, kaniko.Publisher),
				builder.NewStep("notify/integration", builder.NotifyPhase, builder.NotifyIntegration),
			}
			e.BuildDir = kaniko.BuildDir
		}
	}

	return nil
}
