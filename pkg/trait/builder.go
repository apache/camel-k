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
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/builder"
	"github.com/apache/camel-k/v2/pkg/util/jib"
	mvn "github.com/apache/camel-k/v2/pkg/util/maven"
	"github.com/apache/camel-k/v2/pkg/util/property"
)

var commandsRegexp = regexp.MustCompile(`"[^"]+"|[\w/-]+`)

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

// InfluencesBuild overrides base class method.
func (t *builderTrait) InfluencesBuild(this, prev map[string]interface{}) bool {
	return true
}

func (t *builderTrait) Configure(e *Environment) (bool, error) {
	if e.IntegrationKit == nil || !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	if trait := e.Catalog.GetTrait(quarkusTraitID); trait != nil {
		quarkus, ok := trait.(*quarkusTrait)
		isNativeIntegration := quarkus.isNativeIntegration(e)
		isNativeKit, err := quarkus.isNativeKit(e)
		if err != nil {
			return false, err
		}
		if ok && pointer.BoolDeref(quarkus.Enabled, true) && (isNativeIntegration || isNativeKit) {
			nativeArgsCd := filepath.Join("maven", "target", "native-sources")
			command := "cd " + nativeArgsCd + " && echo NativeImage version is $(native-image --version) && echo GraalVM expected version is $(cat graalvm.version) && echo WARN: Make sure they are compatible, otherwise the native compilation may results in error && native-image $(cat native-image.args)"
			// it should be performed as the last custom task
			t.Tasks = append(t.Tasks, fmt.Sprintf(`quarkus-native;%s;/bin/bash -c "%s"`, e.CamelCatalog.GetQuarkusToolingImage(), command))
		}
	}

	return e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted), nil
}

