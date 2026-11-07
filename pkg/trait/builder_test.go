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
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestBuilderTraitNotAppliedBecauseOfNilKit(t *testing.T) {
	env := createBuilderTestEnv()
	env.IntegrationKit = nil
	trait, _ := newBuilderTrait().(*builderTrait)
	configure, conditions, err := trait.Configure(env)
	assert.False(t, configure)
	assert.Empty(t, conditions)
	require.NoError(t, err)
	assert.Nil(t, env.GetTrait("builder"))
	assert.Empty(t, env.Pipeline)
}

func TestBuilderTraitNotAppliedBecauseOfNilPhase(t *testing.T) {
	env := createBuilderTestEnv()
	env.IntegrationKit.Status.Phase = v1.IntegrationKitPhaseInitialization
	conditions, traits, err := NewBuilderTestCatalog().apply(env)
	require.NoError(t, err)
	assert.NotEmpty(t, conditions)
	assert.Empty(t, traits)
	assert.NotEmpty(t, env.ExecutedTraits)
	assert.Nil(t, env.GetTrait("builder"))
	assert.Empty(t, env.Pipeline)
}

func TestJibBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv()
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
	assert.NotNil(t, env.Pipeline[2].Jib)
	assert.Equal(t, "root-jdk-image", env.Pipeline[2].Jib.BaseImage)
	assert.NotEmpty(t, env.Pipeline[2].Jib.Registry)
}

func createBuilderTestEnv() *Environment {
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
		Platform: platform.Platform{
			Registry:            v1.RegistrySpec{Address: "registry"},
			BuildRuntimeVersion: defaults.DefaultRuntimeVersion,
			BuildBaseImage:      "root-jdk-image",
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}

	return res
}

func NewBuilderTestCatalog() *Catalog {
	return NewCatalog(nil)
}

func TestMavenPropertyBuilderTrait(t *testing.T) {
	env := createBuilderTestEnv()
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

func TestExistsTaskRequest(t *testing.T) {
	tasks := []string{"quarkus-native:1000m", "builder:1000m", "shouldfail"}
	assert.True(t, existsTaskRequest(tasks, "quarkus-native"))
	assert.False(t, existsTaskRequest(tasks, "quarkus"))
	assert.False(t, existsTaskRequest(tasks, "shouldfail"))
}

func TestBuilderWithNoNodeSelector(t *testing.T) {
	env := createBuilderTestEnv()
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
	env := createBuilderTestEnv()
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
	env := createBuilderTestEnv()
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
	env := createBuilderTestEnv()
	builderTrait := createNominalBuilderTraitTest()

	err := builderTrait.Apply(env)
	require.NoError(t, err)

	pipelineTasks := tasksByName(env.Pipeline)
	assert.Equal(t, []string{"builder", "package", "jib"}, pipelineTasks)
}

func TestBuilderTasksNodeSelectorPlatformBuildStrategyPod(t *testing.T) {
	env := createBuilderTestEnv()
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
		if t.Jib != nil {
			pipelineTasks[i] = t.Jib.Name
		}
	}
	return pipelineTasks
}

