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
	"fmt"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
)

func TestCanHandlePhaseReadyOrError(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Status.Phase = v1.IntegrationPlatformPhaseReady
	c, err := internal.NewFakeClient(&ip)
	require.NoError(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer := action.CanHandle(&ip)
	assert.True(t, answer)

	ip.Status.Phase = v1.IntegrationPlatformPhaseError
	answer = action.CanHandle(&ip)
	assert.True(t, answer)

	ip.Status.Phase = v1.IntegrationPlatformPhaseCreating
	answer = action.CanHandle(&ip)
	assert.False(t, answer)
}

func TestMonitorReady(t *testing.T) {
	catalog := v1.NewCamelCatalog("ns", fmt.Sprintf("camel-catalog-%s", "1.2.3"))
	catalog.Spec.Runtime.Version = "1.2.3"
	catalog.Spec.Runtime.Provider = v1.RuntimeProviderQuarkus
	catalog.Spec.Runtime.Metadata = map[string]string{
		"camel.version": "3.2.1",
	}
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Spec.Build.Registry.Address = "1.2.3.4"
	ip.Spec.Build.RuntimeVersion = "1.2.3"
	ip.Spec.Build.RuntimeProvider = v1.RuntimeProviderQuarkus
	ip.Spec.Build.Maven.Settings.ConfigMapKeyRef = &corev1.ConfigMapKeySelector{
		Key:                  "k",
		LocalObjectReference: corev1.LocalObjectReference{Name: "v"},
	}
	ip.Status.Build.RuntimeVersion = "1.2.3"
	ip.Status.Build.RuntimeProvider = v1.RuntimeProviderQuarkus
	ip.Status.Build.Registry.Address = "1.2.3.4"
	ip.Status.Phase = v1.IntegrationPlatformPhaseReady
	c, err := internal.NewFakeClient(&ip, &catalog)
	require.NoError(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseReady, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionTrue,
		answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Status)
	assert.Nil(t, answer.Status.GetCondition(v1.IntegrationPlatformConditionType("InsecureRegistryWarning")))
	assert.Equal(t, corev1.ConditionTrue,
		answer.Status.GetCondition(v1.IntegrationPlatformConditionMavenSettingsAvailable).Status)
	assert.Equal(t, "3.2.1", answer.Status.Build.RuntimeCoreVersion)
}

func TestMonitorDriftRuntime(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Spec.Build.RuntimeVersion = "3.2.1"
	ip.Spec.Build.RuntimeProvider = v1.RuntimeProviderQuarkus
	ip.Status.Build.RuntimeVersion = "1.2.3"
	ip.Status.Build.RuntimeProvider = v1.RuntimeProviderQuarkus
	ip.Status.Build.Registry.Address = "1.2.3.4"
	ip.Status.Phase = v1.IntegrationPlatformPhaseReady
	c, err := internal.NewFakeClient(&ip)
	require.NoError(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseNone, answer.Status.Phase)
}

func TestMonitorDriftDefault(t *testing.T) {
	catalog := v1.NewCamelCatalog("ns", fmt.Sprintf("camel-catalog-%s", defaults.DefaultRuntimeVersion))
	catalog.Spec.Runtime.Version = defaults.DefaultRuntimeVersion
	catalog.Spec.Runtime.Provider = v1.RuntimeProviderQuarkus
	catalog.Spec.Runtime.Metadata = map[string]string{
		"camel.version": "3.2.1",
	}
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Spec.Build.Registry.Address = "1.2.3.4"
	ip.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	ip.Status.Build.RuntimeProvider = v1.RuntimeProviderQuarkus
	ip.Status.Build.Registry.Address = "1.2.3.4"
	ip.Status.Phase = v1.IntegrationPlatformPhaseReady
	c, err := internal.NewFakeClient(&ip, &catalog)
	require.NoError(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseReady, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse,
		answer.Status.GetCondition(v1.IntegrationPlatformConditionMavenSettingsAvailable).Status)
	assert.Equal(t, "3.2.1", answer.Status.Build.RuntimeCoreVersion)
}

func TestMonitorDriftConfiguration(t *testing.T) {
	catalog := v1.NewCamelCatalog("ns", fmt.Sprintf("camel-catalog-%s", defaults.DefaultRuntimeVersion))
	catalog.Spec.Runtime.Version = defaults.DefaultRuntimeVersion
	catalog.Spec.Runtime.Provider = v1.RuntimeProviderQuarkus
	catalog.Spec.Runtime.Metadata = map[string]string{
		"camel.version": "3.2.1",
	}
	ip := v1.IntegrationPlatform{
		Spec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				Registry: v1.RegistrySpec{
					Address: "1.2.3.4",
				},
			},
			Traits: v1.Traits{
				Container: &trait.ContainerTrait{
					Name: "override",
				},
			},
			Profile: v1.TraitProfileKnative,
		},
		Status: v1.IntegrationPlatformStatus{
			IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeVersion:     defaults.DefaultRuntimeVersion,
					RuntimeProvider:    v1.RuntimeProviderQuarkus,
					RuntimeCoreVersion: "3.2.1",
					Registry: v1.RegistrySpec{
						Address: "1.2.3.4",
					},
				},
			},
			Phase: v1.IntegrationPlatformPhaseReady,
		},
	}
	ip.Namespace = "ns"
	ip.Name = "ck"
	c, err := internal.NewFakeClient(&ip, &catalog)
	require.NoError(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	// It should not change the phase, however, it should change the values
	assert.Equal(t, v1.IntegrationPlatformPhaseReady, answer.Status.Phase)
	assert.Equal(t, defaults.Version, answer.Status.Version)
	assert.Equal(t, defaults.DefaultRuntimeVersion, answer.Status.Build.RuntimeVersion)
	assert.Equal(t, v1.RuntimeProviderQuarkus, answer.Status.Build.RuntimeProvider)
	assert.Equal(t, "3.2.1", answer.Status.Build.RuntimeCoreVersion)
	assert.Equal(t, v1.TraitProfileKnative, answer.Status.Profile)
	assert.Equal(t, "override", answer.Status.Traits.Container.Name)
}

