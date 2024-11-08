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
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/apache/camel-k/v2/pkg/util/boolean"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/builder"
	"github.com/apache/camel-k/v2/pkg/util/jib"
	mvn "github.com/apache/camel-k/v2/pkg/util/maven"
	"github.com/apache/camel-k/v2/pkg/util/property"
)

const (
	builderTraitID    = "builder"
	builderTraitOrder = 600
)

var commandsRegexp = regexp.MustCompile(`"[^"]+"|[\w/-]+`)

type builderTrait struct {
	BasePlatformTrait
	traitv1.BuilderTrait `property:",squash"`
}

func newBuilderTrait() Trait {
	return &builderTrait{
		BasePlatformTrait: NewBasePlatformTrait(builderTraitID, builderTraitOrder),
	}
}

// InfluencesKit overrides base class method.
func (t *builderTrait) InfluencesKit() bool {
	return true
}

func (t *builderTrait) Matches(trait Trait) bool {
	otherTrait, ok := trait.(*builderTrait)
	if !ok {
		return false
	}
	if t.BaseImage != otherTrait.BaseImage || len(t.Properties) != len(otherTrait.Properties) || len(t.Tasks) != len(otherTrait.Tasks) {
		return false
	}
	// More sofisticated check if len is the same. Sort and compare via slices equal func.
	// Although the Matches func is used as a support for comparison, it makes sense
	// to copy the properties and avoid possible inconsistencies caused by the sorting operation.
	srtThisProps := make([]string, len(t.Properties))
	srtOtheProps := make([]string, len(otherTrait.Properties))
	copy(srtThisProps, t.Properties)
	copy(srtOtheProps, otherTrait.Properties)
	slices.Sort(srtThisProps)
	slices.Sort(srtOtheProps)
	if !slices.Equal(srtThisProps, srtOtheProps) {
		return false
	}
	srtThisTasks := make([]string, len(t.Tasks))
	srtOtheTasks := make([]string, len(otherTrait.Tasks))
	copy(srtThisTasks, t.Tasks)
	copy(srtOtheTasks, otherTrait.Tasks)
	slices.Sort(srtThisTasks)
	slices.Sort(srtOtheTasks)
	return slices.Equal(srtThisTasks, srtOtheTasks)
}

func (t *builderTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.IntegrationKit == nil {
		return false, nil, nil
	}
	condition := t.adaptDeprecatedFields()
	if e.Platform.Status.Build.PublishStrategy == v1.IntegrationPlatformBuildPublishStrategySpectrum {
		condition = newOrAppend(condition, "Spectrum publishing strategy is deprecated and may be removed in future releases. Make sure to use any supported publishing strategy instead.")
	}

	t.setPlatform(e)

	if !e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted) {
		return false, condition, nil
	}

	trait := e.Catalog.GetTrait(quarkusTraitID)
	if trait != nil {
		condition, err := t.configureForQuarkus(trait, e, condition)
		if err != nil {
			return false, condition, err
		}
	}

	return true, condition, nil
}

func (t *builderTrait) configureForQuarkus(trait Trait, e *Environment, condition *TraitCondition) (*TraitCondition, error) {
	quarkus, ok := trait.(*quarkusTrait)
	isNativeIntegration := quarkus.isNativeIntegration(e)

	isNativeKit, err := quarkus.isNativeKit(e)
	if err != nil {
		return condition, err
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
		m := "This is a Quarkus native build: setting build configuration with build Pod strategy and native container sensible resources (if not specified by the user). Make sure your cluster can handle it."
		t.L.Info(m)

		condition = newOrAppend(condition, m)
		t.Strategy = string(v1.BuildStrategyPod)
		t.OrderStrategy = string(v1.BuildOrderStrategySequential)

		if !existsTaskRequest(t.TasksRequestCPU, "quarkus-native") {
			t.TasksRequestCPU = append(t.TasksRequestCPU, "quarkus-native:1000m")
		}
		if !existsTaskRequest(t.TasksRequestMemory, "quarkus-native") {
			t.TasksRequestMemory = append(t.TasksRequestMemory, "quarkus-native:4Gi")
		}
	}

	return condition, nil
}