func TestBuilderMatches(t *testing.T) {
	t1 := builderTrait{
		BasePlatformTrait: NewBasePlatformTrait("builder", 600),
	}
	t2 := builderTrait{
		BasePlatformTrait: NewBasePlatformTrait("builder", 600),
	}
	assert.True(t, t1.Matches(&t2))
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

func TestBuilderTraitPlatforms(t *testing.T) {
	env := createBuilderTestEnv()
	builderTrait := createNominalBuilderTraitTest()
	builderTrait.ImagePlatforms = []string{"linux/amd64", "linux/arm64"}
	err := builderTrait.Apply(env)
	require.NoError(t, err)

	assert.Equal(t, []string{"linux/amd64", "linux/arm64"}, env.Pipeline[2].Jib.Configuration.ImagePlatforms)
}

// TestFilterNodeSelector_NoAllowList verifies that when BUILDER_NODE_SELECTOR_ALLOWED_LABELS is
// not set, all node-selector keys pass through unchanged.
func TestFilterNodeSelector_NoAllowList(t *testing.T) {
	bt := createNominalBuilderTraitTest()
	bt.NodeSelector = map[string]string{
		"kubernetes.io/hostname":         "node-1",
		"node-role.kubernetes.io/worker": "true",
	}

	// env var not set – nil allow list means everything is allowed
	result := bt.filterNodeSelector()
	assert.Equal(t, bt.NodeSelector, result)
}

// TestFilterNodeSelector_AllowListFiltersKeys verifies that only keys present in the allow list
// survive when BUILDER_NODE_SELECTOR_ALLOWED_LABELS is set.
func TestFilterNodeSelector_AllowListFiltersKeys(t *testing.T) {
	t.Setenv("BUILDER_NODE_SELECTOR_ALLOWED_LABELS", "kubernetes.io/hostname")

	bt := createNominalBuilderTraitTest()
	bt.NodeSelector = map[string]string{
		"kubernetes.io/hostname":         "node-1",
		"node-role.kubernetes.io/worker": "true", // not in allow list – must be dropped
	}

	result := bt.filterNodeSelector()
	assert.Len(t, result, 1)
	assert.Equal(t, "node-1", result["kubernetes.io/hostname"])
	assert.NotContains(t, result, "node-role.kubernetes.io/worker")
}

// TestFilterNodeSelector_AllowListAllKeysAllowed verifies that when every key is in the allow list
// the map is returned intact.
func TestFilterNodeSelector_AllowListAllKeysAllowed(t *testing.T) {
	t.Setenv("BUILDER_NODE_SELECTOR_ALLOWED_LABELS", "kubernetes.io/hostname,node-role.kubernetes.io/worker")

	bt := createNominalBuilderTraitTest()
	bt.NodeSelector = map[string]string{
		"kubernetes.io/hostname":         "node-1",
		"node-role.kubernetes.io/worker": "true",
	}

	result := bt.filterNodeSelector()
	assert.Equal(t, bt.NodeSelector, result)
}

// TestFilterNodeSelector_NilNodeSelector verifies that a nil NodeSelector is returned as-is.
func TestFilterNodeSelector_NilNodeSelector(t *testing.T) {
	t.Setenv("BUILDER_NODE_SELECTOR_ALLOWED_LABELS", "kubernetes.io/hostname")

	bt := createNominalBuilderTraitTest()
	bt.NodeSelector = nil

	result := bt.filterNodeSelector()
	assert.Nil(t, result)
}

// TestFilterNodeSelector_EmptyNodeSelector verifies that an empty NodeSelector is returned as-is.
func TestFilterNodeSelector_EmptyNodeSelector(t *testing.T) {
	t.Setenv("BUILDER_NODE_SELECTOR_ALLOWED_LABELS", "kubernetes.io/hostname")

	bt := createNominalBuilderTraitTest()
	bt.NodeSelector = map[string]string{}

	result := bt.filterNodeSelector()
	assert.Empty(t, result)
}

// TestFilterNodeSelector_AllKeysDropped verifies that when no key matches the allow list the
// result is an empty map (not nil).
func TestFilterNodeSelector_AllKeysDropped(t *testing.T) {
	t.Setenv("BUILDER_NODE_SELECTOR_ALLOWED_LABELS", "kubernetes.io/hostname")

	bt := createNominalBuilderTraitTest()
	bt.NodeSelector = map[string]string{
		"node-role.kubernetes.io/worker": "true",
		"topology.kubernetes.io/zone":    "us-east-1a",
	}

	result := bt.filterNodeSelector()
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

// TestBuilderTraitNodeSelectorAppliedWithAllowList is an end-to-end test that verifies the
// pipeline builder task only contains allowed node-selector keys after Apply().
func TestBuilderTraitNodeSelectorAppliedWithAllowList(t *testing.T) {
	t.Setenv("BUILDER_NODE_SELECTOR_ALLOWED_LABELS", "kubernetes.io/hostname")

	env := createBuilderTestEnv()
	bt := createNominalBuilderTraitTest()
	bt.NodeSelector = map[string]string{
		"kubernetes.io/hostname":         "node-42",
		"node-role.kubernetes.io/worker": "true",
	}

	err := bt.Apply(env)
	require.NoError(t, err)
	require.NotNil(t, env.Pipeline[0].Builder)

	ns := env.Pipeline[0].Builder.Configuration.NodeSelector
	assert.Len(t, ns, 1)
	assert.Equal(t, "node-42", ns["kubernetes.io/hostname"])
	assert.NotContains(t, ns, "node-role.kubernetes.io/worker")
}

func TestGetImageNameOk(t *testing.T) {
	env := Environment{
		Platform: platform.Platform{
			Registry: v1.RegistrySpec{
				Address: "1.2.3.4",
			},
			CatalogNamespace: "camel-k",
		},
		Integration: &v1.Integration{
			Name:            "my-it",
			ResourceVersion: "987",
		},
	}
	imageName, err := getImageName(&env)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3.4/camel-k/camel-k-my-it:987", imageName)
}

func TestGetImageNameFail(t *testing.T) {
	env := Environment{
		Platform: platform.Platform{},
		Integration: &v1.Integration{
			Name:            "my-it",
			ResourceVersion: "987",
		},
	}
	_, err := getImageName(&env)
	require.Error(t, err)
	assert.Equal(t, "missing registry address", err.Error())
}
