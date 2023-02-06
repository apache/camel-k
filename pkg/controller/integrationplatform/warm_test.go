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

	"github.com/apache/camel-k/pkg/platform"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func TestWarm_Succeeded(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ip.Namespace,
			Name:      ip.Name + "-cache",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
		},
	}

	c, err := test.NewFakeClient(&ip, &pod)
	assert.Nil(t, err)

	assert.Nil(t, platform.ConfigureDefaults(context.TODO(), c, &ip, false))

	h := NewWarmAction(c)
	h.InjectLogger(log.Log)
	h.InjectClient(c)

	answer, err := h.Handle(context.TODO(), &ip)
	assert.Nil(t, err)
	assert.NotNil(t, answer)
}

func TestWarm_Failing(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ip.Namespace,
			Name:      ip.Name + "-cache",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
		},
	}

	c, err := test.NewFakeClient(&ip, &pod)
	assert.Nil(t, err)

	assert.Nil(t, platform.ConfigureDefaults(context.TODO(), c, &ip, false))

	h := NewWarmAction(c)
	h.InjectLogger(log.Log)
	h.InjectClient(c)

	answer, err := h.Handle(context.TODO(), &ip)
	assert.NotNil(t, err)
	assert.Nil(t, answer)
}

func TestWarm_WarmingUp(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = xid.New().String()
	ip.Spec.Cluster = v1.IntegrationPlatformClusterOpenShift
	ip.Spec.Profile = v1.TraitProfileOpenShift

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ip.Namespace,
			Name:      ip.Name + "-cache",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	c, err := test.NewFakeClient(&ip, &pod)
	assert.Nil(t, err)

	assert.Nil(t, platform.ConfigureDefaults(context.TODO(), c, &ip, false))

	h := NewWarmAction(c)
	h.InjectLogger(log.Log)
	h.InjectClient(c)

	answer, err := h.Handle(context.TODO(), &ip)
	assert.Nil(t, err)
	assert.Nil(t, answer)
}
