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

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"

	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/test"

	"github.com/stretchr/testify/assert"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestMonitorSyntheticIntegrationImportingKindUnavailable(t *testing.T) {
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
				v1.IntegrationImportedKindLabel: "SomeKind",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
		},
	}
	c, err := test.NewFakeClient(importedIt)
	assert.Nil(t, err)

	a := monitorSyntheticAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor-synthetic", a.Name())
	assert.True(t, a.CanHandle(importedIt))
	handledIt, err := a.Handle(context.TODO(), importedIt)
	assert.NotNil(t, err)
	assert.Equal(t, v1.IntegrationPhaseError, handledIt.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, v1.IntegrationConditionImportingKindAvailableReason, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Reason)
	assert.Equal(t, "cannot create a synthetic environment for SomeKind kind", handledIt.Status.GetCondition(v1.IntegrationConditionReady).Message)
}

func TestMonitorSyntheticIntegrationCannotMonitorPods(t *testing.T) {
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
			Phase: v1.IntegrationPhaseRunning,
			Conditions: []v1.IntegrationCondition{
				{
					Type:   v1.IntegrationConditionDeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   v1.IntegrationConditionReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-deploy",
			Annotations: map[string]string{
				v1.IntegrationLabel: "my-imported-it",
			},
		},
	}
	c, err := test.NewFakeClient(importedIt, deploy)
	assert.Nil(t, err)

	a := monitorSyntheticAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor-synthetic", a.Name())
	assert.True(t, a.CanHandle(importedIt))
	handledIt, err := a.Handle(context.TODO(), importedIt)
	assert.Nil(t, err)
	assert.Equal(t, v1.IntegrationPhaseCannotMonitor, handledIt.Status.Phase)
	// Ready condition should be still true
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	// Check monitoring pods condition
	assert.Equal(t, corev1.ConditionFalse, handledIt.Status.GetCondition(v1.IntegrationConditionMonitoringPodsAvailable).Status)
	assert.Equal(t, v1.IntegrationConditionMonitoringPodsAvailableReason, handledIt.Status.GetCondition(v1.IntegrationConditionMonitoringPodsAvailable).Reason)
	assert.Equal(t, "Could not find `camel.apache.org/integration: my-imported-it` label in the Deployment/my-deploy template. Make sure to include this label in the template for Pod monitoring purposes.", handledIt.Status.GetCondition(v1.IntegrationConditionMonitoringPodsAvailable).Message)
}

func TestMonitorSyntheticIntegrationDeployment(t *testing.T) {
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
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Container: &trait.ContainerTrait{
					Name: "my-cnt",
				},
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
			Conditions: []v1.IntegrationCondition{
				{
					Type:   v1.IntegrationConditionDeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   v1.IntegrationConditionReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-deploy",
			Annotations: map[string]string{
				v1.IntegrationLabel: "my-imported-it",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						v1.IntegrationLabel: "my-imported-it",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-cnt",
							Image: "my-img",
						},
					},
				},
			},
		},
	}
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pod",
			Labels: map[string]string{
				v1.IntegrationLabel: "my-imported-it",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "my-cnt",
					Image: "my-img",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	c, err := test.NewFakeClient(importedIt, deploy, pod)
	assert.Nil(t, err)

	a := monitorSyntheticAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor-synthetic", a.Name())
	assert.True(t, a.CanHandle(importedIt))
	handledIt, err := a.Handle(context.TODO(), importedIt)
	assert.Nil(t, err)
	assert.Equal(t, v1.IntegrationPhaseRunning, handledIt.Status.Phase)
	assert.Equal(t, int32(1), *handledIt.Status.Replicas)
	// Ready condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, v1.IntegrationConditionDeploymentReadyReason, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Reason)
	assert.Equal(t, "1/1 ready replicas", handledIt.Status.GetCondition(v1.IntegrationConditionReady).Message)
	// Check monitoring pods condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionMonitoringPodsAvailable).Status)
	assert.Equal(t, v1.IntegrationConditionMonitoringPodsAvailableReason, handledIt.Status.GetCondition(v1.IntegrationConditionMonitoringPodsAvailable).Reason)
}

