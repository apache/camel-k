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
	"regexp"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/builder"
	"github.com/apache/camel-k/v2/pkg/util/jib"
	mvn "github.com/apache/camel-k/v2/pkg/util/maven"
	"github.com/apache/camel-k/v2/pkg/util/property"
)

const (
	builderTraitID = "builder"
)

var commandsRegexp = regexp.MustCompile(`"[^"]+"|[\w/-]+`)

type builderTrait struct {
	BasePlatformTrait
	traitv1.BuilderTrait `property:",squash"`
}

func newBuilderTrait() Trait {
	return &builderTrait{
		BasePlatformTrait: NewBasePlatformTrait("builder", 600),
	}
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
	if e.IntegrationKit == nil {
		return false, nil
	}

	t.adaptDeprecatedFields()

	if e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted) {
		if trait := e.Catalog.GetTrait(quarkusTraitID); trait != nil {
			quarkus, ok := trait.(*quarkusTrait)
			isNativeIntegration := quarkus.isNativeIntegration(e)
			isNativeKit, err := quarkus.isNativeKit(e)
			if err != nil {
				return false, err
			}
			if ok && (isNativeIntegration || isNativeKit) {
				// TODO expect maven repository in local repo (need to change builder pod accordingly!)
				command := builder.QuarkusRuntimeSupport(e.CamelCatalog.GetCamelQuarkusVersion()).BuildCommands()
				nativeBuilderImage := quarkus.NativeBuilderImage
				if nativeBuilderImage == "" {
					// default from the catalog
					nativeBuilderImage = e.CamelCatalog.GetQuarkusToolingImage()
				}
				// it should be performed as the last custom task
				t.Tasks = append(t.Tasks, fmt.Sprintf(`quarkus-native;%s;/bin/bash -c "%s"`, nativeBuilderImage, command))
				// Force the build to run in a separate Pod and strictly sequential
				t.L.Info("This is a Quarkus native build: setting build configuration with build Pod strategy, and native container with 1 CPU core and 4 GiB memory. Make sure your cluster can handle it.")
				t.Strategy = string(v1.BuildStrategyPod)
				t.OrderStrategy = string(v1.BuildOrderStrategySequential)
				t.TasksRequestCPU = append(t.TasksRequestCPU, "quarkus-native:1000m")
				t.TasksRequestMemory = append(t.TasksRequestMemory, "quarkus-native:4Gi")
			}
		}

		return true, nil
	}

	return false, nil
}

func (t *builderTrait) adaptDeprecatedFields() {
	if t.RequestCPU != "" {
		t.L.Info("The request-cpu parameter is deprecated and may be removed in future releases. Make sure to use tasks-request-cpu parameter instead.")
		t.TasksRequestCPU = append(t.TasksRequestCPU, fmt.Sprintf("builder:%s", t.RequestCPU))
	}
	if t.LimitCPU != "" {
		t.L.Info("The limit-cpu parameter is deprecated and may be removed in future releases. Make sure to use tasks-limit-cpu parameter instead.")
		t.TasksLimitCPU = append(t.TasksLimitCPU, fmt.Sprintf("builder:%s", t.LimitCPU))
	}
	if t.RequestMemory != "" {
		t.L.Info("The request-memory parameter is deprecated and may be removed in future releases. Make sure to use tasks-request-memory parameter instead.")
		t.TasksRequestMemory = append(t.TasksRequestMemory, fmt.Sprintf("builder:%s", t.RequestMemory))
	}
	if t.LimitMemory != "" {
		t.L.Info("The limit-memory parameter is deprecated and may be removed in future releases. Make sure to use tasks-limit-memory parameter instead.")
		t.TasksLimitMemory = append(t.TasksLimitMemory, fmt.Sprintf("builder:%s", t.LimitMemory))
	}
}

