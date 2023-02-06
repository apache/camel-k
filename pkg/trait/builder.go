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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/builder"
	mvn "github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/property"
)

type builderTrait struct {
	BaseTrait
	traitv1.BuilderTrait `property:",squash"`
}

func newBuilderTrait() Trait {
	return &builderTrait{
		BaseTrait: NewBaseTrait("builder", 600),
	}
}

// IsPlatformTrait overrides base class method.
func (t *builderTrait) IsPlatformTrait() bool {
	return true
}

// InfluencesKit overrides base class method.
func (t *builderTrait) InfluencesKit() bool {
	return true
}

func (t *builderTrait) Configure(e *Environment) (bool, error) {
	if e.IntegrationKit == nil || !pointer.BoolDeref(t.Enabled, true) {
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
		if err := e.Client.Status().Update(e.Ctx, e.IntegrationKit); err != nil {
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
		var platform string
		var found bool
		if platform, found = e.Platform.Status.Build.PublishStrategyOptions[builder.BuildahPlatform]; !found {
			platform = ""
			t.L.Infof("Attribute platform for buildah not found, default from host will be used!")
		} else {
			t.L.Infof("User defined %s platform, will be used from buildah!", platform)
		}
		var executorImage string
		if image, found := e.Platform.Status.Build.PublishStrategyOptions[builder.BuildahImage]; found {
			executorImage = image
			t.L.Infof("User defined executor image %s will be used for buildah", image)
		}
		e.BuildTasks = append(e.BuildTasks, v1.Task{Buildah: &v1.BuildahTask{
			Platform: platform,
			BaseTask: v1.BaseTask{
				Name: "buildah",
			},
			PublishTask: v1.PublishTask{
				Image:    getImageName(e),
				Registry: e.Platform.Status.Build.Registry,
			},
			Verbose:       t.Verbose,
			ExecutorImage: executorImage,
		}})
	//nolint: staticcheck,nolintlint
	case v1.IntegrationPlatformBuildPublishStrategyKaniko:
		persistentVolumeClaim := e.Platform.Status.Build.PublishStrategyOptions[builder.KanikoPVCName]
		cacheEnabled := e.Platform.Status.Build.IsOptionEnabled(builder.KanikoBuildCacheEnabled)

		var executorImage string
		if image, found := e.Platform.Status.Build.PublishStrategyOptions[builder.KanikoExecutorImage]; found {
			executorImage = image
			t.L.Infof("User defined executor image %s will be used for kaniko", image)
		}

		e.BuildTasks = append(e.BuildTasks, v1.Task{Kaniko: &v1.KanikoTask{
			BaseTask: v1.BaseTask{
				Name: "kaniko",
			},
			PublishTask: v1.PublishTask{
				Image:    getImageName(e),
				Registry: e.Platform.Status.Build.Registry,
			},
			Cache: v1.KanikoTaskCache{
				Enabled:               &cacheEnabled,
				PersistentVolumeClaim: persistentVolumeClaim,
			},
			Verbose:       t.Verbose,
			ExecutorImage: executorImage,
		}})
	}

	return nil
}

func (t *builderTrait) builderTask(e *Environment) (*v1.BuilderTask, error) {
	maven := v1.MavenBuildSpec{
		MavenSpec: e.Platform.Status.Build.Maven,
	}
	// Add Maven repositories defined in the IntegrationKit
	for _, repo := range e.IntegrationKit.Spec.Repositories {
		maven.Repositories = append(maven.Repositories, mvn.NewRepository(repo))
	}

	task := &v1.BuilderTask{
		BaseTask: v1.BaseTask{
			Name: "builder",
		},
		BaseImage:    e.Platform.Status.Build.BaseImage,
		Runtime:      e.CamelCatalog.Runtime,
		Dependencies: e.IntegrationKit.Spec.Dependencies,
		Maven:        maven,
	}

	if task.Maven.Properties == nil {
		task.Maven.Properties = make(map[string]string)
	}
	// User provided Maven properties
	if t.Properties != nil {
		for _, v := range t.Properties {
			key, value := property.SplitPropertyFileEntry(v)
			if len(key) == 0 || len(value) == 0 {
				return nil, fmt.Errorf("maven property must have key=value format, it was %v", v)
			}

			task.Maven.Properties[key] = value
		}
	}

	steps := make([]builder.Step, 0)
	steps = append(steps, builder.Project.CommonSteps...)

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
