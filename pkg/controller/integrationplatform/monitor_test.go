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
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
)

func TestCanHandlePhaseReadyOrError(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift
	ip.Spec.Build.Registry.Address = defaults.OpenShiftRegistryAddress

	ip.Spec.Build.RuntimeVersion = defaults.DefaultRuntimeVersion

	ip.Status.Phase = v1.IntegrationPlatformPhaseReady

	c, err := test.NewFakeClient(&ip)
	assert.Nil(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer := action.CanHandle(&ip)
	assert.True(t, answer)

	ip.Status.Phase = v1.IntegrationPlatformPhaseError
	answer = action.CanHandle(&ip)
	assert.True(t, answer)

	ip.Status.Phase = v1.IntegrationPlatformPhaseCreateCatalog
	answer = action.CanHandle(&ip)
	assert.False(t, answer)
}

func TestMonitor(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift
	ip.Spec.Build.Registry.Address = defaults.OpenShiftRegistryAddress

	catalog := v1.NewCamelCatalog("ns", fmt.Sprintf("camel-catalog-%s", defaults.DefaultRuntimeVersion))
	catalog.Spec.Runtime.Version = defaults.DefaultRuntimeVersion
	catalog.Spec.Runtime.Provider = v1.RuntimeProviderQuarkus

	c, err := test.NewFakeClient(&ip, &catalog)
	assert.Nil(t, err)

	err = platform.ConfigureDefaults(context.TODO(), c, &ip, false)
	assert.Nil(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	assert.Nil(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseReady, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Status)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Status)
}

func TestMonitorTransitionToCreateCatalog(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift
	ip.Spec.Build.Registry.Address = defaults.OpenShiftRegistryAddress

	ip.Spec.Build.RuntimeVersion = defaults.DefaultRuntimeVersion

	c, err := test.NewFakeClient(&ip)
	assert.Nil(t, err)

	err = platform.ConfigureDefaults(context.TODO(), c, &ip, false)
	assert.Nil(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	assert.Nil(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseCreateCatalog, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Status)
	assert.Equal(t, corev1.ConditionFalse, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Status)
	assert.Equal(t, v1.IntegrationPlatformConditionCamelCatalogAvailableReason, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Reason)
	assert.Equal(t, fmt.Sprintf("camel catalog %s not available, please review given runtime version", defaults.DefaultRuntimeVersion), answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Message)
}

func TestMonitorRetainErrorState(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift
	ip.Spec.Build.Registry.Address = defaults.OpenShiftRegistryAddress

	ip.Spec.Build.RuntimeVersion = defaults.DefaultRuntimeVersion

	ip.Status.Phase = v1.IntegrationPlatformPhaseError

	c, err := test.NewFakeClient(&ip)
	assert.Nil(t, err)

	err = platform.ConfigureDefaults(context.TODO(), c, &ip, false)
	assert.Nil(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	assert.Nil(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseError, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionTrue, answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Status)
	assert.Equal(t, corev1.ConditionFalse, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Status)
	assert.Equal(t, v1.IntegrationPlatformConditionCamelCatalogAvailableReason, answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Reason)
	assert.Equal(t, fmt.Sprintf("camel catalog %s not available, please review given runtime version", defaults.DefaultRuntimeVersion), answer.Status.GetCondition(v1.IntegrationPlatformConditionCamelCatalogAvailable).Message)
}

func TestMonitorMissingRegistryError(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift

	catalog := v1.NewCamelCatalog("ns", fmt.Sprintf("camel-catalog-%s", defaults.DefaultRuntimeVersion))
	catalog.Spec.Runtime.Version = defaults.DefaultRuntimeVersion
	catalog.Spec.Runtime.Provider = v1.RuntimeProviderQuarkus

	c, err := test.NewFakeClient(&ip, &catalog)
	assert.Nil(t, err)

	err = platform.ConfigureDefaults(context.TODO(), c, &ip, false)
	assert.Nil(t, err)

	action := NewMonitorAction()
	action.InjectLogger(log.Log)
	action.InjectClient(c)

	answer, err := action.Handle(context.TODO(), &ip)
	assert.Nil(t, err)
	assert.NotNil(t, answer)

	assert.Equal(t, v1.IntegrationPlatformPhaseError, answer.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Status)
	assert.Equal(t, v1.IntegrationPlatformConditionTypeRegistryAvailableReason, answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Reason)
	assert.Equal(t, "registry address not available, you need to set one", answer.Status.GetCondition(v1.IntegrationPlatformConditionTypeRegistryAvailable).Message)
}
