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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestBuilderTraitNotAppliedBecauseOfNilKit(t *testing.T) {
	environments := []*Environment{
		createBuilderTestEnv(v1.IntegrationPlatformClusterOpenShift, v1.IntegrationPlatformBuildPublishStrategyS2I, v1.BuildStrategyRoutine),
	}

	for _, e := range environments {
		e := e // pin
		e.IntegrationKit = nil

		t.Run(string(e.Platform.Status.Cluster), func(t *testing.T) {
			trait, _ := newBuilderTrait().(*builderTrait)

			configure, conditions, err := trait.Configure(e)
			assert.False(t, configure)
			assert.Empty(t, conditions)
			require.NoError(t, err)
			assert.Nil(t, e.GetTrait("builder"))
			assert.Empty(t, e.Pipeline)
		})
	}
}

func TestBuilderTraitNotAppliedBecauseOfNilPhase(t *testing.T) {
	environments := []*Environment{
		createBuilderTestEnv(v1.IntegrationPlatformClusterOpenShift, v1.IntegrationPlatformBuildPublishStrategyS2I, v1.BuildStrategyRoutine),
	}

	for _, e := range environments {
		e := e // pin
		e.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseInitialization

		t.Run(string(e.Platform.Status.Cluster), func(t *testing.T) {
			conditions, traits, err := NewBuilderTestCatalog().apply(e)

			require.NoError(t, err)
			assert.NotEmpty(t, conditions)
			assert.Empty(t, traits)
			assert.NotEmpty(t, e.ExecutedTraits)
			assert.Nil(t, e.GetTrait("builder"))
			assert.Empty(t, e.Pipeline)
		})
	}
}

func TestS2IBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterOpenShift, v1.IntegrationPlatformBuildPublishStrategyS2I, v1.BuildStrategyRoutine)
	conditions, traits, err := NewBuilderTestCatalog().apply(env)

	require.NoError(t, err)
	assert.Empty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, env.ExecutedTraits)
	assert.NotNil(t, env.GetTrait("builder"))
	assert.NotEmpty(t, env.Pipeline)
	assert.Len(t, env.Pipeline, 3)
	assert.NotNil(t, env.Pipeline[0].Builder)
	assert.NotNil(t, env.Pipeline[1].Package)
	assert.NotNil(t, env.Pipeline[2].S2i)
	assert.Equal(t, "root-jdk-image", env.Pipeline[2].S2i.BaseImage)
	assert.Empty(t, env.Pipeline[2].S2i.Registry)
}

func TestJibBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterOpenShift, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyRoutine)
	conditions, traits, err := NewBuilderTestCatalog().apply(env)

	require.NoError(t, err)
	assert.NotEmpty(t, traits)
	assert.NotEmpty(t, conditions)
	assert.NotEmpty(t, env.ExecutedTraits)
	assert.NotNil(t, env.GetTrait("builder"))
	assert.NotEmpty(t, env.Pipeline)
	assert.Len(t, env.Pipeline, 3)
	assert.NotNil(t, env.Pipeline[0].Builder)
	assert.NotNil(t, env.Pipeline[1].Package)
	assert.NotNil(t, env.Pipeline[2].Jib)
	assert.Equal(t, "root-jdk-image", env.Pipeline[2].Jib.BaseImage)
	assert.NotEmpty(t, env.Pipeline[2].Jib.Registry)
}

func createBuilderTestEnv(cluster v1.IntegrationPlatformCluster, strategy v1.IntegrationPlatformBuildPublishStrategy, buildStrategy v1.BuildStrategy) *Environment {
	c, err := camel.DefaultCatalog()
	if err != nil {
		panic(err)
	}
	itk := &v1.IntegrationKit{
		Status: v1.IntegrationKitStatus{
			Phase: v1.IntegrationKitPhaseBuildSubmitted,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-kit",
		},
	}
	client, _ := internal.NewFakeClient(itk)
	res := &Environment{
		Ctx:          context.TODO(),
		CamelCatalog: c,
		Catalog:      NewCatalog(nil),
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: itk,
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: cluster,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: strategy,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  defaults.DefaultRuntimeVersion,
					RuntimeProvider: v1.RuntimeProviderQuarkus,
					BuildConfiguration: v1.BuildConfiguration{
						Strategy: buildStrategy,
					},
					BaseImage: "root-jdk-image",
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}

	res.Platform.ResyncStatusFullConfig()

	return res
}