func TestMonitorSyntheticIntegrationCronJob(t *testing.T) {
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
			Phase: v1.IntegrationPhaseRunning,
			Conditions: []v1.IntegrationCondition{
				{
					Type:   v1.IntegrationConditionCronJobAvailable,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   v1.IntegrationConditionReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	cron := &batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "CronJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-cron",
			Labels: map[string]string{
				v1.IntegrationLabel: "my-imported-it",
			},
		},
		Spec: batchv1.CronJobSpec{
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								v1.IntegrationLabel: "my-imported-it",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "my-cnt",
									Image: "my-img",
								},
							},
						},
					},
				},
			},
		},
	}
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pod",
			Labels: map[string]string{
				v1.IntegrationLabel: "my-imported-it",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "my-cnt",
					Image: "my-img",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	c, err := test.NewFakeClient(importedIt, cron, pod)
	assert.Nil(t, err)

	a := monitorSyntheticAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor-synthetic", a.Name())
	assert.True(t, a.CanHandle(importedIt))
	handledIt, err := a.Handle(context.TODO(), importedIt)
	assert.Nil(t, err)
	assert.Equal(t, v1.IntegrationPhaseRunning, handledIt.Status.Phase)
	assert.Equal(t, int32(1), *handledIt.Status.Replicas)
	// Ready condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, v1.IntegrationConditionCronJobCreatedReason, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Reason)
	assert.Equal(t, "cronjob created", handledIt.Status.GetCondition(v1.IntegrationConditionReady).Message)
	// Check monitoring pods condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionMonitoringPodsAvailable).Status)
	assert.Equal(t, v1.IntegrationConditionMonitoringPodsAvailableReason, handledIt.Status.GetCondition(v1.IntegrationConditionMonitoringPodsAvailable).Reason)
}

func TestMonitorSyntheticIntegrationKnativeService(t *testing.T) {
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
			Phase: v1.IntegrationPhaseRunning,
			Conditions: []v1.IntegrationCondition{
				{
					Type:   v1.IntegrationConditionKnativeServiceAvailable,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   v1.IntegrationConditionReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	ksvc := &servingv1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-ksvc",
			Labels: map[string]string{
				v1.IntegrationLabel: "my-imported-it",
			},
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							v1.IntegrationLabel: "my-imported-it",
						},
					},
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "my-cnt",
									Image: "my-img",
								},
							},
						},
					},
				},
			},
		},
		Status: servingv1.ServiceStatus{
			Status: duckv1.Status{
				Conditions: duckv1.Conditions{
					apis.Condition{
						Type:   servingv1.ServiceConditionReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
	}
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pod",
			Labels: map[string]string{
				v1.IntegrationLabel: "my-imported-it",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "my-cnt",
					Image: "my-img",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	c, err := test.NewFakeClient(importedIt, ksvc, pod)
	assert.Nil(t, err)

	a := monitorSyntheticAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor-synthetic", a.Name())
	assert.True(t, a.CanHandle(importedIt))
	handledIt, err := a.Handle(context.TODO(), importedIt)
	assert.Nil(t, err)
	assert.Equal(t, v1.IntegrationPhaseRunning, handledIt.Status.Phase)
	assert.Equal(t, int32(1), *handledIt.Status.Replicas)
	// Ready condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, v1.IntegrationConditionKnativeServiceReadyReason, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Reason)
	// Check monitoring pods condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionMonitoringPodsAvailable).Status)
	assert.Equal(t, v1.IntegrationConditionMonitoringPodsAvailableReason, handledIt.Status.GetCondition(v1.IntegrationConditionMonitoringPodsAvailable).Reason)
}
