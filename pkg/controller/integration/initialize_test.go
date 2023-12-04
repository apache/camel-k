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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestCamelImportDeployment(t *testing.T) {
	importedIt := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-imported-it",
			Annotations: map[string]string{
				v1.IntegrationImportedNameLabel: "my-deploy",
				v1.IntegrationSyntheticLabel:    "true",
				v1.IntegrationImportedKindLabel: "Deployment",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseInitialization,
		},
	}
	c, err := test.NewFakeClient(importedIt)
	assert.Nil(t, err)

	a := initializeAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(importedIt))
	handledIt, err := a.Handle(context.TODO(), importedIt)
	assert.Nil(t, err)
	assert.Equal(t, v1.IntegrationPhaseRunning, handledIt.Status.Phase)
	// Ready condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, v1.IntegrationConditionDeploymentReadyReason, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Reason)
	assert.Equal(t, "imported from my-deploy Deployment", handledIt.Status.GetCondition(v1.IntegrationConditionReady).Message)
	// Deployment condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionDeploymentAvailable).Status)
	assert.Equal(t, v1.IntegrationConditionDeploymentAvailableReason, handledIt.Status.GetCondition(v1.IntegrationConditionDeploymentAvailable).Reason)
	assert.Equal(t, "imported from my-deploy Deployment", handledIt.Status.GetCondition(v1.IntegrationConditionDeploymentAvailable).Message)
}

func TestCamelImportCronJob(t *testing.T) {
	importedIt := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-imported-it",
			Annotations: map[string]string{
				v1.IntegrationImportedNameLabel: "my-cron",
				v1.IntegrationSyntheticLabel:    "true",
				v1.IntegrationImportedKindLabel: "CronJob",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseInitialization,
		},
	}
	c, err := test.NewFakeClient(importedIt)
	assert.Nil(t, err)

	a := initializeAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(importedIt))
	handledIt, err := a.Handle(context.TODO(), importedIt)
	assert.Nil(t, err)
	assert.Equal(t, v1.IntegrationPhaseRunning, handledIt.Status.Phase)
	// Ready condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, v1.IntegrationConditionDeploymentReadyReason, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Reason)
	assert.Equal(t, "imported from my-cron CronJob", handledIt.Status.GetCondition(v1.IntegrationConditionReady).Message)
	// CronJob condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionCronJobAvailable).Status)
	assert.Equal(t, v1.IntegrationConditionCronJobCreatedReason, handledIt.Status.GetCondition(v1.IntegrationConditionCronJobAvailable).Reason)
	assert.Equal(t, "imported from my-cron CronJob", handledIt.Status.GetCondition(v1.IntegrationConditionCronJobAvailable).Message)
}

func TestCamelImportKnativeService(t *testing.T) {
	importedIt := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-imported-it",
			Annotations: map[string]string{
				v1.IntegrationImportedNameLabel: "my-ksvc",
				v1.IntegrationSyntheticLabel:    "true",
				v1.IntegrationImportedKindLabel: "KnativeService",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseInitialization,
		},
	}
	c, err := test.NewFakeClient(importedIt)
	assert.Nil(t, err)

	a := initializeAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(importedIt))
	handledIt, err := a.Handle(context.TODO(), importedIt)
	assert.Nil(t, err)
	assert.Equal(t, v1.IntegrationPhaseRunning, handledIt.Status.Phase)
	// Ready condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, v1.IntegrationConditionKnativeServiceReadyReason, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Reason)
	assert.Equal(t, "imported from my-ksvc KnativeService", handledIt.Status.GetCondition(v1.IntegrationConditionReady).Message)
	// Knative Service condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionKnativeServiceAvailable).Status)
	assert.Equal(t, v1.IntegrationConditionKnativeServiceAvailableReason, handledIt.Status.GetCondition(v1.IntegrationConditionKnativeServiceAvailable).Reason)
	assert.Equal(t, "imported from my-ksvc KnativeService", handledIt.Status.GetCondition(v1.IntegrationConditionKnativeServiceAvailable).Message)
}

func TestCamelImportUnsupportedKind(t *testing.T) {
	importedIt := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-imported-it",
			Annotations: map[string]string{
				v1.IntegrationImportedNameLabel: "my-kind",
				v1.IntegrationSyntheticLabel:    "true",
				v1.IntegrationImportedKindLabel: "SomeKind",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseInitialization,
		},
	}
	c, err := test.NewFakeClient(importedIt)
	assert.Nil(t, err)

	a := initializeAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "initialize", a.Name())
	assert.True(t, a.CanHandle(importedIt))
	handledIt, err := a.Handle(context.TODO(), importedIt)
	assert.Nil(t, err)
	assert.Equal(t, v1.IntegrationPhaseError, handledIt.Status.Phase)
	// Ready condition
	assert.Equal(t, corev1.ConditionFalse, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, v1.IntegrationConditionImportingKindAvailableReason, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Reason)
	assert.Equal(t, "Unsupported SomeKind import kind", handledIt.Status.GetCondition(v1.IntegrationConditionReady).Message)
}