func (t *builderTrait) Apply(e *Environment) error {
	// local pipeline tasks
	var pipelineTasks []v1.Task

	// task configuration resources
	tasksConf, err := t.parseTasksConf()
	if err != nil {
		return err
	}
	// Building task
	builderTask, err := t.builderTask(e, taskConfOrDefault(tasksConf, "builder"))
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

		customTasks, err := t.customTasks(tasksConf)
		if err != nil {
			return err
		}

		pipelineTasks = append(pipelineTasks, customTasks...)
	}

	// Packaging task
	// It's the same builder configuration, but with different steps and conf
	packageTask := builderTask.DeepCopy()
	packageTask.Name = "package"
	packageTask.Configuration = *taskConfOrDefault(tasksConf, "package")
	packageTask.Steps = make([]string, 0)
	pipelineTasks = append(pipelineTasks, v1.Task{Package: packageTask})

	// Publishing task
	switch e.Platform.Status.Build.PublishStrategy {
	case v1.IntegrationPlatformBuildPublishStrategySpectrum:
		pipelineTasks = append(pipelineTasks, v1.Task{Spectrum: &v1.SpectrumTask{
			BaseTask: v1.BaseTask{
				Name:          "spectrum",
				Configuration: *taskConfOrDefault(tasksConf, "spectrum"),
			},
			PublishTask: v1.PublishTask{
				BaseImage: t.getBaseImage(e),
				Image:     getImageName(e),
				Registry:  e.Platform.Status.Build.Registry,
			},
		}})

	case v1.IntegrationPlatformBuildPublishStrategyJib:
		pipelineTasks = append(pipelineTasks, v1.Task{Jib: &v1.JibTask{
			BaseTask: v1.BaseTask{
				Name:          "jib",
				Configuration: *taskConfOrDefault(tasksConf, "jib"),
			},
			PublishTask: v1.PublishTask{
				BaseImage: t.getBaseImage(e),
				Image:     getImageName(e),
				Registry:  e.Platform.Status.Build.Registry,
			},
		}})

	case v1.IntegrationPlatformBuildPublishStrategyS2I:
		pipelineTasks = append(pipelineTasks, v1.Task{S2i: &v1.S2iTask{
			BaseTask: v1.BaseTask{
				Name:          "s2i",
				Configuration: *taskConfOrDefault(tasksConf, "s2i"),
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
				Name:          "buildah",
				Configuration: *taskConfOrDefault(tasksConf, "buildah"),
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
				Name:          "kaniko",
				Configuration: *taskConfOrDefault(tasksConf, "kaniko"),
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

func (t *builderTrait) builderTask(e *Environment, taskConf *v1.BuildConfiguration) (*v1.BuilderTask, error) {
	maven := v1.MavenBuildSpec{
		MavenSpec: e.Platform.Status.Build.Maven,
	}
	// Add Maven repositories defined in the IntegrationKit
	for _, repo := range e.IntegrationKit.Spec.Repositories {
		maven.Repositories = append(maven.Repositories, mvn.NewRepository(repo))
	}

	if t.Strategy != "" {
		t.L.Infof("User defined build strategy %s", t.Strategy)
		found := false
		for _, s := range v1.BuildStrategies {
			if string(s) == t.Strategy {
				found = true
				taskConf.Strategy = s
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
				taskConf.OrderStrategy = s
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
			Name:          "builder",
			Configuration: *taskConf,
		},
		BaseImage:    t.getBaseImage(e),
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

	if e.Platform.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategyJib {
		profile, err := jib.JibMavenProfile(e.CamelCatalog.GetJibMavenPluginVersion(), e.CamelCatalog.GetJibLayerFilterExtensionMavenVersion())
		if err != nil {
			return nil, fmt.Errorf("error generating default maven jib profile: %w. ", err)
		}
		if err := jib.CreateProfileConfigmap(e.Ctx, e.Client, e.IntegrationKit, profile); err != nil {
			return nil, fmt.Errorf("could not create default maven jib profile configmap: %w. ", err)
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

func (t *builderTrait) getBaseImage(e *Environment) string {
	baseImage := t.BaseImage
	if baseImage == "" {
		baseImage = e.Platform.Status.Build.BaseImage
	}
	return baseImage
}

func (t *builderTrait) customTasks(tasksConf map[string]*v1.BuildConfiguration) ([]v1.Task, error) {
	customTasks := make([]v1.Task, len(t.Tasks))

	for i, t := range t.Tasks {
		splitted := strings.Split(t, ";")
		if len(splitted) < 3 {
			return nil, fmt.Errorf(`provide a custom task with at least 3 arguments, ie "my-task-name;my-image;echo 'hello', was %v"`, t)
		}
		var containerCommand string
		if len(splitted) > 3 {
			// recompose in case of usage of separator char in the script
			containerCommand = strings.Join(splitted[2:], ";")
		} else {
			containerCommand = splitted[2]
		}
		containerCommands := splitContainerCommand(containerCommand)
		customTasks[i] = v1.Task{
			Custom: &v1.UserTask{
				BaseTask: v1.BaseTask{
					Name:          splitted[0],
					Configuration: *taskConfOrDefault(tasksConf, splitted[0]),
				},
				ContainerImage:    splitted[1],
				ContainerCommands: containerCommands,
			},
		}
	}
	return customTasks, nil
}

func taskConfOrDefault(tasksConf map[string]*v1.BuildConfiguration, taskName string) *v1.BuildConfiguration {
	if tasksConf == nil || tasksConf[taskName] == nil {
		return &v1.BuildConfiguration{}
	}

	return tasksConf[taskName]
}

func (t *builderTrait) parseTasksConf() (map[string]*v1.BuildConfiguration, error) {
	tasksConf := make(map[string]*v1.BuildConfiguration)

	for _, t := range t.TasksRequestCPU {
		splits := strings.Split(t, ":")
		if len(splits) != 2 {
			return nil, fmt.Errorf("could not parse %s, expected format <task-name>:<task-resource>", t)
		}
		taskName := splits[0]
		taskResource := splits[1]
		if tasksConf[taskName] == nil {
			tasksConf[taskName] = &v1.BuildConfiguration{}
		}
		tasksConf[taskName].RequestCPU = taskResource
	}

	for _, t := range t.TasksRequestMemory {
		splits := strings.Split(t, ":")
		if len(splits) != 2 {
			return nil, fmt.Errorf("could not parse %s, expected format <task-name>:<task-resource>", t)
		}
		taskName := splits[0]
		taskResource := splits[1]
		if tasksConf[taskName] == nil {
			tasksConf[taskName] = &v1.BuildConfiguration{}
		}
		tasksConf[taskName].RequestMemory = taskResource
	}

	for _, t := range t.TasksLimitCPU {
		splits := strings.Split(t, ":")
		if len(splits) != 2 {
			return nil, fmt.Errorf("could not parse %s, expected format <task-name>:<task-resource>", t)
		}
		taskName := splits[0]
		taskResource := splits[1]
		if tasksConf[taskName] == nil {
			tasksConf[taskName] = &v1.BuildConfiguration{}
		}
		tasksConf[taskName].LimitCPU = taskResource
	}

	for _, t := range t.TasksLimitMemory {
		splits := strings.Split(t, ":")
		if len(splits) != 2 {
			return nil, fmt.Errorf("could not parse %s, expected format <task-name>:<task-resource>", t)
		}
		taskName := splits[0]
		taskResource := splits[1]
		if tasksConf[taskName] == nil {
			tasksConf[taskName] = &v1.BuildConfiguration{}
		}
		tasksConf[taskName].LimitMemory = taskResource
	}

	return tasksConf, nil
}

// we may get a command in the following format `/bin/bash -c "ls && echo 'hello'`
// which should provide a string with {"/bin/bash", "-c", "ls && echo 'hello'"}.
// if however we have a command which is not quoted, then we leave it the way it is.
func splitContainerCommand(command string) []string {
	if !strings.Contains(command, "\"") {
		// No quotes, then, splits all commands found
		return strings.Split(command, " ")
	}
	matches := commandsRegexp.FindAllString(command, -1)
	removeQuotes := make([]string, 0, len(matches))
	for _, m := range matches {
		removeQuotes = append(removeQuotes, strings.ReplaceAll(m, "\"", ""))
	}

	return removeQuotes
}
