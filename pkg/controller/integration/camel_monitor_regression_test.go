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
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	monitorv1alpha1 "github.com/camel-tooling/camel-monitor-operator/pkg/apis/camel/v1alpha1"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCamelMonitorLabeledResourceWithDifferentName(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "example.org/monitor")
	integration := v1.NewIntegration("test", "sample")
	monitor := monitorv1alpha1.NewCamelMonitor("test", "custom-monitor")
	monitor.Labels = map[string]string{"example.org/monitor": integration.Name}
	c, err := internal.NewFakeClient()
	require.NoError(t, err)
	require.NoError(t, c.Create(context.Background(), &monitor))

	require.NoError(t, updateCamelMonitorCondition(context.Background(), c, &integration))
	condition := integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable)
	require.NotNil(t, condition)
	require.Equal(t, corev1.ConditionTrue, condition.Status)
}

func TestCamelMonitorSyntheticResourceUsesDiscoveryLabel(t *testing.T) {
	t.Setenv(kubernetes.CamelMonitorOperatorLabelEnvVar, "example.org/monitor")
	integration := v1.NewIntegration("test", "sample")
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Namespace: integration.Namespace, Name: "sample-revision",
		Labels: map[string]string{v1.IntegrationLabel: integration.Name, "example.org/monitor": "custom-app"},
	}}
	// The official monitor operator keeps source annotations, without copying labels.
	monitor := monitorv1alpha1.NewCamelMonitor("test", "custom-app-sample-revision")
	monitor.Annotations = map[string]string{
		monitorv1alpha1.MonitorImportedKindLabel: "Deployment",
		monitorv1alpha1.MonitorImportedNameLabel: deployment.Name,
	}
	c, err := internal.NewFakeClient(deployment)
	require.NoError(t, err)
	require.NoError(t, c.Create(context.Background(), &monitor))

	require.NoError(t, updateCamelMonitorCondition(context.Background(), c, &integration))
	condition := integration.Status.GetCondition(v1.IntegrationConditionCamelMonitorAvailable)
	require.NotNil(t, condition)
	require.Equal(t, corev1.ConditionTrue, condition.Status)
}
