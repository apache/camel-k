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

package integrationplatform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/maven"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreate(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Status = v1.IntegrationPlatformStatus{
		IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				RuntimeProvider: v1.RuntimeProviderQuarkus,
				RuntimeVersion:  defaults.DefaultRuntimeVersion,
			},
		},
	}
	c, err := test.NewFakeClient(&ip)
	require.NoError(t, err)

	h := NewCreateAction()
	h.InjectLogger(log.Log)
	h.InjectClient(c)
	// We don't want to test the installation procedure here
	os.Setenv("KAMEL_INSTALL_DEFAULT_KAMELETS", "false")
	answer, err := h.Handle(context.TODO(), &ip)

	require.NoError(t, err)
	assert.NotNil(t, answer)
	assert.Equal(t, v1.IntegrationPlatformPhaseReady, answer.Status.Phase)
	assert.Equal(t, defaults.DefaultRuntimeVersion, answer.Status.Build.RuntimeVersion)
	assert.Equal(t, v1.RuntimeProviderQuarkus, answer.Status.Build.RuntimeProvider)
	assert.NotEqual(t, "", answer.Status.Build.RuntimeCoreVersion)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Status)

	list := v1.NewCamelCatalogList()
	err = c.List(context.TODO(), &list, k8sclient.InNamespace(ip.Namespace))
	require.NoError(t, err)
	assert.NotEmpty(t, list.Items)
}

func TestCatalogAlreadyPresent(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Status = v1.IntegrationPlatformStatus{
		IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				RuntimeProvider: v1.RuntimeProviderQuarkus,
				RuntimeVersion:  defaults.DefaultRuntimeVersion,
			},
		},
	}

	catalog := v1.NewCamelCatalog("ns", fmt.Sprintf("camel-catalog-%s", defaults.DefaultRuntimeVersion))
	catalog.Spec.Runtime.Version = defaults.DefaultRuntimeVersion
	catalog.Spec.Runtime.Provider = v1.RuntimeProviderQuarkus
	catalog.Spec.Runtime.Metadata = map[string]string{
		"camel.version": "4.4.0",
	}

	c, err := test.NewFakeClient(&ip, &catalog)
	require.NoError(t, err)

	action := NewCreateAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	// We don't want to test the installation procedure here
	os.Setenv("KAMEL_INSTALL_DEFAULT_KAMELETS", "false")
	answer, err := action.Handle(context.TODO(), &ip)
	os.Unsetenv("KAMEL_INSTALL_DEFAULT_KAMELETS")
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseReady, answer.Status.Phase)
	assert.Equal(t, "4.4.0", answer.Status.Build.RuntimeCoreVersion)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Status)
}

func TestCreateNewCatalog(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Status = v1.IntegrationPlatformStatus{
		IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				RuntimeProvider: v1.RuntimeProviderQuarkus,
				RuntimeVersion:  defaults.DefaultRuntimeVersion,
			},
		},
	}
	if strings.Contains(ip.Spec.Build.RuntimeVersion, "SNAPSHOT") {
		maven.DefaultMavenRepositories += ",https://repository.apache.org/content/repositories/snapshots-group@snapshots@id=apache-snapshots"
	}

	c, err := test.NewFakeClient(&ip)
	require.NoError(t, err)

	// use local Maven executable in tests
	t.Setenv("MAVEN_WRAPPER", boolean.FalseString)
	_, ok := os.LookupEnv("MAVEN_CMD")
	if !ok {
		t.Setenv("MAVEN_CMD", "mvn")
	}

	fakeClient := c.(*test.FakeClient) //nolint
	fakeClient.AddReactor("create", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction) //nolint

		assert.Equal(t, "ns", createAction.GetNamespace())

		return true, createAction.GetObject(), nil
	})

	err = platform.ConfigureDefaults(context.TODO(), c, &ip, false)
	require.NoError(t, err)

	action := NewCreateAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	// Set the folder where to install testing kamelets
	tmpDir, err := os.MkdirTemp("/tmp", "kamelets*")
	assert.NoError(t, err)
	os.Setenv(kameletDirEnv, tmpDir)
	answer, err := action.Handle(context.TODO(), &ip)
	os.Unsetenv(kameletDirEnv)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseReady, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Status)
	// We don't know exactly which is the core version, it is enough to check is not empty in the test
	assert.NotEqual(t, "", answer.Status.Build.RuntimeCoreVersion)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionKameletCatalogAvailable).Status)
	assert.Contains(t, answer.Status.GetCondition(v1.IntegrationPlatformConditionKameletCatalogAvailable).Message,
		fmt.Sprintf("successfully installed Kamelet catalog version %s", answer.Status.Build.RuntimeCoreVersion),
		"failed 0 Kamelets")

	list := v1.NewCamelCatalogList()
	err = c.List(context.TODO(), &list, k8sclient.InNamespace(ip.Namespace))

	require.NoError(t, err)
	assert.NotEmpty(t, list.Items)
}

func TestCreateCatalogError(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Spec.Build.RuntimeVersion = "0.0.0"
	c, err := test.NewFakeClient(&ip)
	require.NoError(t, err)

	// use local Maven executable in tests
	t.Setenv("MAVEN_WRAPPER", boolean.FalseString)
	_, ok := os.LookupEnv("MAVEN_CMD")
	if !ok {
		t.Setenv("MAVEN_CMD", "mvn")
	}

	fakeClient := c.(*test.FakeClient) //nolint
	fakeClient.AddReactor("create", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction) //nolint

		assert.Equal(t, "ns", createAction.GetNamespace())

		return true, nil, errors.New("failed to create catalog for some reason")
	})

	err = platform.ConfigureDefaults(context.TODO(), c, &ip, false)
	require.NoError(t, err)

	action := NewCreateAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	require.Error(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseError, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Status)
	assert.Equal(t, v1.IntegrationPlatformConditionCamelCatalogAvailableReason, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Reason)
	assert.Contains(t, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Message, "camel catalog 0.0.0 not available, please review given runtime version. Error:")
}