func NewBuilderTestCatalog() *Catalog {
	return NewCatalog(nil)
}

func TestMavenPropertyBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyS2I, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Properties = append(builderTrait.Properties, "build-time-prop1=build-time-value1")

	err := builderTrait.Apply(env)

	require.NoError(t, err)
	assert.Equal(t, "build-time-value1", env.Pipeline[0].Builder.Maven.Properties["build-time-prop1"])
}

func createNominalBuilderTraitTest() *builderTrait {
	builderTrait, _ := newBuilderTrait().(*builderTrait)
	return builderTrait
}

func TestCustomTaskBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategySpectrum, v1.BuildStrategyPod)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, "test;alpine;ls")

	err := builderTrait.Apply(env)

	require.NoError(t, err)
	builderTask := findCustomTaskByName(env.Pipeline, "builder")
	customTask := findCustomTaskByName(env.Pipeline, "test")
	packageTask := findCustomTaskByName(env.Pipeline, "package")
	publisherTask := findCustomTaskByName(env.Pipeline, "spectrum")

	assert.NotNil(t, builderTask)
	assert.NotNil(t, customTask)
	assert.NotNil(t, packageTask)
	assert.NotNil(t, publisherTask)
	assert.Len(t, env.Pipeline, 4)
	assert.Equal(t, "test", customTask.Custom.Name)
	assert.Equal(t, "alpine", customTask.Custom.ContainerImage)
	assert.Equal(t, "ls", customTask.Custom.ContainerCommands[0])
}

func TestCustomTaskBuilderTraitValidStrategyOverride(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategySpectrum, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, "test;alpine;ls")
	builderTrait.Strategy = "pod"

	err := builderTrait.Apply(env)

	require.NoError(t, err)

	customTask := findCustomTaskByName(env.Pipeline, "test")

	assert.Len(t, env.Pipeline, 4)
	assert.Equal(t, "test", customTask.Custom.Name)
	assert.Equal(t, "alpine", customTask.Custom.ContainerImage)
	assert.Equal(t, "ls", customTask.Custom.ContainerCommands[0])
}

func TestCustomTaskBuilderTraitInvalidStrategy(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategySpectrum, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, "test;alpine;ls")

	err := builderTrait.Apply(env)

	// The error will be reported to IntegrationKits
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationKitPhaseError, env.IntegrationKit.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, env.IntegrationKit.Status.Conditions[0].Status)
	assert.Equal(t, env.IntegrationKit.Status.Conditions[0].Type, v1.IntegrationKitConditionType("IntegrationKitTasksValid"))
}

func TestCustomTaskBuilderTraitInvalidStrategyOverride(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategySpectrum, v1.BuildStrategyPod)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, "test;alpine;ls")
	builderTrait.Strategy = "routine"

	err := builderTrait.Apply(env)

	// The error will be reported to IntegrationKits
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationKitPhaseError, env.IntegrationKit.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, env.IntegrationKit.Status.Conditions[0].Status)
	assert.Equal(t, env.IntegrationKit.Status.Conditions[0].Type, v1.IntegrationKitConditionType("IntegrationKitTasksValid"))
}

func TestMavenProfilesBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyS2I, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.MavenProfiles = []string{"configmap:maven-profile/owasp-profile.xml", "secret:maven-profile-secret"}

	err := builderTrait.Apply(env)

	require.NoError(t, err)

	assert.Equal(t, v1.ValueSource{
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "maven-profile",
			},
			Key: "owasp-profile.xml",
		},
	}, env.Pipeline[0].Builder.Maven.MavenSpec.Profiles[0])
	assert.Equal(t, v1.ValueSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "maven-profile-secret",
			},
			Key: "profile.xml",
		},
	}, env.Pipeline[0].Builder.Maven.MavenSpec.Profiles[1])
}

func TestInvalidMavenProfilesBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyS2I, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.MavenProfiles = []string{"fakeprofile"}

	err := builderTrait.Apply(env)

	// The error will be reported to IntegrationKits
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationKitPhaseError, env.IntegrationKit.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, env.IntegrationKit.Status.Conditions[0].Status)
	assert.Contains(t, env.IntegrationKit.Status.Conditions[0].Message, "fakeprofile")
}

func TestMavenBuilderTraitJib(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()

	err := builderTrait.Apply(env)

	require.NoError(t, err)

	assert.Equal(t, v1.ValueSource{
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "my-kit-publish-jib-profile",
			},
			Key: "profile.xml",
		},
	}, env.Pipeline[0].Builder.Maven.MavenSpec.Profiles[0])
}

func TestBuilderCustomTasks(t *testing.T) {
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, "test;alpine;ls")
	builderTrait.Tasks = append(builderTrait.Tasks, `test;alpine;mvn test`)

	tasks, err := builderTrait.customTasks(nil, "my-kit-img")

	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "test", tasks[0].Custom.Name)
	assert.Equal(t, "alpine", tasks[0].Custom.ContainerImage)
	assert.Equal(t, "ls", tasks[0].Custom.ContainerCommands[0])
	assert.Equal(t, "test", tasks[1].Custom.Name)
	assert.Equal(t, "alpine", tasks[1].Custom.ContainerImage)
	assert.Equal(t, "mvn", tasks[1].Custom.ContainerCommands[0])
	assert.Equal(t, "test", tasks[1].Custom.ContainerCommands[1])

}

func TestBuilderCustomTasksFailure(t *testing.T) {
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, "test;alpine")

	_, err := builderTrait.customTasks(nil, "my-kit-img")

	require.Error(t, err)
	assert.Equal(t, "provide a custom task with at least 3 arguments, ie \"my-task-name;my-image;echo 'hello'\", was test;alpine", err.Error())
}

func TestBuilderCustomTasksBashScript(t *testing.T) {
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, "test;alpine;/bin/bash -c \"cd test && echo 'helooo'\"")

	tasks, err := builderTrait.customTasks(nil, "my-kit-img")

	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "test", tasks[0].Custom.Name)
	assert.Equal(t, "alpine", tasks[0].Custom.ContainerImage)
	assert.Equal(t, "/bin/bash", tasks[0].Custom.ContainerCommands[0])
	assert.Equal(t, "-c", tasks[0].Custom.ContainerCommands[1])
	assert.Equal(t, "cd test && echo 'helooo'", tasks[0].Custom.ContainerCommands[2])
}

func TestBuilderCustomTasksSecurityContextScript(t *testing.T) {
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, "test;alpine;/bin/bash -c \"cd test && echo 'helooo'\";1000")

	tasks, err := builderTrait.customTasks(nil, "my-kit-img")

	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "test", tasks[0].Custom.Name)
	assert.Equal(t, "alpine", tasks[0].Custom.ContainerImage)
	assert.Equal(t, "/bin/bash", tasks[0].Custom.ContainerCommands[0])
	assert.Equal(t, "-c", tasks[0].Custom.ContainerCommands[1])
	assert.Equal(t, "cd test && echo 'helooo'", tasks[0].Custom.ContainerCommands[2])
	assert.Equal(t, int64(1000), *tasks[0].Custom.ContainerUserID)
}

