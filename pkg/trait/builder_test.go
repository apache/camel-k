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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/test"
)

func TestBuilderTraitNotAppliedBecauseOfNilKit(t *testing.T) {
	environments := []*Environment{
		createBuilderTestEnv(v1.IntegrationPlatformClusterOpenShift, v1.IntegrationPlatformBuildPublishStrategyS2I, v1.BuildStrategyRoutine),
		createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyKaniko, v1.BuildStrategyRoutine),
	}

	for _, e := range environments {
		e := e // pin
		e.IntegrationKit = nil

		t.Run(string(e.Platform.Status.Cluster), func(t *testing.T) {
			conditions, err := NewBuilderTestCatalog().apply(e)

			assert.Nil(t, err)
			assert.Empty(t, conditions)
			assert.NotEmpty(t, e.ExecutedTraits)
			assert.Nil(t, e.GetTrait("builder"))
			assert.Empty(t, e.Pipeline)
		})
	}
}

func TestBuilderTraitNotAppliedBecauseOfNilPhase(t *testing.T) {
	environments := []*Environment{
		createBuilderTestEnv(v1.IntegrationPlatformClusterOpenShift, v1.IntegrationPlatformBuildPublishStrategyS2I, v1.BuildStrategyRoutine),
		createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyKaniko, v1.BuildStrategyRoutine),
	}

	for _, e := range environments {
		e := e // pin
		e.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseInitialization

		t.Run(string(e.Platform.Status.Cluster), func(t *testing.T) {
			conditions, err := NewBuilderTestCatalog().apply(e)

			assert.Nil(t, err)
			assert.Empty(t, conditions)
			assert.NotEmpty(t, e.ExecutedTraits)
			assert.Nil(t, e.GetTrait("builder"))
			assert.Empty(t, e.Pipeline)
		})
	}
}

func TestS2IBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterOpenShift, v1.IntegrationPlatformBuildPublishStrategyS2I, v1.BuildStrategyRoutine)
	conditions, err := NewBuilderTestCatalog().apply(env)

	assert.Nil(t, err)
	assert.Empty(t, conditions)
	assert.NotEmpty(t, env.ExecutedTraits)
	assert.NotNil(t, env.GetTrait("builder"))
	assert.NotEmpty(t, env.Pipeline)
	assert.Len(t, env.Pipeline, 3)
	assert.NotNil(t, env.Pipeline[0].Builder)
	assert.NotNil(t, env.Pipeline[1].Package)
	assert.NotNil(t, env.Pipeline[2].S2i)
}

func TestKanikoBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyKaniko, v1.BuildStrategyRoutine)
	conditions, err := NewBuilderTestCatalog().apply(env)

	assert.Nil(t, err)
	assert.Empty(t, conditions)
	assert.NotEmpty(t, env.ExecutedTraits)
	assert.NotNil(t, env.GetTrait("builder"))
	assert.NotEmpty(t, env.Pipeline)
	assert.Len(t, env.Pipeline, 3)
	assert.NotNil(t, env.Pipeline[0].Builder)
	assert.NotNil(t, env.Pipeline[1].Package)
	assert.NotNil(t, env.Pipeline[2].Kaniko)
}

func createBuilderTestEnv(cluster v1.IntegrationPlatformCluster, strategy v1.IntegrationPlatformBuildPublishStrategy, buildStrategy v1.BuildStrategy) *Environment {
	c, err := camel.DefaultCatalog()
	if err != nil {
		panic(err)
	}
	client, _ := test.NewFakeClient()
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
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseBuildSubmitted,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-kit",
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: cluster,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy:        strategy,
					Registry:               v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:         defaults.DefaultRuntimeVersion,
					RuntimeProvider:        v1.RuntimeProviderQuarkus,
					PublishStrategyOptions: map[string]string{},
					BuildConfiguration: v1.BuildConfiguration{
						Strategy: buildStrategy,
					},
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
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyKaniko, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Properties = append(builderTrait.Properties, "build-time-prop1=build-time-value1")

	err := builderTrait.Apply(env)

	assert.Nil(t, err)
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

	assert.Nil(t, err)
	builderTask := findCustomTaskByName(env.Pipeline, "builder")
	customTask := findCustomTaskByName(env.Pipeline, "test")
	packageTask := findCustomTaskByName(env.Pipeline, "package")
	publisherTask := findCustomTaskByName(env.Pipeline, "spectrum")

	assert.NotNil(t, builderTask)
	assert.NotNil(t, customTask)
	assert.NotNil(t, packageTask)
	assert.NotNil(t, publisherTask)
	assert.Equal(t, 4, len(env.Pipeline))
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

	assert.Nil(t, err)

	customTask := findCustomTaskByName(env.Pipeline, "test")

	assert.Equal(t, 4, len(env.Pipeline))
	assert.Equal(t, "test", customTask.Custom.Name)
	assert.Equal(t, "alpine", customTask.Custom.ContainerImage)
	assert.Equal(t, "ls", customTask.Custom.ContainerCommands[0])
}

