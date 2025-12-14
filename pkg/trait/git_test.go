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
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitApplyMissingTasks(t *testing.T) {
	it := v1.NewIntegration("default", "test")
	it.Spec.Git = &v1.GitConfigSpec{}
	it.Status = v1.IntegrationStatus{
		Phase: v1.IntegrationPhaseBuildSubmitted,
	}
	e := &Environment{
		Integration: &it,
	}
	g := gitTrait{}

	ok, _, err := g.Configure(e)
	require.NoError(t, err)
	assert.True(t, ok)
	err = g.Apply(e)
	require.Error(t, err)
	assert.Equal(t, "unable to find builder task: test", err.Error())

	e.Pipeline = []v1.Task{
		v1.Task{Builder: &v1.BuilderTask{}},
	}

	err = g.Apply(e)
	require.Error(t, err)
	assert.Equal(t, "unable to find package task: test", err.Error())
}

func TestGitApplyOk(t *testing.T) {
	it := v1.NewIntegration("default", "test")
	it.Spec.Git = &v1.GitConfigSpec{}
	it.Status = v1.IntegrationStatus{
		Phase: v1.IntegrationPhaseBuildSubmitted,
	}
	e := &Environment{
		Integration: &it,
		Pipeline: []v1.Task{
			v1.Task{Builder: &v1.BuilderTask{}},
			v1.Task{Package: &v1.BuilderTask{}},
		},
	}
	g := gitTrait{}

	ok, _, err := g.Configure(e)
	assert.True(t, ok)
	require.NoError(t, err)
	err = g.Apply(e)
	require.NoError(t, err)

	buildTask := getBuilderTask(e.Pipeline)
	require.NotNil(t, buildTask)
	packageTask := getPackageTask(e.Pipeline)
	require.NotNil(t, packageTask)

	assert.Contains(t, buildTask.Steps,
		"github.com/apache/camel-k/v2/pkg/builder/CloneProject",
		"github.com/apache/camel-k/v2/pkg/builder/InjectJibProfile",
		"github.com/apache/camel-k/v2/pkg/builder/BuildMavenContext",
		"github.com/apache/camel-k/v2/pkg/builder/ExecuteMavenContext",
		"github.com/apache/camel-k/v2/pkg/builder/ComputeDependencies",
	)

	assert.Contains(t, packageTask.Steps,
		"github.com/apache/camel-k/v2/pkg/builder/ComputeDependencies",
		"github.com/apache/camel-k/v2/pkg/builder/StandardImageContext",
		"github.com/apache/camel-k/v2/pkg/builder/JvmDockerfile",
	)
}