func TestBuilderCustomTasksConfiguration(t *testing.T) {
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.TasksRequestCPU = append(builderTrait.TasksRequestCPU, "builder:1000m")
	builderTrait.TasksLimitCPU = append(builderTrait.TasksLimitCPU, "custom1:500m")
	builderTrait.TasksRequestMemory = append(builderTrait.TasksRequestMemory, "package:8Gi")
	builderTrait.TasksLimitMemory = append(builderTrait.TasksLimitMemory, "spectrum:4Gi")

	tasksConf, err := builderTrait.parseTasksConf(&v1.BuildConfiguration{})

	require.NoError(t, err)
	assert.Equal(t, 4, len(tasksConf))
	assert.Equal(t, "1000m", tasksConf["builder"].RequestCPU)
	assert.Equal(t, "500m", tasksConf["custom1"].LimitCPU)
	assert.Equal(t, "8Gi", tasksConf["package"].RequestMemory)
	assert.Equal(t, "4Gi", tasksConf["spectrum"].LimitMemory)
}

func TestBuilderCustomTasksConfigurationError(t *testing.T) {
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.TasksLimitCPU = append(builderTrait.TasksLimitCPU, "syntax error")

	_, err := builderTrait.parseTasksConf(&v1.BuildConfiguration{})

	require.Error(t, err)
	assert.Equal(t, "could not parse syntax error, expected format <task-name>:<task-resource>", err.Error())
}

func TestUserTaskCommands(t *testing.T) {
	command := `/bin/bash -c "ls && echo 'hello' && $(cat /path/to/a/resource)"`
	podCommands := splitContainerCommand(command)

	assert.Len(t, podCommands, 3)
	assert.Equal(t, "/bin/bash", podCommands[0])
	assert.Equal(t, "-c", podCommands[1])
	assert.Equal(t, "ls && echo 'hello' && $(cat /path/to/a/resource)", podCommands[2])
}

func TestUserTaskSingleCommand(t *testing.T) {
	command := `cat /path/to/a/resource`
	podCommands := splitContainerCommand(command)

	assert.Len(t, podCommands, 2)
	assert.Equal(t, "cat", podCommands[0])
	assert.Equal(t, "/path/to/a/resource", podCommands[1])
}

func TestUserTaskMultiCommands(t *testing.T) {
	command := `"cat /path/to/a/resource" "echo ciao"`
	podCommands := splitContainerCommand(command)

	assert.Len(t, podCommands, 2)
	assert.Equal(t, "cat /path/to/a/resource", podCommands[0])
	assert.Equal(t, "echo ciao", podCommands[1])
}

func TestBuilderDeprecatedParams(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.LimitCPU = "100m"
	builderTrait.RequestCPU = "100m"
	builderTrait.LimitMemory = "100Mi"
	builderTrait.RequestMemory = "100Mi"

	active, condition, err := builderTrait.Configure(env)

	require.NoError(t, err)
	assert.True(t, active)
	assert.NotNil(t, condition)
	assert.Contains(t, condition.message, "The request-cpu parameter is deprecated and may be removed in future releases")
	assert.Contains(t, condition.message, "The limit-cpu parameter is deprecated and may be removed in future releases")
	assert.Contains(t, condition.message, "The request-memory parameter is deprecated and may be removed in future releases")
	assert.Contains(t, condition.message, "The limit-memory parameter is deprecated and may be removed in future releases")
	assert.Len(t, builderTrait.TasksLimitCPU, 1)
	assert.Len(t, builderTrait.TasksRequestCPU, 1)
	assert.Len(t, builderTrait.TasksLimitMemory, 1)
	assert.Len(t, builderTrait.TasksRequestMemory, 1)
	assert.Equal(t, "builder:100m", builderTrait.TasksLimitCPU[0])
	assert.Equal(t, "builder:100m", builderTrait.TasksRequestCPU[0])
	assert.Equal(t, "builder:100Mi", builderTrait.TasksLimitMemory[0])
	assert.Equal(t, "builder:100Mi", builderTrait.TasksRequestMemory[0])
}

