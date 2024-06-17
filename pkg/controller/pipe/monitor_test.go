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

package pipe

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeIntegrationSpecChanged(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Source: v1.Endpoint{
				URI: pointer.String("timer:tick"),
			},
			Sink: v1.Endpoint{
				URI: pointer.String("log:info"),
			},
		},
		Status: v1.PipeStatus{
			Phase: v1.PipePhaseReady,
		},
	}
	it := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
		},
	}
	c, err := test.NewFakeClient(pipe, it)
	require.NoError(t, err)

	a := NewMonitorAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseNone, handledPipe.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, handledPipe.Status.GetCondition(v1.PipeConditionReady).Status)
}

func TestPipeIntegrationReady(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Source: v1.Endpoint{
				URI: pointer.String("timer:tick"),
			},
			Sink: v1.Endpoint{
				URI: pointer.String("log:info"),
			},
		},
		Status: v1.PipeStatus{
			Phase: v1.PipePhaseReady,
		},
	}

	c, err := test.NewFakeClient(pipe)
	require.NoError(t, err)
	// We calculate the integration the same way it does the operator
	// as we don't expect it to change in this test.
	it, err := CreateIntegrationFor(context.TODO(), c, pipe)
	it.Status.Phase = v1.IntegrationPhaseRunning
	it.Status.SetCondition(v1.IntegrationConditionReady, corev1.ConditionTrue, "Running", "Running")
	require.NoError(t, err)
	c, err = test.NewFakeClient(pipe, it)
	require.NoError(t, err)

	a := NewMonitorAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseReady, handledPipe.Status.Phase)
	assert.Equal(t, corev1.ConditionTrue, handledPipe.Status.GetCondition(v1.PipeConditionReady).Status)
}

func TestPipeIntegrationUnknown(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Source: v1.Endpoint{
				URI: pointer.String("timer:tick"),
			},
			Sink: v1.Endpoint{
				URI: pointer.String("log:info"),
			},
		},
		Status: v1.PipeStatus{
			Phase: v1.PipePhaseReady,
		},
	}

	c, err := test.NewFakeClient(pipe)
	require.NoError(t, err)
	// We calculate the integration the same way it does the operator
	// as we don't expect it to change in this test.
	it, err := CreateIntegrationFor(context.TODO(), c, pipe)
	it.Status.Phase = v1.IntegrationPhaseRunning
	require.NoError(t, err)
	c, err = test.NewFakeClient(pipe, it)
	require.NoError(t, err)

	a := NewMonitorAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseReady, handledPipe.Status.Phase)
	assert.Equal(t, corev1.ConditionUnknown, handledPipe.Status.GetCondition(v1.PipeConditionReady).Status)
}

func TestPipeIntegrationError(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Source: v1.Endpoint{
				URI: pointer.String("timer:tick"),
			},
			Sink: v1.Endpoint{
				URI: pointer.String("log:info"),
			},
		},
		Status: v1.PipeStatus{
			Phase: v1.PipePhaseReady,
		},
	}

	c, err := test.NewFakeClient(pipe)
	require.NoError(t, err)
	// We calculate the integration the same way it does the operator
	// as we don't expect it to change in this test.
	it, err := CreateIntegrationFor(context.TODO(), c, pipe)
	it.Status.Phase = v1.IntegrationPhaseError
	it.Status.SetCondition(v1.IntegrationConditionReady, corev1.ConditionFalse, "ErrorReason", "Error message")
	require.NoError(t, err)
	c, err = test.NewFakeClient(pipe, it)
	require.NoError(t, err)

	a := NewMonitorAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseError, handledPipe.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, handledPipe.Status.GetCondition(v1.PipeConditionReady).Status)
	assert.Equal(t, "Error message", handledPipe.Status.GetCondition(v1.PipeConditionReady).Message)
}

func TestPipeIntegrationErrorFromPipeErrorPhase(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Status: v1.PipeStatus{
			Phase: v1.PipePhaseError,
		},
	}

	c, err := test.NewFakeClient(pipe)
	require.NoError(t, err)

	a := NewMonitorAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.Error(t, err)
	assert.Equal(t, v1.PipePhaseError, handledPipe.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, handledPipe.Status.GetCondition(v1.PipeConditionReady).Status)
	assert.Equal(t, "no ref or URI specified in endpoint", handledPipe.Status.GetCondition(v1.PipeConditionReady).Message)
}

func TestPipeIntegrationCreatingFromPipeErrorPhase(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Source: v1.Endpoint{
				URI: pointer.String("timer:tick"),
			},
			Sink: v1.Endpoint{
				URI: pointer.String("log:info"),
			},
		},
		Status: v1.PipeStatus{
			Phase: v1.PipePhaseError,
		},
	}

	c, err := test.NewFakeClient(pipe)
	require.NoError(t, err)

	a := NewMonitorAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseCreating, handledPipe.Status.Phase)
}

func TestPipeIntegrationCreatingFromPipeCreatingPhase(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
		},
		Spec: v1.PipeSpec{
			Source: v1.Endpoint{
				URI: pointer.String("timer:tick"),
			},
			Sink: v1.Endpoint{
				URI: pointer.String("log:info"),
			},
		},
		Status: v1.PipeStatus{
			Phase: v1.PipePhaseCreating,
		},
	}

	c, err := test.NewFakeClient(pipe)
	require.NoError(t, err)
	// We calculate the integration the same way it does the operator
	// as we don't expect it to change in this test.
	it, err := CreateIntegrationFor(context.TODO(), c, pipe)
	require.NoError(t, err)
	it.Status.Phase = v1.IntegrationPhaseBuildingKit
	c, err = test.NewFakeClient(pipe, it)
	require.NoError(t, err)

	a := NewMonitorAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseCreating, handledPipe.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, handledPipe.Status.GetCondition(v1.PipeConditionReady).Status)
	assert.Equal(t, "Integration \"my-pipe\" is in \"Creating\" phase", handledPipe.Status.GetCondition(v1.PipeConditionReady).Message)
}

func TestPipeIntegrationPipeTraitAnnotations(t *testing.T) {
	pipe := &v1.Pipe{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.PipeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pipe",
			Annotations: map[string]string{
				v1.TraitAnnotationPrefix + "camel.runtime-version": "1.2.3",
			},
		},
		Spec: v1.PipeSpec{
			Source: v1.Endpoint{
				URI: pointer.String("timer:tick"),
			},
			Sink: v1.Endpoint{
				URI: pointer.String("log:info"),
			},
		},
		Status: v1.PipeStatus{
			Phase: v1.PipePhaseCreating,
		},
	}

	c, err := test.NewFakeClient(pipe)
	require.NoError(t, err)
	// We calculate the integration the same way it does the operator
	// as we don't expect it to change in this test.
	it, err := CreateIntegrationFor(context.TODO(), c, pipe)
	require.NoError(t, err)
	it.Status.Phase = v1.IntegrationPhaseBuildingKit
	c, err = test.NewFakeClient(pipe, it)
	require.NoError(t, err)

	a := NewMonitorAction()
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor", a.Name())
	assert.True(t, a.CanHandle(pipe))
	handledPipe, err := a.Handle(context.TODO(), pipe)
	require.NoError(t, err)
	assert.Equal(t, v1.PipePhaseCreating, handledPipe.Status.Phase)
	assert.Equal(t, corev1.ConditionFalse, handledPipe.Status.GetCondition(v1.PipeConditionReady).Status)
	assert.Equal(t, "Integration \"my-pipe\" is in \"Creating\" phase", handledPipe.Status.GetCondition(v1.PipeConditionReady).Message)
}