func TestMonitorMissingRegistryError(t *testing.T) {
	catalog := v1.NewCamelCatalog("ns", fmt.Sprintf("camel-catalog-%s", "1.2.3"))
	catalog.Spec.Runtime.Version = "1.2.3"
	catalog.Spec.Runtime.Provider = v1.RuntimeProviderQuarkus
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Spec.Build.RuntimeVersion = "1.2.3"
	ip.Spec.Build.RuntimeProvider = v1.RuntimeProviderQuarkus
	ip.Status.Build.RuntimeVersion = "1.2.3"
	ip.Status.Build.RuntimeProvider = v1.RuntimeProviderQuarkus
	c, err := internal.NewFakeClient(&ip, &catalog)
	require.NoError(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseError, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Status)
	assert.Equal(t,
		v1.IntegrationPlatformConditionTypeRegistryAvailableReason,
		answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Reason)
	assert.Equal(t,
		"registry address not available, you need to set one",
		answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Message)

	// fix and see if it reconciles correctly
	ip.Spec.Build.Registry = v1.RegistrySpec{
		Address: "1.2.3.4",
	}
	answer, err = action.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseReady, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Status)
	assert.Equal(t,
		v1.IntegrationPlatformConditionTypeRegistryAvailableReason,
		answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Reason)
	assert.Equal(t,
		"registry available at 1.2.3.4",
		answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Message)
}

func TestMonitorMissingCatalogError(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Status.Build.RuntimeVersion = defaults.DefaultRuntimeVersion
	ip.Status.Build.RuntimeProvider = v1.RuntimeProviderQuarkus
	c, err := internal.NewFakeClient(&ip)
	require.NoError(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseError, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Status)
	assert.Equal(t, v1.IntegrationPlatformConditionCamelCatalogAvailableReason, answer.Status.GetCondition(
		v1.IntegrationPlatformConditionCamelCatalogAvailable).Reason)
	assert.Equal(t, fmt.Sprintf("camel catalog %s not available, please review given runtime version",
		defaults.DefaultRuntimeVersion), answer.Status.GetCondition(
		v1.IntegrationPlatformConditionCamelCatalogAvailable).Message)
}

func TestMonitorWarningInsecureRegistry(t *testing.T) {
	catalog := v1.NewCamelCatalog("ns", fmt.Sprintf("camel-catalog-%s", "1.2.3"))
	catalog.Spec.Runtime.Version = "1.2.3"
	catalog.Spec.Runtime.Provider = v1.RuntimeProviderQuarkus
	catalog.Spec.Runtime.Metadata = map[string]string{
		"camel.version": "3.2.1",
	}
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Spec.Build.Registry.Address = "1.2.3.4"
	ip.Spec.Build.Registry.Insecure = true
	ip.Spec.Build.RuntimeVersion = "1.2.3"
	ip.Spec.Build.RuntimeProvider = v1.RuntimeProviderQuarkus
	ip.Status.Build.RuntimeVersion = "1.2.3"
	ip.Status.Build.RuntimeProvider = v1.RuntimeProviderQuarkus
	ip.Status.Build.Registry.Address = "1.2.3.4"
	ip.Status.Build.Registry.Insecure = true
	ip.Status.Phase = v1.IntegrationPlatformPhaseReady
	c, err := internal.NewFakeClient(&ip, &catalog)
	require.NoError(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseReady, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Status)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionType("InsecureRegistryWarning")).Status)
	assert.Equal(t, "Registry is insecure. This setup should not be used in a production environment.",
		answer.Status.GetCondition(
			v1.IntegrationPlatformConditionType("InsecureRegistryWarning")).Message)
}