func existsTaskRequest(tasks []string, taskName string) bool {
	for _, task := range tasks {
		ts := strings.Split(task, ":")
		if len(ts) == 2 && ts[0] == taskName {
			return true
		}
	}
	return false
}

func (t *builderTrait) adaptDeprecatedFields() *TraitCondition {
	var condition *TraitCondition
	if t.RequestCPU != "" {
		m := "The request-cpu parameter is deprecated and may be removed in future releases. Make sure to use tasks-request-cpu parameter instead."
		t.L.Info(m)
		condition = newOrAppend(condition, m)
		t.TasksRequestCPU = append(t.TasksRequestCPU, fmt.Sprintf("builder:%s", t.RequestCPU))
	}
	if t.LimitCPU != "" {
		m := "The limit-cpu parameter is deprecated and may be removed in future releases. Make sure to use tasks-limit-cpu parameter instead."
		t.L.Info(m)
		condition = newOrAppend(condition, m)
		t.TasksLimitCPU = append(t.TasksLimitCPU, fmt.Sprintf("builder:%s", t.LimitCPU))
	}
	if t.RequestMemory != "" {
		m := "The request-memory parameter is deprecated and may be removed in future releases. Make sure to use tasks-request-memory parameter instead."
		t.L.Info(m)
		condition = newOrAppend(condition, m)
		t.TasksRequestMemory = append(t.TasksRequestMemory, fmt.Sprintf("builder:%s", t.RequestMemory))
	}
	if t.LimitMemory != "" {
		m := "The limit-memory parameter is deprecated and may be removed in future releases. Make sure to use tasks-limit-memory parameter instead."
		t.L.Info(m)
		if condition == nil {
			condition = NewIntegrationCondition("Builder", v1.IntegrationConditionTraitInfo, corev1.ConditionTrue, TraitConfigurationReason, "")
		}
		condition = newOrAppend(condition, m)
		t.TasksLimitMemory = append(t.TasksLimitMemory, fmt.Sprintf("builder:%s", t.LimitMemory))
	}

	return condition
}

func newOrAppend(condition *TraitCondition, message string) *TraitCondition {
	if condition == nil {
		condition = NewIntegrationCondition("Builder", v1.IntegrationConditionTraitInfo, corev1.ConditionTrue, TraitConfigurationReason, message)
	} else {
		condition.message += "; " + message
	}

	return condition
}

