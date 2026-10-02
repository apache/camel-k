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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	servingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
)

func newKnativeServiceController(ready *metav1.Condition) *knativeServiceController {
	svc := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-ksvc",
		},
	}
	if ready != nil {
		svc.Status.Conditions = []metav1.Condition{*ready}
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
	c := newKnativeServiceController(&metav1.Condition{
		Type:   servingv1.ServiceConditionReady,
		Status: metav1.ConditionTrue,
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
	c := newKnativeServiceController(&metav1.Condition{
		Type:    servingv1.ServiceConditionReady,
		Status:  metav1.ConditionFalse,
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
	c := newKnativeServiceController(&metav1.Condition{
		Type:    servingv1.ServiceConditionReady,
		Status:  metav1.ConditionFalse,
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

func TestKnativeServiceControllerUnknown(t *testing.T) {
	c := newKnativeServiceController(&metav1.Condition{
		Type:    servingv1.ServiceConditionReady,
		Status:  metav1.ConditionUnknown,
		Reason:  "RevisionMissing",
		Message: "waiting for the revision",
	})

	done, err := c.checkReadyCondition(context.TODO())
	require.NoError(t, err)
	assert.False(t, done)
	assert.Equal(t, v1.IntegrationPhaseRunning, c.integration.Status.Phase)

	assert.False(t, c.updateReadyCondition(0))
	cond := c.integration.Status.GetCondition(v1.IntegrationConditionReady)
	require.NotNil(t, cond)
	assert.Equal(t, corev1.ConditionFalse, cond.Status)
	assert.Equal(t, "RevisionMissing", cond.Reason)
	assert.Equal(t, "waiting for the revision", cond.Message)
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

// knativeServicePayload mimics a Knative Serving Service as returned by the API server:
// it carries a number of fields the duck type does not model, which must be ignored.
const knativeServicePayload = `{
	"apiVersion": "serving.knative.dev/v1",
	"kind": "Service",
	"metadata": {"name": "my-ksvc", "namespace": "ns", "generation": 2},
	"spec": {
		"template": {
			"metadata": {"labels": {"camel.apache.org/integration": "my-it"}},
			"spec": {
				"containerConcurrency": 0,
				"containers": [{"name": "integration", "image": "my-image"}],
				"enableServiceLinks": false,
				"timeoutSeconds": 300
			}
		},
		"traffic": [{"latestRevision": true, "percent": 100}]
	},
	"status": {
		"address": {"url": "http://my-ksvc.ns.svc.cluster.local"},
		"conditions": [
			{"lastTransitionTime": "2026-01-02T03:04:05Z", "status": "True", "type": "ConfigurationsReady"},
			{"lastTransitionTime": "2026-01-02T03:04:05Z", "status": "False", "type": "Ready",
				"reason": "RevisionFailed", "message": "Revision \"my-ksvc-00001\" failed with message: Container failed."},
			{"lastTransitionTime": "2026-01-02T03:04:05Z", "status": "False", "type": "RoutesReady",
				"reason": "RevisionMissing", "message": "Configuration \"my-ksvc\" does not have any ready Revision.", "severity": "Info"}
		],
		"latestCreatedRevisionName": "my-ksvc-00001",
		"latestReadyRevisionName": "my-ksvc-00001",
		"observedGeneration": 2,
		"traffic": [{"latestRevision": true, "percent": 100, "revisionName": "my-ksvc-00001"}],
		"url": "http://my-ksvc.ns.example.com"
	}
}`

func TestKnativeServiceControllerFromAPIServerPayload(t *testing.T) {
	svc := &servingv1.Service{}
	require.NoError(t, json.Unmarshal([]byte(knativeServicePayload), svc))

	assert.Equal(t, "http://my-ksvc.ns.example.com", svc.Status.URL.String())
	require.NotNil(t, svc.Status.Address)
	assert.Equal(t, "my-ksvc.ns.svc.cluster.local", svc.Status.Address.URL.Host)
	assert.Equal(t, int64(300), ptr.Deref(svc.Spec.Template.Spec.TimeoutSeconds, 0))
	assert.Len(t, svc.Spec.Template.Spec.Containers, 1)

	c := &knativeServiceController{
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
	assert.True(t, c.hasTemplateIntegrationLabel())
	assert.Equal(t, "KnativeService/my-ksvc", c.getControllerName())

	done, err := c.checkReadyCondition(context.TODO())
	require.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, v1.IntegrationPhaseError, c.integration.Status.Phase)
	cond := c.integration.Status.GetCondition(v1.IntegrationConditionReady)
	require.NotNil(t, cond)
	assert.Equal(t, corev1.ConditionFalse, cond.Status)
	assert.Contains(t, cond.Message, "my-ksvc-00001")
}