func TestCustomTaskBuilderTraitInvalidStrategy(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategySpectrum, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, "test;alpine;ls")

	err := builderTrait.Apply(env)

	assert.NotNil(t, err)
	assert.Equal(t, env.IntegrationKit.Status.Phase, v1.IntegrationKitPhaseError)
	assert.Equal(t, env.IntegrationKit.Status.Conditions[0].Status, corev1.ConditionFalse)
	assert.Equal(t, env.IntegrationKit.Status.Conditions[0].Type, v1.IntegrationKitConditionType("IntegrationKitTasksValid"))
}

func TestCustomTaskBuilderTraitInvalidStrategyOverride(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategySpectrum, v1.BuildStrategyPod)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, "test;alpine;ls")
	builderTrait.Strategy = "routine"

	err := builderTrait.Apply(env)

	assert.NotNil(t, err)
	assert.Equal(t, env.IntegrationKit.Status.Phase, v1.IntegrationKitPhaseError)
	assert.Equal(t, env.IntegrationKit.Status.Conditions[0].Status, corev1.ConditionFalse)
	assert.Equal(t, env.IntegrationKit.Status.Conditions[0].Type, v1.IntegrationKitConditionType("IntegrationKitTasksValid"))
}

func findCustomTaskByName(tasks []v1.Task, name string) v1.Task {
	for _, t := range tasks {
		if t.Custom != nil && t.Custom.Name == name {
			return t
		}
	}
	return v1.Task{}
}

func TestMavenProfilesBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyKaniko, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.MavenProfiles = []string{"configmap:maven-profile/owasp-profile.xml", "secret:maven-profile-secret"}

	err := builderTrait.Apply(env)

	assert.Nil(t, err)

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
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyKaniko, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.MavenProfiles = []string{"fakeprofile"}

	err := builderTrait.Apply(env)

	assert.NotNil(t, err)
	assert.Equal(t, env.IntegrationKit.Status.Phase, v1.IntegrationKitPhaseError)
	assert.Equal(t, env.IntegrationKit.Status.Conditions[0].Status, corev1.ConditionFalse)
	assert.Contains(t, env.IntegrationKit.Status.Conditions[0].Message, "fakeprofile")
}

func TestMavenBuilderTraitJib(t *testing.T) {
	env := createBuilderTestEnv(v1.IntegrationPlatformClusterKubernetes, v1.IntegrationPlatformBuildPublishStrategyJib, v1.BuildStrategyRoutine)
	builderTrait := createNominalBuilderTraitTest()

	err := builderTrait.Apply(env)

	assert.Nil(t, err)

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

	tasks, err := builderTrait.customTasks(nil)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(tasks))
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

	_, err := builderTrait.customTasks(nil)

	assert.NotNil(t, err)
}

func TestBuilderCustomTasksScript(t *testing.T) {
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.Tasks = append(builderTrait.Tasks, "test;alpine;/bin/bash -c \"cd test && ls; echo 'helooo'\"")

	tasks, err := builderTrait.customTasks(nil)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(tasks))
	assert.Equal(t, "test", tasks[0].Custom.Name)
	assert.Equal(t, "alpine", tasks[0].Custom.ContainerImage)
	assert.Equal(t, "/bin/bash", tasks[0].Custom.ContainerCommands[0])
	assert.Equal(t, "-c", tasks[0].Custom.ContainerCommands[1])
	assert.Equal(t, "cd test && ls; echo 'helooo'", tasks[0].Custom.ContainerCommands[2])
}

func TestBuilderCustomTasksConfiguration(t *testing.T) {
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.TasksRequestCPU = append(builderTrait.TasksRequestCPU, "builder:1000m")
	builderTrait.TasksLimitCPU = append(builderTrait.TasksLimitCPU, "custom1:500m")
	builderTrait.TasksRequestMemory = append(builderTrait.TasksRequestMemory, "package:8Gi")
	builderTrait.TasksLimitMemory = append(builderTrait.TasksLimitMemory, "spectrum:4Gi")

	tasksConf, err := builderTrait.parseTasksConf()

	assert.Nil(t, err)
	assert.Equal(t, 4, len(tasksConf))
	assert.Equal(t, "1000m", tasksConf["builder"].RequestCPU)
	assert.Equal(t, "500m", tasksConf["custom1"].LimitCPU)
	assert.Equal(t, "8Gi", tasksConf["package"].RequestMemory)
	assert.Equal(t, "4Gi", tasksConf["spectrum"].LimitMemory)
}

func TestBuilderCustomTasksConfigurationError(t *testing.T) {
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.TasksLimitCPU = append(builderTrait.TasksLimitCPU, "syntax error")

	_, err := builderTrait.parseTasksConf()

	assert.NotNil(t, err)
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

	assert.Nil(t, err)
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
