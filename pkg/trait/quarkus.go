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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/defaults"
)

type quarkusPackageType string

const (
	quarkusTraitId = "quarkus"

	fastJarPackageType quarkusPackageType = "fast-jar"
	nativePackageType  quarkusPackageType = "native"
)

// The Quarkus trait configures the Quarkus runtime.
//
// It's enabled by default.
//
// +camel-k:trait=quarkus
type quarkusTrait struct {
	BaseTrait `property:",squash"`
	// The Quarkus package type, either `fast-jar` or `native` (default `fast-jar`)
	PackageType *quarkusPackageType `property:"package-type" json:"packageType,omitempty"`
}

func newQuarkusTrait() Trait {
	return &quarkusTrait{
		BaseTrait: NewBaseTrait(quarkusTraitId, 1700),
	}
}

func (t *quarkusTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		return false, nil
	}

	if t.PackageType == nil {
		packageType := fastJarPackageType
		t.PackageType = &packageType
	}

	return e.IntegrationKitInPhase(v1.IntegrationKitPhaseNone, v1.IntegrationKitPhaseBuildSubmitted) ||
			e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseDeploying) ||
			e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseRunning),
		nil
}

func (t *quarkusTrait) Apply(e *Environment) error {
	switch e.IntegrationKit.Status.Phase {

	case v1.IntegrationKitPhaseNone:
		if e.IntegrationKit.Labels == nil {
			e.IntegrationKit.Labels = make(map[string]string)
		}
		e.IntegrationKit.Labels[v1.IntegrationKitLayoutLabel] = string(*t.PackageType)

	case v1.IntegrationKitPhaseBuildSubmitted:
		build := getBuilderTask(e.BuildTasks)
		if build == nil {
			return fmt.Errorf("unable to find builder task: %s", e.Integration.Name)
		}

		if build.Maven.Properties == nil {
			build.Maven.Properties = make(map[string]string)
		}

		steps, err := builder.StepsFrom(build.Steps...)
		if err != nil {
			return err
		}

		steps = append(steps, builder.Quarkus.CommonSteps...)

		if t.isNativePackageType() {
			build.Maven.Properties["quarkus.package.type"] = string(nativePackageType)
			steps = append(steps, builder.Image.NativeImageContext)
			// Spectrum does not rely on Dockerfile to assemble the image
			if e.Platform.Status.Build.PublishStrategy != v1.IntegrationPlatformBuildPublishStrategySpectrum {
				steps = append(steps, builder.Image.ExecutableDockerfile)
			}
		} else {
			build.Maven.Properties["quarkus.package.type"] = string(fastJarPackageType)
			steps = append(steps, builder.Quarkus.ComputeQuarkusDependencies, builder.Image.IncrementalImageContext)
			// Spectrum does not rely on Dockerfile to assemble the image
			if e.Platform.Status.Build.PublishStrategy != v1.IntegrationPlatformBuildPublishStrategySpectrum {
				steps = append(steps, builder.Image.JvmDockerfile)
			}
		}

		// Sort steps by phase
		sort.SliceStable(steps, func(i, j int) bool {
			return steps[i].Phase() < steps[j].Phase()
		})

		build.Steps = builder.StepIDsFor(steps...)

	case v1.IntegrationKitPhaseReady:
		if e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) && t.isNativePackageType() {
			container := e.getIntegrationContainer()
			if container == nil {
				return fmt.Errorf("unable to find integration container: %s", e.Integration.Name)
			}

			container.Command = []string{"./camel-k-integration-" + defaults.Version + "-runner"}
			container.WorkingDir = builder.DeploymentDir
		}
	}

	return nil
}

// IsPlatformTrait overrides base class method
func (t *quarkusTrait) IsPlatformTrait() bool {
	return true
}

// InfluencesKit overrides base class method
func (t *quarkusTrait) InfluencesKit() bool {
	return true
}

func (t *quarkusTrait) isNativePackageType() bool {
	return t.PackageType != nil && *t.PackageType == nativePackageType
}

func getBuilderTask(tasks []v1.Task) *v1.BuilderTask {
	for i, task := range tasks {
		if task.Builder != nil {
			return tasks[i].Builder
		}
	}
	return nil
}
