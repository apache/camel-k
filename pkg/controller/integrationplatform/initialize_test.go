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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultRuntimeSpec(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	c, err := test.NewFakeClient(&ip)
	require.NoError(t, err)

	h := NewInitializeAction()
	h.InjectLogger(log.Log)
	h.InjectClient(c)

	answer, err := h.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationPlatformPhaseCreating, answer.Status.Phase)
	assert.Equal(t, defaults.DefaultRuntimeVersion, answer.Status.Build.RuntimeVersion)
	assert.Equal(t, v1.RuntimeProviderQuarkus, answer.Status.Build.RuntimeProvider)
}

func TestUserRuntimeSpec(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Spec = v1.IntegrationPlatformSpec{
		Build: v1.IntegrationPlatformBuildSpec{
			RuntimeVersion:  "1.2.3",
			RuntimeProvider: "MyProvider",
		},
	}
	c, err := test.NewFakeClient(&ip)
	require.NoError(t, err)

	h := NewInitializeAction()
	h.InjectLogger(log.Log)
	h.InjectClient(c)

	answer, err := h.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationPlatformPhaseCreating, answer.Status.Phase)
	assert.Equal(t, "1.2.3", answer.Status.Build.RuntimeVersion)
	assert.Equal(t, v1.RuntimeProvider("MyProvider"), answer.Status.Build.RuntimeProvider)
}

func TestDefaultTimeouts(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	c, err := test.NewFakeClient(&ip)

	require.NoError(t, err)
	require.NoError(t, platform.ConfigureDefaults(context.TODO(), c, &ip, false))

	h := NewInitializeAction()
	h.InjectLogger(log.Log)
	h.InjectClient(c)

	answer, err := h.Handle(context.TODO(), &ip)
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationPlatformPhaseCreating, answer.Status.Phase)
	assert.Equal(t, 5*time.Minute, answer.Status.Build.GetTimeout().Duration)
}

func TestMavenComputedFromBuildTimeouts(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	timeout, err := time.ParseDuration("1m1ms")
	require.NoError(t, err)
	ip.Spec.Build.Timeout = &metav1.Duration{
		Duration: timeout,
	}
	c, err := test.NewFakeClient(&ip)
	require.NoError(t, err)

	h := NewInitializeAction()
	h.InjectLogger(log.Log)
	h.InjectClient(c)
	answer, err := h.Handle(context.TODO(), &ip)

	require.NoError(t, err)
	assert.NotNil(t, answer)
	assert.Equal(t, 1*time.Minute, answer.Status.Build.GetTimeout().Duration)
}

func TestTruncatedTimeouts(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	bt, err := time.ParseDuration("5m1ms")
	require.NoError(t, err)
	ip.Spec.Build.Timeout = &metav1.Duration{
		Duration: bt,
	}
	c, err := test.NewFakeClient(&ip)
	require.NoError(t, err)

	h := NewInitializeAction()
	h.InjectLogger(log.Log)
	h.InjectClient(c)
	answer, err := h.Handle(context.TODO(), &ip)

	require.NoError(t, err)
	assert.NotNil(t, answer)
	assert.Equal(t, 5*time.Minute, answer.Status.Build.GetTimeout().Duration)
}