func (t *builderTrait) Apply(e *Environment) error {
	// local pipeline tasks
	var pipelineTasks []v1.Task

	// task configuration resources
	defaultBuildConf := &v1.BuildConfiguration{}
	if e.Platform != nil {
		defaultBuildConf = &e.Platform.Status.Build.BuildConfiguration
	}
	tasksConf, err := t.parseTasksConf(defaultBuildConf)
	if err != nil {
		return err
	}

	imageName := getImageName(e)
	// Building task
	builderTask, err := t.builderTask(e, taskConfOrDefault(tasksConf, "builder"))
	if err != nil {
		if err := failIntegrationKit(
			e,
			"IntegrationKitPropertiesFormatValid",
			corev1.ConditionFalse,
			"IntegrationKitPropertiesFormatValid",
			fmt.Sprintf("One or more properties where not formatted as expected: %s", err.Error()),
		); err != nil {
			return err
		}
		return nil
	}
	builderTask.Configuration.NodeSelector = t.NodeSelector
	builderTask.Configuration.Annotations = t.Annotations
	pipelineTasks = append(pipelineTasks, v1.Task{Builder: builderTask})

	// Custom tasks
	if t.Tasks != nil {
		ct, err := t.determineCustomTasks(e, builderTask, tasksConf)
		if err != nil {
			return err
		}
		if ct == nil {
			return nil
		}

		pipelineTasks = append(pipelineTasks, ct...)
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
				Image:     imageName,
				Registry:  e.Platform.Status.Build.Registry,
			},
		}})

	case v1.IntegrationPlatformBuildPublishStrategyJib:
		jibTask := v1.Task{Jib: &v1.JibTask{
			BaseTask: v1.BaseTask{
				Name:          "jib",
				Configuration: *taskConfOrDefault(tasksConf, "jib"),
			},
			PublishTask: v1.PublishTask{
				BaseImage: t.getBaseImage(e),
				Image:     imageName,
				Registry:  e.Platform.Status.Build.Registry,
			},
		}}
		if t.ImagePlatforms != nil {
			jibTask.Jib.Configuration.ImagePlatforms = t.ImagePlatforms
		}
		pipelineTasks = append(pipelineTasks, jibTask)

	case v1.IntegrationPlatformBuildPublishStrategyS2I:
		pipelineTasks = append(pipelineTasks, v1.Task{S2i: &v1.S2iTask{
			BaseTask: v1.BaseTask{
				Name:          "s2i",
				Configuration: *taskConfOrDefault(tasksConf, "s2i"),
			},
			PublishTask: v1.PublishTask{
				BaseImage: t.getBaseImage(e),
				Image:     imageName,
			},
			Tag: e.IntegrationKit.ResourceVersion,
		}})
	}

	// filter only those tasks required by the user
	if t.TasksFilter != "" {
		flt := strings.Split(t.TasksFilter, ",")
		if pipelineTasks, err = filter(pipelineTasks, flt); err != nil {
			if err := failIntegrationKit(
				e,
				"IntegrationKitTasksValid",
				corev1.ConditionFalse,
				"IntegrationKitTasksValid",
				err.Error(),
			); err != nil {
				return err
			}
			return err
		}
	}
	// add local pipeline tasks to env pipeline
	e.Pipeline = append(e.Pipeline, pipelineTasks...)
	return nil
}

