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
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The builder trait is internally used to determine the best strategy to
// build and configure IntegrationKits.
//
// +camel-k:trait=builder
type builderTrait struct {
	BaseTrait `property:",squash"`
	// Enable verbose logging on build components that support it (e.g. Kaniko build pod).
	Verbose *bool `property:"verbose" json:"verbose,omitempty"`
	// A list of properties to be provided to the build task
	Properties []string `property:"properties" json:"properties,omitempty"`
}

func newBuilderTrait() Trait {
	return &builderTrait{
		BaseTrait: NewBaseTrait("builder", 600),
	}
}

// IsPlatformTrait overrides base class method
func (t *builderTrait) IsPlatformTrait() bool {
	return true
}

// InfluencesKit overrides base class method
func (t *builderTrait) InfluencesKit() bool {
	return true
}

func (t *builderTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted), nil
}

func (t *builderTrait) Apply(e *Environment) error {
	builderTask, err := t.builderTask(e)
	if err != nil {
		e.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseError
		e.IntegrationKit.Status.SetCondition("IntegrationKitPropertiesFormatValid", corev1.ConditionFalse,
			"IntegrationKitPropertiesFormatValid", fmt.Sprintf("One or more properties where not formatted as expected: %s", err.Error()))
		if _, err := e.Client.CamelV1().IntegrationKits(e.IntegrationKit.Namespace).UpdateStatus(e.C, e.IntegrationKit, metav1.UpdateOptions{}); err != nil {
			return err
		}

		return nil
	}

	e.BuildTasks = append(e.BuildTasks, v1.Task{Builder: builderTask})

	switch e.Platform.Status.Build.PublishStrategy {
	case v1.IntegrationPlatformBuildPublishStrategySpectrum:
		e.BuildTasks = append(e.BuildTasks, v1.Task{Spectrum: &v1.SpectrumTask{
			BaseTask: v1.BaseTask{
				Name: "spectrum",
			},
			PublishTask: v1.PublishTask{
				BaseImage: e.Platform.Status.Build.BaseImage,
				Image:     getImageName(e),
				Registry:  e.Platform.Status.Build.Registry,
			},
		}})

	case v1.IntegrationPlatformBuildPublishStrategyS2I:
		e.BuildTasks = append(e.BuildTasks, v1.Task{S2i: &v1.S2iTask{
			BaseTask: v1.BaseTask{
				Name: "s2i",
			},
			Tag: e.IntegrationKit.ResourceVersion,
		}})

	case v1.IntegrationPlatformBuildPublishStrategyBuildah:
		e.BuildTasks = append(e.BuildTasks, v1.Task{Buildah: &v1.BuildahTask{
			BaseTask: v1.BaseTask{
				Name: "buildah",
			},
			PublishTask: v1.PublishTask{
				Image:    getImageName(e),
				Registry: e.Platform.Status.Build.Registry,
			},
			HttpProxySecret: e.Platform.Status.Build.HTTPProxySecret,
			Verbose:         t.Verbose,
		}})

	case v1.IntegrationPlatformBuildPublishStrategyKaniko:
		e.BuildTasks = append(e.BuildTasks, v1.Task{Kaniko: &v1.KanikoTask{
			BaseTask: v1.BaseTask{
				Name: "kaniko",
			},
			PublishTask: v1.PublishTask{
				Image:    getImageName(e),
				Registry: e.Platform.Status.Build.Registry,
			},
			Cache: v1.KanikoTaskCache{
				Enabled:               e.Platform.Status.Build.KanikoBuildCache,
				PersistentVolumeClaim: e.Platform.Status.Build.PersistentVolumeClaim,
			},
			HttpProxySecret: e.Platform.Status.Build.HTTPProxySecret,
			Verbose:         t.Verbose,
		}})
	}

	return nil
}

func (t *builderTrait) builderTask(e *Environment) (*v1.BuilderTask, error) {
	task := &v1.BuilderTask{
		BaseTask: v1.BaseTask{
			Name: "builder",
		},
		Runtime:      e.CamelCatalog.Runtime,
		Dependencies: e.IntegrationKit.Spec.Dependencies,
		Properties:   e.Platform.Status.Build.Properties,
		Maven:        e.Platform.Status.Build.Maven,
	}

	// initialize properties if nil
	if task.Properties == nil {
		task.Properties = make(map[string]string)
	}
	// User provided build time configuration properties
	if t.Properties != nil {
		for _, v := range t.Properties {
			split := strings.Split(v, "=")
			if len(split) != 2 {
				return nil, fmt.Errorf("Build time configuration property must have key=value format, it was %v", v)
			}

			task.Properties[split[0]] = split[1]
		}
	}

	steps := make([]builder.Step, 0)
	steps = append(steps, builder.DefaultSteps...)

	quarkus := e.Catalog.GetTrait("quarkus").(*quarkusTrait)
	quarkus.addBuildSteps(&steps)

	// sort steps by phase
	sort.SliceStable(steps, func(i, j int) bool {
		return steps[i].Phase() < steps[j].Phase()
	})

	task.Steps = builder.StepIDsFor(steps...)

	return task, nil
}

func getImageName(e *Environment) string {
	organization := e.Platform.Status.Build.Registry.Organization
	if organization == "" {
		organization = e.Platform.Namespace
	}
	return e.Platform.Status.Build.Registry.Address + "/" + organization + "/camel-k-" + e.IntegrationKit.Name + ":" + e.IntegrationKit.ResourceVersion
}
