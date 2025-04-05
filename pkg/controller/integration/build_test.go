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

package integration

import (
	"context"
	"fmt"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIntegrationBuildSubmitted(t *testing.T) {
	it := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-it",
		},
		Spec: v1.IntegrationSpec{
			Git: &v1.GitConfigSpec{
				URL: "missing",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseBuildSubmitted,
		},
	}
	platform := &v1.IntegrationPlatform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationPlatformKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "camel-k",
		},
		Status: v1.IntegrationPlatformStatus{
			Phase: v1.IntegrationPlatformPhaseReady,
			IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeVersion: defaults.DefaultRuntimeVersion,
				},
			},
		},
	}
	catalog := &v1.CamelCatalog{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.CamelCatalogKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "camel-k-catalog",
		},
		Spec: v1.CamelCatalogSpec{
			Runtime: v1.RuntimeSpec{
				Provider: v1.RuntimeProviderQuarkus,
				Version:  defaults.DefaultRuntimeVersion,
			},
		},
	}
	c, err := internal.NewFakeClient(it, platform, catalog)
	require.NoError(t, err)

	a := buildAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "build", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationPhaseBuildRunning, handledIt.Status.Phase)
	require.NotNil(t, handledIt.Status.Conditions[0], "The Integration must have a CamelCatalog warning condition")
	assert.Equal(t,
		fmt.Sprintf(
			"Operated with CamelCatalog version %s which may be different from the runtime used in the container",
			defaults.DefaultRuntimeVersion),
		handledIt.Status.Conditions[0].Message)

	// Verify the downstream build created
	expectedBuild := &v1.Build{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.BuildKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-it",
		},
	}
	err = c.Get(context.Background(), ctrl.ObjectKeyFromObject(expectedBuild), expectedBuild)
	require.NoError(t, err)
	require.NotNil(t, expectedBuild.OwnerReferences[0], "The Build should have an Integration owner reference")
	assert.Equal(t, "Integration", expectedBuild.OwnerReferences[0].Kind)
	assert.Equal(t, "my-it", expectedBuild.OwnerReferences[0].Name)
}

func TestIntegrationBuildRunningBuildMissing(t *testing.T) {
	it := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-it",
		},
		Spec: v1.IntegrationSpec{
			Git: &v1.GitConfigSpec{
				URL: "missing",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseBuildRunning,
		},
	}
	c, err := internal.NewFakeClient(it)
	require.NoError(t, err)

	a := buildAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "build", a.Name())
	assert.True(t, a.CanHandle(it))
	_, err = a.Handle(context.TODO(), it)
	require.Error(t, err)
	assert.Equal(t, "builds.camel.apache.org \"my-it\" not found", err.Error())
}

func TestIntegrationBuildRunningBuildSucceeded(t *testing.T) {
	it := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-it",
		},
		Spec: v1.IntegrationSpec{
			Git: &v1.GitConfigSpec{
				URL: "missing",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseBuildRunning,
		},
	}
	build := &v1.Build{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.BuildKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-it",
		},
		Status: v1.BuildStatus{
			Phase: v1.BuildPhaseSucceeded,
			Image: "my-image",
			Artifacts: []v1.Artifact{
				{
					Target: "/deploy/my.jar",
				},
			},
			Digest: "123456",
		},
	}
	c, err := internal.NewFakeClient(it, build)
	require.NoError(t, err)

	a := buildAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "build", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.NoError(t, err)
	require.NotNil(t, handledIt)
	assert.Equal(t, v1.IntegrationPhaseDeploying, handledIt.Status.Phase)
	assert.Equal(t, "my-image@123456", handledIt.Status.Image)
	assert.Equal(t, "/deploy/my.jar", handledIt.Status.Jar)
}

func TestIntegrationBuildRunningBuildError(t *testing.T) {
	it := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-it",
		},
		Spec: v1.IntegrationSpec{
			Git: &v1.GitConfigSpec{
				URL: "missing",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseBuildRunning,
		},
	}
	build := &v1.Build{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.BuildKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-it",
		},
		Status: v1.BuildStatus{
			Phase: v1.BuildPhaseError,
			Failure: &v1.Failure{
				Reason: "build failed",
			},
		},
	}
	c, err := internal.NewFakeClient(it, build)
	require.NoError(t, err)

	a := buildAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "build", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.NoError(t, err)
	require.NotNil(t, handledIt)
	assert.Equal(t, v1.IntegrationPhaseError, handledIt.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, "BuildError", handledIt.Status.GetCondition(v1.IntegrationConditionReady).Reason)
	assert.Equal(t, "build failed", handledIt.Status.GetCondition(v1.IntegrationConditionReady).Message)
}
