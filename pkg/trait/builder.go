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
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/apache/camel-k/v2/pkg/util/boolean"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/builder"
	"github.com/apache/camel-k/v2/pkg/platform"
	mvn "github.com/apache/camel-k/v2/pkg/util/maven"
	"github.com/apache/camel-k/v2/pkg/util/property"
)

const (
	builderTraitID    = "builder"
	builderTraitOrder = 600

	quarkusNativeTaskName = "quarkus-native"
)

var commandsRegexp = regexp.MustCompile(`"[^"]+"|[\w/-]+`)

type builderTrait struct {
	BasePlatformTrait
	traitv1.BuilderTrait `property:",squash"`

	// no task when empty
	quarkusNativeTask string
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
	if t.BaseImage != otherTrait.BaseImage || len(t.Properties) != len(otherTrait.Properties) {
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

	return slices.Equal(srtThisProps, srtOtheProps)
}

func (t *builderTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.IntegrationKit != nil || e.IntegrationInPhase(v1.IntegrationPhaseBuildSubmitted) {
		t.setPlatform(e)
		if e.IntegrationKit != nil && !e.IntegrationKitInPhase(v1.IntegrationKitPhaseBuildSubmitted) {
			return false, nil, nil
		}

		trait := e.Catalog.GetTrait(quarkusTraitID)
		if trait != nil {
			condition, err := t.configureForQuarkus(trait, e)
			if err != nil {
				return false, condition, err
			}
		}

		return true, nil, nil
	}

	return false, nil, nil
}

func (t *builderTrait) configureForQuarkus(trait Trait, e *Environment) (*TraitCondition, error) {
	var condition *TraitCondition
	quarkus, ok := trait.(*quarkusTrait)
	isNativeIntegration := quarkus.isNativeIntegration(e)

	isNativeKit, err := quarkus.isNativeKit(e)
	if err != nil {
		return nil, err
	}

	if ok && (isNativeIntegration || isNativeKit) {
		// TODO expect maven repository in local repo (need to change builder pod accordingly!)
		command := builder.QuarkusRuntimeSupport(e.CamelCatalog.GetCamelQuarkusVersion()).BuildCommands()
		nativeBuilderImage := quarkus.NativeBuilderImage
		if nativeBuilderImage == "" {
			// default from the catalog
			nativeBuilderImage = e.CamelCatalog.GetQuarkusToolingImage()
			// GraalVM expected version is 25.0.0 starting from Quarkus 3.31
			// As we cannot change the catalog, until we reach 3.x, we can patch it here
			nativeBuilderImage = strings.ReplaceAll(nativeBuilderImage, "jdk-21", "jdk-25")
		}

		// Force the build to run in a separate Pod and strictly sequential
		m := "This is a Quarkus native build: setting build configuration with build Pod strategy and native container sensible resources " +
			"(if not specified by the user). Make sure your cluster can handle it."
		t.L.Info(m)
		condition = NewIntegrationCondition("Builder", v1.IntegrationConditionTraitInfo, corev1.ConditionTrue, TraitConfigurationReason, m)

		if !existsTaskRequest(t.TasksRequestCPU, quarkusNativeTaskName) {
			t.TasksRequestCPU = append(t.TasksRequestCPU, quarkusNativeTaskName+":1000m")
		}
		if !existsTaskRequest(t.TasksRequestMemory, quarkusNativeTaskName) {
			t.TasksRequestMemory = append(t.TasksRequestMemory, quarkusNativeTaskName+":4Gi")
		}

		t.quarkusNativeTask = fmt.Sprintf(`%s;%s;/bin/bash -c "%s"`, quarkusNativeTaskName, nativeBuilderImage, command)
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

func (t *builderTrait) Apply(e *Environment) error {
	// local pipeline tasks
	var pipelineTasks []v1.Task

	// task configuration resources
	defaultBuildConf := &e.Platform.BuildConfiguration

	tasksConf, err := t.parseTasksConf(defaultBuildConf)
	if err != nil {
		return err
	}

	imageName, err := getImageName(e)
	if err != nil {
		return err
	}

	// Building task
	builderTask, err := t.builderTask(e, taskConfOrDefault(tasksConf, "builder"))
	if err != nil {
		if err := failIntegrationKit(
			e,
			"IntegrationKitPropertiesFormatValid",
			corev1.ConditionFalse,
			"IntegrationKitPropertiesFormatValid",
			"One or more properties where not formatted as expected: "+err.Error(),
		); err != nil {
			return err
		}

		return nil
	}
	builderTask.Configuration.NodeSelector = t.filterNodeSelector()
	builderTask.Configuration.Annotations = t.Annotations
	pipelineTasks = append(pipelineTasks, v1.Task{Builder: builderTask})

	if t.quarkusNativeTask != "" {
		quarkusNativeTask, err := parseTask(t.quarkusNativeTask, tasksConf, imageName)
		if err != nil {
			return err
		}
		pipelineTasks = append(pipelineTasks, *quarkusNativeTask)
	}

	// Packaging task
	// It's the same builder configuration, but with different steps and conf
	packageTask := builderTask.DeepCopy()
	packageTask.Name = "package"
	packageTask.Configuration = *taskConfOrDefault(tasksConf, "package")
	packageTask.Steps = make([]string, 0)
	pipelineTasks = append(pipelineTasks, v1.Task{Package: packageTask})

	// Publishing task
	jibTask := v1.Task{Jib: &v1.JibTask{
		Name:          "jib",
		Configuration: *taskConfOrDefault(tasksConf, "jib"),
		BaseImage:     t.getBaseImage(e),
		Image:         imageName,
		Registry:      e.Platform.Registry,
	}}
	if t.ImagePlatforms != nil {
		jibTask.Jib.Configuration.ImagePlatforms = t.ImagePlatforms
	}
	pipelineTasks = append(pipelineTasks, jibTask)

	// add local pipeline tasks to env pipeline
	e.Pipeline = append(e.Pipeline, pipelineTasks...)

	return nil
}

// when this trait fails, we must report the failure into the related IntegrationKit if it affects the success of the Build.
func failIntegrationKit(e *Environment, conditionType v1.IntegrationKitConditionType, status corev1.ConditionStatus, reason, message string) error {
	if e.IntegrationKit != nil {
		e.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseError
		e.IntegrationKit.Status.SetCondition(conditionType, status, reason, message)
		e.IntegrationKit.Status.Failure = &v1.Failure{
			Reason: message,
			Time:   metav1.Now(),
		}
		if err := e.Client.Status().Update(e.Ctx, e.IntegrationKit); err != nil {
			return err
		}
	}

	return nil
}

//nolint:gocognit
func (t *builderTrait) builderTask(e *Environment, taskConf *v1.BuildConfiguration) (*v1.BuilderTask, error) {
	maven := e.Platform.Maven

	// Add Maven repositories defined in the IntegrationKit or Integration
	if e.IntegrationKit != nil {
		for _, repo := range e.IntegrationKit.Spec.Repositories {
			if !platform.IsMavenRepoAllowed(repo) {
				return nil, fmt.Errorf("maven repository %s is not allowed by the operator", repo)
			}
			maven.Repositories = append(maven.Repositories, mvn.NewRepository(repo))
		}
	}
	if e.Integration != nil {
		for _, repo := range e.Integration.Spec.Repositories {
			if !platform.IsMavenRepoAllowed(repo) {
				return nil, fmt.Errorf("maven repository %s is not allowed by the operator", repo)
			}
			maven.Repositories = append(maven.Repositories, mvn.NewRepository(repo))
		}
	}

	dependencies := getDependencies(e)

	task := &v1.BuilderTask{
		Name:          "builder",
		Configuration: *taskConf,
		BaseImage:     t.getBaseImage(e),
		Runtime:       e.CamelCatalog.Runtime,
		Dependencies:  dependencies,
		Maven:         maven,
	}

	if e.Integration != nil && e.Integration.Spec.Git != nil {
		task.Git = e.Integration.Spec.Git
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
	if e.IntegrationKit != nil && e.IntegrationKit.HasCapability("master") &&
		e.CamelCatalog.Runtime.Capabilities["master"].BuildTimeProperties != nil {
		task.Maven.Properties["camel.k.master.enabled"] = boolean.TrueString
		for _, cp := range e.CamelCatalog.Runtime.Capabilities["master"].BuildTimeProperties {
			task.Maven.Properties[CapabilityPropertyKey(cp.Key, task.Maven.Properties)] = cp.Value
		}
	}

	steps := make([]builder.Step, 0, len(builder.Project.CommonSteps))
	steps = append(steps, builder.Project.CommonSteps...)

	// sort steps by phase
	sort.SliceStable(steps, func(i, j int) bool {
		return steps[i].Phase() < steps[j].Phase()
	})

	task.Steps = builder.StepIDsFor(steps...)

	return task, nil
}

func getDependencies(e *Environment) []string {
	if e.IntegrationKit != nil {
		return e.IntegrationKit.Spec.Dependencies
	}

	return nil
}

func getImageName(e *Environment) (string, error) {
	if e.Platform.Registry.Address == "" {
		return "", errors.New("missing registry address")
	}
	var imageName string
	if e.IntegrationKit != nil {
		imageName = fmt.Sprintf("%s:%s", e.IntegrationKit.Name, e.IntegrationKit.ResourceVersion)
	} else {
		imageName = fmt.Sprintf("%s:%s", e.Integration.Name, e.Integration.ResourceVersion)
	}
	organization := e.Platform.Registry.Organization
	if organization == "" {
		// default organization name to catalog namespace (same as deprecated IntegrationPlatform namespace)
		organization = e.Platform.CatalogNamespace
	}

	return e.Platform.Registry.Address + "/" + organization + "/camel-k-" + imageName, nil
}

func (t *builderTrait) getBaseImage(e *Environment) string {
	baseImage := t.BaseImage
	if baseImage == "" {
		baseImage = e.Platform.BuildBaseImage
	}

	return baseImage
}

func parseTask(task string, tasksConf map[string]*v1.BuildConfiguration, imageName string) (*v1.Task, error) {
	splitted := strings.Split(task, ";")
	if len(splitted) < 3 {
		return nil, fmt.Errorf(`provide a custom task with at least 3 arguments, ie "my-task-name;my-image;echo 'hello'", was %v`, task)
	}
	parsedTask := v1.Task{
		Custom: &v1.UserTask{
			Name:              splitted[0],
			Configuration:     *taskConfOrDefault(tasksConf, splitted[0]),
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
		parsedTask.Custom.ContainerUserID = &uid
	}

	return &parsedTask, nil
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

// Will set a default platform if either specified in the trait or the platform/profile configuration.
func (t *builderTrait) setPlatform(e *Environment) {
	if t.ImagePlatforms == nil {
		if e.Platform.BuildConfiguration.ImagePlatforms != nil {
			t.ImagePlatforms = e.Platform.BuildConfiguration.ImagePlatforms
		}
	}
}

// filterNodeSelector returns the node selector map after applying the operator-level allow list.
// If BUILDER_NODE_SELECTOR_ALLOWED_LABELS is empty/unset all keys are accepted (backward compatible).
// Otherwise only keys present in the allow list are retained; every dropped key is logged at info level.
func (t *builderTrait) filterNodeSelector() map[string]string {
	if len(t.NodeSelector) == 0 {
		return t.NodeSelector
	}

	allowList := platform.BuilderNodeSelectorAllowList()
	if len(allowList) == 0 {
		// no restriction configured – accept everything
		return t.NodeSelector
	}

	filtered := make(map[string]string, len(t.NodeSelector))
	for k, v := range t.NodeSelector {
		if slices.Contains(allowList, k) {
			filtered[k] = v
		} else {
			t.L.Info("builder.nodeSelector key is not in the allowed list and will be ignored",
				"key", k, "allowedKeys", allowList)
		}
	}

	return filtered
}
