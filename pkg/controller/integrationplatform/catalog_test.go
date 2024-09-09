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

	"github.com/apache/camel-k/v2/pkg/util/boolean"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/resources"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/maven"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCanHandlePhaseCreateCatalog(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift
	ip.Spec.Build.Registry.Address = defaults.OpenShiftRegistryAddress

	ip.Spec.Build.RuntimeVersion = defaults.DefaultRuntimeVersion

	ip.Status.Phase = v1.IntegrationPlatformPhaseCreateCatalog

	c, err := test.NewFakeClient(&ip)
	require.NoError(t, err)

	action := NewCreateCatalogAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer := action.CanHandle(&ip)
	assert.True(t, answer)

	ip.Status.Phase = v1.IntegrationPlatformPhaseError
	answer = action.CanHandle(&ip)
	assert.False(t, answer)

	ip.Status.Phase = v1.IntegrationPlatformPhaseReady
	answer = action.CanHandle(&ip)
	assert.False(t, answer)
}

func TestCreateCatalog(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift
	ip.Spec.Build.Registry.Address = defaults.OpenShiftRegistryAddress

	ip.Status.Phase = v1.IntegrationPlatformPhaseCreateCatalog
	ip.Spec.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
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

	action := NewCreateCatalogAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseReady, answer.Status.Phase, "Error", answer.Status.Conditions[0].Message)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Status)
	// We don't know exactly which is the core version, it is enough to check is not empty in the test
	assert.NotEqual(t, "", answer.Status.Build.RuntimeCoreVersion)

	list := v1.NewCamelCatalogList()
	err = c.List(context.TODO(), &list, k8sclient.InNamespace(ip.Namespace))

	require.NoError(t, err)
	assert.NotEmpty(t, list.Items)

	items, err := resources.WithPrefix("/camel-catelog-")
	require.NoError(t, err)

	for _, k := range items {
		found := false

		for _, c := range list.Items {
			n := strings.TrimSuffix(k, ".yaml")
			n = strings.ToLower(n)

			if c.Name == n {
				found = true
			}
		}

		assert.True(t, found)
	}
}

func TestCatalogAlreadyPresent(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift
	ip.Spec.Build.Registry.Address = defaults.OpenShiftRegistryAddress

	ip.Status.Phase = v1.IntegrationPlatformPhaseCreateCatalog

	catalog := v1.NewCamelCatalog("ns", fmt.Sprintf("camel-catalog-%s", defaults.DefaultRuntimeVersion))
	catalog.Spec.Runtime.Version = defaults.DefaultRuntimeVersion
	catalog.Spec.Runtime.Provider = v1.RuntimeProviderQuarkus
	catalog.Spec.Runtime.Metadata = map[string]string{
		"camel.version": "4.4.0",
	}

	c, err := test.NewFakeClient(&ip, &catalog)
	require.NoError(t, err)

	err = platform.ConfigureDefaults(context.TODO(), c, &ip, false)
	require.NoError(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseReady, answer.Status.Phase)
	assert.Equal(t, "4.4.0", answer.Status.Build.RuntimeCoreVersion)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Status)
}

func TestCreateCatalogError(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift
	ip.Spec.Build.Registry.Address = defaults.OpenShiftRegistryAddress

	ip.Status.Phase = v1.IntegrationPlatformPhaseCreateCatalog

	// force catalog build to fail
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

	action := NewCreateCatalogAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseError, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Status)
	assert.Equal(t, v1.IntegrationPlatformConditionCamelCatalogAvailableReason, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Reason)
	assert.Contains(t, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Message, "camel catalog 0.0.0 not available, please review given runtime version. Error:")
}
