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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/duck/knative/apis"
	servingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
)

func newKnativeServiceController(ready *apis.Condition) *knativeServiceController {
	svc := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-ksvc",
		},
	}
	if ready != nil {
		svc.Status.Conditions = apis.Conditions{*ready}
	}

	return &knativeServiceController{
		obj: svc,
		integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-it",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
	}
}

func TestKnativeServiceControllerReady(t *testing.T) {
	c := newKnativeServiceController(&apis.Condition{
		Type:   servingv1.ServiceConditionReady,
		Status: corev1.ConditionTrue,
	})

	done, err := c.checkReadyCondition(context.TODO())
	require.NoError(t, err)
	assert.False(t, done)
	assert.Equal(t, v1.IntegrationPhaseRunning, c.integration.Status.Phase)

	assert.True(t, c.updateReadyCondition(1))
	cond := c.integration.Status.GetCondition(v1.IntegrationConditionReady)
	require.NotNil(t, cond)
	assert.Equal(t, corev1.ConditionTrue, cond.Status)
	assert.Equal(t, v1.IntegrationConditionKnativeServiceReadyReason, cond.Reason)
}

func TestKnativeServiceControllerRevisionFailed(t *testing.T) {
	c := newKnativeServiceController(&apis.Condition{
		Type:    servingv1.ServiceConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  "RevisionFailed",
		Message: "revision failed",
	})

	done, err := c.checkReadyCondition(context.TODO())
	require.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, v1.IntegrationPhaseError, c.integration.Status.Phase)
	cond := c.integration.Status.GetCondition(v1.IntegrationConditionReady)
	require.NotNil(t, cond)
	assert.Equal(t, corev1.ConditionFalse, cond.Status)
	assert.Equal(t, "revision failed", cond.Message)
}

func TestKnativeServiceControllerNotReady(t *testing.T) {
	c := newKnativeServiceController(&apis.Condition{
		Type:    servingv1.ServiceConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  "Deploying",
		Message: "still deploying",
	})

	done, err := c.checkReadyCondition(context.TODO())
	require.NoError(t, err)
	assert.False(t, done)
	assert.Equal(t, v1.IntegrationPhaseRunning, c.integration.Status.Phase)

	assert.False(t, c.updateReadyCondition(0))
	cond := c.integration.Status.GetCondition(v1.IntegrationConditionReady)
	require.NotNil(t, cond)
	assert.Equal(t, corev1.ConditionFalse, cond.Status)
	assert.Equal(t, "Deploying", cond.Reason)
	assert.Equal(t, "still deploying", cond.Message)
}

func TestKnativeServiceControllerMissingCondition(t *testing.T) {
	c := newKnativeServiceController(nil)

	done, err := c.checkReadyCondition(context.TODO())
	require.NoError(t, err)
	assert.False(t, done)
	assert.Equal(t, v1.IntegrationPhaseRunning, c.integration.Status.Phase)

	assert.False(t, c.updateReadyCondition(0))
	cond := c.integration.Status.GetCondition(v1.IntegrationConditionReady)
	require.NotNil(t, cond)
	assert.Equal(t, corev1.ConditionFalse, cond.Status)
}