// when this trait fails, we must report the failure into the related IntegrationKit if it affects the success of the Build.
func failIntegrationKit(e *Environment, conditionType v1.IntegrationKitConditionType, status corev1.ConditionStatus, reason, message string) error {
	e.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseError
	e.IntegrationKit.Status.SetCondition(conditionType, status, reason, message)
	if err := e.Client.Status().Update(e.Ctx, e.IntegrationKit); err != nil {
		return err
	}
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
	// User provided build-time properties
	if t.Properties != nil {
		for _, v := range t.Properties {
			key, value := property.SplitPropertyFileEntry(v)
			if len(key) == 0 || len(value) == 0 {
				return nil, fmt.Errorf("maven property must have key=value format, it was %v", v)
			}

			task.Maven.Properties[key] = value
		}
	}

	// Build time property required by master capability
	if e.IntegrationKit.HasCapability("master") && e.CamelCatalog.Runtime.Capabilities["master"].BuildTimeProperties != nil {
		task.Maven.Properties["camel.k.master.enabled"] = boolean.TrueString
		for _, cp := range e.CamelCatalog.Runtime.Capabilities["master"].BuildTimeProperties {
			task.Maven.Properties[CapabilityPropertyKey(cp.Key, task.Maven.Properties)] = cp.Value
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
				mavenProfile, err := v1.DecodeValueSource(v, "profile.xml")
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

func (t *builderTrait) determineCustomTasks(e *Environment, builderTask *v1.BuilderTask, tasksConf map[string]*v1.BuildConfiguration) ([]v1.Task, error) {
	imageName := getImageName(e)

	realBuildStrategy := builderTask.Configuration.Strategy
	if realBuildStrategy == "" {
		realBuildStrategy = e.Platform.Status.Build.BuildConfiguration.Strategy
	}

	if len(t.Tasks) > 0 && realBuildStrategy != v1.BuildStrategyPod {
		err := failIntegrationKit(
			e,
			"IntegrationKitTasksValid",
			corev1.ConditionFalse,
			"IntegrationKitTasksValid",
			fmt.Sprintf("Pipeline tasks unavailable when using `%s` platform build strategy: use `%s` instead.",
				realBuildStrategy,
				v1.BuildStrategyPod),
		)

		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	return t.customTasks(tasksConf, imageName)
}

// the format expected is "<task-name>;<task-image>;<task-container-command>[;<task-container-user-id>]".
func (t *builderTrait) customTasks(tasksConf map[string]*v1.BuildConfiguration, imageName string) ([]v1.Task, error) {
	customTasks := make([]v1.Task, len(t.Tasks))

	for i, t := range t.Tasks {
		splitted := strings.Split(t, ";")
		if len(splitted) < 3 {
			return nil, fmt.Errorf(`provide a custom task with at least 3 arguments, ie "my-task-name;my-image;echo 'hello'", was %v`, t)
		}
		customTasks[i] = v1.Task{
			Custom: &v1.UserTask{
				BaseTask: v1.BaseTask{
					Name:          splitted[0],
					Configuration: *taskConfOrDefault(tasksConf, splitted[0]),
				},
				PublishingImage:   imageName,
				ContainerImage:    splitted[1],
				ContainerCommands: splitContainerCommand(splitted[2]),
			},
		}
		if len(splitted) > 3 {
			uid, err := strconv.ParseInt(splitted[3], 10, 64)
			if err != nil {
				return nil, fmt.Errorf(`provide a custom task with a correct numeric user id, was %v`, splitted[3])
			}
			customTasks[i].Custom.ContainerUserID = &uid
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

func (t *builderTrait) parseTasksConf(defaultBuildConf *v1.BuildConfiguration) (map[string]*v1.BuildConfiguration, error) {
	tasksConf := make(map[string]*v1.BuildConfiguration)

	for _, t := range t.TasksRequestCPU {
		splits := strings.Split(t, ":")
		if len(splits) != 2 {
			return nil, fmt.Errorf("could not parse %s, expected format <task-name>:<task-resource>", t)
		}
		taskName := splits[0]
		taskResource := splits[1]
		if tasksConf[taskName] == nil {
			tasksConf[taskName] = defaultBuildConf
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
			tasksConf[taskName] = defaultBuildConf
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
			tasksConf[taskName] = defaultBuildConf
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
			tasksConf[taskName] = defaultBuildConf
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

func filter(tasks []v1.Task, filterTasks []string) ([]v1.Task, error) {
	var filteredTasks []v1.Task
	for _, f := range filterTasks {
		found := false

		for _, t := range tasks {
			switch {
			case t.Builder != nil && t.Builder.Name == f:
				filteredTasks = append(filteredTasks, t)
				found = true
			case t.Custom != nil && t.Custom.Name == f:
				filteredTasks = append(filteredTasks, t)
				found = true
			case t.Package != nil && t.Package.Name == f:
				filteredTasks = append(filteredTasks, t)
				found = true
			case t.Spectrum != nil && t.Spectrum.Name == f:
				filteredTasks = append(filteredTasks, t)
				found = true
			case t.S2i != nil && t.S2i.Name == f:
				filteredTasks = append(filteredTasks, t)
				found = true
			case t.Jib != nil && t.Jib.Name == f:
				filteredTasks = append(filteredTasks, t)
				found = true
			}
		}

		if !found {
			// If we reach this point it means no tasks exists for the name
			return nil, fmt.Errorf("no task exist for %s name", f)
		}
	}
	// make sure the last task is either a publishing task or a custom task
	if len(filteredTasks) == 0 || !publishingOrUserTask(filteredTasks[len(filteredTasks)-1]) {
		return nil, fmt.Errorf("last pipeline task is not a publishing or a user task")
	}
	return filteredTasks, nil
}

// return true if the task is either a publishing task or a custom user task.
func publishingOrUserTask(t v1.Task) bool {
	switch {
	case t.Custom != nil:
		return true
	case t.Spectrum != nil:
		return true
	case t.Jib != nil:
		return true
	}

	return false
}

// Will set a default platform if either specified in the trait or the platform/profile configuration.
func (t *builderTrait) setPlatform(e *Environment) {
	if t.ImagePlatforms == nil {
		if e.Platform != nil && e.Platform.Status.Build.BuildConfiguration.ImagePlatforms != nil {
			t.ImagePlatforms = e.Platform.Status.Build.BuildConfiguration.ImagePlatforms
		}
	}
}