func TestExistsTaskRequest(t *testing.T) {
	tasks := []string{"quarkus-native:1000m", "builder:1000m", "shouldfail"}
	assert.True(t, existsTaskRequest(tasks, "quarkus-native"))
	assert.False(t, existsTaskRequest(tasks, "quarkus"))
	assert.False(t, existsTaskRequest(tasks, "shouldfail"))
}

func TestBuilderWithNoNodeSelector(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()

	active, condition, err := builderTrait.Configure(env)
	require.NoError(t, err)

	err = builderTrait.Apply(env)
	require.NoError(t, err)

	assert.True(t, active)
	assert.Nil(t, condition)

	assert.Nil(t, builderTrait.NodeSelector)
	assert.Nil(t, env.Pipeline[0].Builder.Configuration.NodeSelector)
}

func TestBuilderWithNodeSelector(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.NodeSelector = map[string]string{
		"size": "large",
	}

	active, condition, err := builderTrait.Configure(env)
	require.NoError(t, err)

	err = builderTrait.Apply(env)
	require.NoError(t, err)

	assert.True(t, active)
	assert.Nil(t, condition)

	assert.Equal(t, map[string]string{"size": "large"}, env.Pipeline[0].Builder.Configuration.NodeSelector)
	assert.Equal(t, map[string]string{"size": "large"}, builderTrait.NodeSelector)
}

func TestBuilderWithAnnotations(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Annotations = map[string]string{
		"annotation": "value",
	}

	active, condition, err := builderTrait.Configure(env)
	require.NoError(t, err)

	err = builderTrait.Apply(env)
	require.NoError(t, err)

	assert.True(t, active)
	assert.Nil(t, condition)

	assert.Equal(t, map[string]string{"annotation": "value"}, env.Pipeline[0].Builder.Configuration.Annotations)
	assert.Equal(t, map[string]string{"annotation": "value"}, builderTrait.Annotations)
}

func TestBuilderNoTasksFilter(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyPod)
	builderTrait := createNominalBuilderTraitTest()

	err := builderTrait.Apply(env)
	require.NoError(t, err)

	pipelineTasks := tasksByName(env.Pipeline)
	assert.Equal(t, []string{"builder", "package", "jib"}, pipelineTasks)
}

func TestBuilderTasksFilterNotExistingTasks(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyPod)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.TasksFilter = "builder,missing-task"

	err := builderTrait.Apply(env)
	require.Error(t, err)
	assert.Equal(t, "no task exist for missing-task name", err.Error())
}

func TestBuilderTasksFilterMissingPublishTasks(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyPod)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.TasksFilter = "builder,package"

	err := builderTrait.Apply(env)
	require.Error(t, err)
	assert.Equal(t, "last pipeline task is not a publishing or a user task", err.Error())
}

func TestBuilderTasksFilterOperatorTasks(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyPod)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.TasksFilter = "builder,package,jib"

	err := builderTrait.Apply(env)
	require.NoError(t, err)
	pipelineTasks := tasksByName(env.Pipeline)
	assert.Equal(t, []string{"builder", "package", "jib"}, pipelineTasks)
}

func TestBuilderTasksFilterAndReorderOperatorTasks(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyPod)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.TasksFilter = "package,builder,jib"

	err := builderTrait.Apply(env)
	require.NoError(t, err)
	pipelineTasks := tasksByName(env.Pipeline)
	assert.Equal(t, []string{"package", "builder", "jib"}, pipelineTasks)
}

func TestBuilderTasksFilterAndReorderCustomTasks(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyPod)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, `my-custom-publish;alpine;mvn test`)
	builderTrait.Tasks = append(builderTrait.Tasks, "my-custom-task;alpine;ls")
	builderTrait.TasksFilter = "builder,my-custom-task,package,my-custom-publish"

	err := builderTrait.Apply(env)
	require.NoError(t, err)
	pipelineTasks := tasksByName(env.Pipeline)
	assert.Equal(t, []string{"builder", "my-custom-task", "package", "my-custom-publish"}, pipelineTasks)
}