func (t *builderTrait) Apply(e *Environment) error {
	// local pipeline tasks
	var pipelineTasks []v1.Task

	// Building task
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
	pipelineTasks = append(pipelineTasks, v1.Task{Builder: builderTask})

	// Custom tasks
	if t.Tasks != nil {
		realBuildStrategy := builderTask.Configuration.Strategy
		if realBuildStrategy == "" {
			realBuildStrategy = e.Platform.Status.Build.BuildConfiguration.Strategy
		}
		if len(t.Tasks) > 0 && realBuildStrategy != v1.BuildStrategyPod {
			e.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseError
			e.IntegrationKit.Status.SetCondition("IntegrationKitTasksValid",
				corev1.ConditionFalse,
				"IntegrationKitTasksValid",
				fmt.Sprintf("Pipeline tasks unavailable when using `%s` platform build strategy: use `%s` instead.",
					realBuildStrategy,
					v1.BuildStrategyPod),
			)
			if err := e.Client.Status().Update(e.Ctx, e.IntegrationKit); err != nil {
				return err
			}
			return nil
		}

		customTasks, err := t.customTasks()
		if err != nil {
			return err
		}

		pipelineTasks = append(pipelineTasks, customTasks...)
	}

	// Packaging task
	// It's the same builder configuration, but with different steps
	packageTask := builderTask.DeepCopy()
	packageTask.Name = "package"
	packageTask.Steps = make([]string, 0)
	pipelineTasks = append(pipelineTasks, v1.Task{Package: packageTask})

	// Publishing task
	switch e.Platform.Status.Build.PublishStrategy {
	case v1.IntegrationPlatformBuildPublishStrategySpectrum:
		pipelineTasks = append(pipelineTasks, v1.Task{Spectrum: &v1.SpectrumTask{
			BaseTask: v1.BaseTask{
				Name: "spectrum",
			},
			PublishTask: v1.PublishTask{
				BaseImage: e.Platform.Status.Build.BaseImage,
				Image:     getImageName(e),
				Registry:  e.Platform.Status.Build.Registry,
			},
		}})

	case v1.IntegrationPlatformBuildPublishStrategyJib:
		pipelineTasks = append(pipelineTasks, v1.Task{Jib: &v1.JibTask{
			BaseTask: v1.BaseTask{
				Name: "jib",
			},
			PublishTask: v1.PublishTask{
				BaseImage: e.Platform.Status.Build.BaseImage,
				Image:     getImageName(e),
				Registry:  e.Platform.Status.Build.Registry,
			},
		}})

	case v1.IntegrationPlatformBuildPublishStrategyS2I:
		pipelineTasks = append(pipelineTasks, v1.Task{S2i: &v1.S2iTask{
			BaseTask: v1.BaseTask{
				Name: "s2i",
			},
			Tag: e.IntegrationKit.ResourceVersion,
		}})

	case v1.IntegrationPlatformBuildPublishStrategyBuildah:
		t.L.Infof("Warning: Buildah publishing strategy is deprecated and may be removed in future releases. Use any alternative publishing strategy.")
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
		pipelineTasks = append(pipelineTasks, v1.Task{Buildah: &v1.BuildahTask{
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
		t.L.Infof("Warning: Kaniko publishing strategy is deprecated and may be removed in future releases. Use any alternative publishing strategy.")
		persistentVolumeClaim := e.Platform.Status.Build.PublishStrategyOptions[builder.KanikoPVCName]
		cacheEnabled := e.Platform.Status.Build.IsOptionEnabled(builder.KanikoBuildCacheEnabled)

		var executorImage string
		if image, found := e.Platform.Status.Build.PublishStrategyOptions[builder.KanikoExecutorImage]; found {
			executorImage = image
			t.L.Infof("User defined executor image %s will be used for kaniko", image)
		}

		pipelineTasks = append(pipelineTasks, v1.Task{Kaniko: &v1.KanikoTask{
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

	// add local pipeline tasks to env pipeline
	e.Pipeline = append(e.Pipeline, pipelineTasks...)
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

	if trait := e.Catalog.GetTrait(quarkusTraitID); trait != nil {
		quarkus, ok := trait.(*quarkusTrait)
		isNativeIntegration := quarkus.isNativeIntegration(e)
		isNativeKit, err := quarkus.isNativeKit(e)
		if err != nil {
			return nil, err
		}
		// The builder trait must define certain resources requirements when we have a native build
		if ok && pointer.BoolDeref(quarkus.Enabled, true) && (isNativeIntegration || isNativeKit) {
			// Force the build to run in a separate Pod and strictly sequential
			t.L.Info("This is a Quarkus native build: setting build configuration with build Pod strategy, 1 CPU core and 4 GiB memory. Make sure your cluster can handle it.")
			t.Strategy = string(v1.BuildStrategyPod)
			t.OrderStrategy = string(v1.BuildOrderStrategySequential)
			t.RequestCPU = "1000m"
			t.RequestMemory = "4Gi"
		}
	}

	buildConfig := v1.BuildConfiguration{
		RequestCPU:    t.RequestCPU,
		RequestMemory: t.RequestMemory,
		LimitCPU:      t.LimitCPU,
		LimitMemory:   t.LimitMemory,
	}

	if t.Strategy != "" {
		t.L.Infof("User defined build strategy %s", t.Strategy)
		found := false
		for _, s := range v1.BuildStrategies {
			if string(s) == t.Strategy {
				found = true
				buildConfig.Strategy = s
				break
			}
		}
		if !found {
			var strategies []string
			for _, s := range v1.BuildStrategies {
				strategies = append(strategies, string(s))
			}
			return nil, fmt.Errorf("unknown build strategy: %s. One of [%s] is expected", t.Strategy, strings.Join(strategies, ", "))
		}
	}

	if t.OrderStrategy != "" {
		t.L.Infof("User defined build order strategy %s", t.OrderStrategy)
		found := false
		for _, s := range v1.BuildOrderStrategies {
			if string(s) == t.OrderStrategy {
				found = true
				buildConfig.OrderStrategy = s
				break
			}
		}
		if !found {
			var strategies []string
			for _, s := range v1.BuildOrderStrategies {
				strategies = append(strategies, string(s))
			}
			return nil, fmt.Errorf("unknown build order strategy: %s. One of [%s] is expected", t.OrderStrategy, strings.Join(strategies, ", "))
		}
	}

	task := &v1.BuilderTask{
		BaseTask: v1.BaseTask{
			Name: "builder",
		},
		Configuration: buildConfig,
		BaseImage:     e.Platform.Status.Build.BaseImage,
		Runtime:       e.CamelCatalog.Runtime,
		Dependencies:  e.IntegrationKit.Spec.Dependencies,
		Maven:         maven,
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

	if e.Platform.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategyJib {
		if err := jib.CreateProfileConfigmap(e.Ctx, e.Client, e.IntegrationKit); err != nil {
			return nil, fmt.Errorf("could not create default maven jib profile: %w. ", err)
		}
		t.MavenProfiles = append(t.MavenProfiles, "configmap:"+e.IntegrationKit.Name+"-publish-jib-profile/profile.xml")
	}

	// User provides a maven profile
	if t.MavenProfiles != nil {
		mavenProfiles := make([]v1.ValueSource, 0)
		for _, v := range t.MavenProfiles {
			if v != "" {
				mavenProfile, err := v1.DecodeValueSource(v, "profile.xml",
					"illegal profile definition, syntax: configmap|secret:resource-name[/profile path]")
				if err != nil {
					return nil, fmt.Errorf("invalid maven profile: %s: %w. ", v, err)
				}
				mavenProfiles = append(mavenProfiles, mavenProfile)
			}
		}
		task.Maven.Profiles = mavenProfiles
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

func (t *builderTrait) customTasks() ([]v1.Task, error) {
	customTasks := make([]v1.Task, len(t.Tasks), len(t.Tasks))
	for i, t := range t.Tasks {
		splitted := strings.Split(t, ";")
		if len(splitted) < 3 {
			return nil, fmt.Errorf(`You need to provide a custom task with at least 3 arguments, ie "my-task-name;my-image;echo 'hello', was %v"`, t)
		}
		var containerCommand string
		if len(splitted) > 3 {
			//recompose in case of usage of separator char in the script
			containerCommand = strings.Join(splitted[2:], ";")
		} else {
			containerCommand = splitted[2]
		}
		containerCommands := splitContainerCommand(containerCommand)
		customTasks[i] = v1.Task{
			Custom: &v1.UserTask{
				BaseTask: v1.BaseTask{
					Name: splitted[0],
				},
				ContainerImage:    splitted[1],
				ContainerCommands: containerCommands,
			},
		}
	}
	return customTasks, nil
}

// we may get a command in the following format `/bin/bash -c "ls && echo 'hello'`
// which should provide a string with {"/bin/bash", "-c", "ls && echo 'hello'"}
func splitContainerCommand(command string) []string {
	matches := commandsRegexp.FindAllString(command, -1)
	removeQuotes := make([]string, 0, len(matches))
	for _, m := range matches {
		removeQuotes = append(removeQuotes, strings.Replace(m, "\"", "", -1))
	}

	return removeQuotes
}