func TestBuilderTasksNodeSelectorPlatformBuildStrategyPod(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyPod)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.NodeSelector = map[string]string{
		"kubernetes.io/arch": "amd64",
	}

	err := builderTrait.Apply(env)
	require.NoError(t, err)
	builderTask := getBuilderTask(env.Pipeline)
	assert.NotNil(t, builderTask)
	assert.Equal(t, map[string]string{"kubernetes.io/arch": "amd64"}, builderTask.Configuration.NodeSelector)
}

func findCustomTaskByName(tasks []v1.Task, name string) v1.Task {
	for _, t := range tasks {
		if t.Custom != nil && t.Custom.Name == name {
			return t
		}
	}
	return v1.Task{}
}

func tasksByName(tasks []v1.Task) []string {
	pipelineTasks := make([]string, len(tasks))
	for i, t := range tasks {
		if t.Builder != nil {
			pipelineTasks[i] = t.Builder.Name
		}
		if t.Custom != nil {
			pipelineTasks[i] = t.Custom.Name
		}
		if t.Package != nil {
			pipelineTasks[i] = t.Package.Name
		}
		if t.S2i != nil {
			pipelineTasks[i] = t.S2i.Name
		}
		if t.Spectrum != nil {
			pipelineTasks[i] = t.Spectrum.Name
		}
		if t.Jib != nil {
			pipelineTasks[i] = t.Jib.Name
		}
	}
	return pipelineTasks
}

func TestBuilderMatches(t *testing.T) {
	t1 := builderTrait{
		BasePlatformTrait: NewBasePlatformTrait("builder", 600),
		BuilderTrait: traitv1.BuilderTrait{
			OrderStrategy: "dependencies",
		},
	}
	t2 := builderTrait{
		BasePlatformTrait: NewBasePlatformTrait("builder", 600),
		BuilderTrait: traitv1.BuilderTrait{
			OrderStrategy: "dependencies",
		},
	}
	assert.True(t, t1.Matches(&t2))
	// This is a property that does not influence the build
	t2.OrderStrategy = "fifo"
	assert.True(t, t1.Matches(&t2))
	// Changing properties which influences build
	t1.Properties = []string{"hello=world"}
	assert.False(t, t1.Matches(&t2))
	t2.Properties = []string{"hello=world"}
	assert.True(t, t1.Matches(&t2))
	t1.Properties = []string{"hello=world", "weare=theworld"}
	assert.False(t, t1.Matches(&t2))
	// should detect swap
	t2.Properties = []string{"weare=theworld", "hello=world"}
	assert.True(t, t1.Matches(&t2))
}

func TestBuilderMatchesTasks(t *testing.T) {
	t1 := builderTrait{
		BasePlatformTrait: NewBasePlatformTrait("builder", 600),
		BuilderTrait:      traitv1.BuilderTrait{},
	}
	t2 := builderTrait{
		BasePlatformTrait: NewBasePlatformTrait("builder", 600),
		BuilderTrait: traitv1.BuilderTrait{
			Tasks: []string{"task1;my-task;do-something"},
		},
	}
	t3 := builderTrait{
		BasePlatformTrait: NewBasePlatformTrait("builder", 600),
		BuilderTrait: traitv1.BuilderTrait{
			Tasks: []string{"task1;my-task;do-something-else"},
		},
	}
	assert.False(t, t1.Matches(&t2))
	assert.False(t, t2.Matches(&t3))
}

func TestBuilderTraitPlatforms(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.ImagePlatforms = []string{"linux/amd64", "linux/arm64"}
	err := builderTrait.Apply(env)
	require.NoError(t, err)

	assert.Equal(t, []string{"linux/amd64", "linux/arm64"}, env.Pipeline[2].Jib.Configuration.ImagePlatforms)
}

func TestBuilderTraitOrderStrategy(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.OrderStrategy = "fifo"
	err := builderTrait.Apply(env)
	require.NoError(t, err)

	assert.Equal(t, v1.BuildOrderStrategyFIFO, env.Pipeline[0].Builder.Configuration.OrderStrategy)
}
