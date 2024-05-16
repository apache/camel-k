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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/client"

	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIntegrationSecretAndConfigmapResourceVersions(t *testing.T) {
	cm := kubernetes.NewConfigMap("default", "cm-test", "test.txt", "test.txt", "xyz", nil)
	sec := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sec-test",
			Namespace: "default",
		},
		Immutable: pointer.Bool(true),
	}
	sec.Data = map[string][]byte{
		"test.txt": []byte("hello"),
	}
	it := &v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-it",
			Namespace: "default",
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Mount: &trait.MountTrait{
					Configs:   []string{"configmap:cm-test"},
					Resources: []string{"secret:sec-test"},
				},
			},
		},
	}
	c, err := test.NewFakeClient(cm, sec)
	assert.Nil(t, err)
	// Default hot reload (false)
	configmaps, secrets := getIntegrationSecretAndConfigmapResourceVersions(context.TODO(), c, it)
	assert.Len(t, configmaps, 0)
	assert.Len(t, secrets, 0)
	// Enabled hot reload (true)
	it.Spec.Traits.Mount.HotReload = pointer.Bool(true)
	configmaps, secrets = getIntegrationSecretAndConfigmapResourceVersions(context.TODO(), c, it)
	assert.Len(t, configmaps, 1)
	assert.Len(t, secrets, 1)
	// We cannot guess resource version value. It should be enough to have any non empty value though.
	assert.NotEqual(t, "", configmaps[0])
	assert.NotEqual(t, "", secrets[0])
}

func TestMonitorIntegration(t *testing.T) {
	c, it, err := nominalEnvironment()
	require.NoError(t, err)

	a := monitorAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationPhaseRunning, handledIt.Status.Phase)
	assert.Equal(t, int32(1), *handledIt.Status.Replicas)
	// Ready condition
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, v1.IntegrationConditionDeploymentReadyReason, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Reason)
}

func TestMonitorFailureIntegration(t *testing.T) {
	c, it, err := nominalEnvironment()
	require.NoError(t, err)

	// simulate a trait execution failure
	it.Status.RuntimeVersion = "0.0.0"

	a := monitorAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.Error(t, err)
	assert.Equal(t, v1.IntegrationPhaseError, handledIt.Status.Phase)
	// Ready condition
	assert.Equal(t, corev1.ConditionFalse, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, v1.IntegrationConditionInitializationFailedReason, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Reason)
}

func TestMonitorIntegrationDuringOperatorUpgrade(t *testing.T) {
	c, it, err := nominalEnvironment()
	require.NoError(t, err)

	// Simulate a different version on which this Integration
	// was supposedly coming from
	it.Status.Version = "1.1.1"
	hash, _ := digest.ComputeForIntegration(it, nil, nil)
	it.Status.Digest = hash

	a := monitorAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "monitor", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationPhaseRunning, handledIt.Status.Phase)
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionReady).Status)
	assert.Equal(t, corev1.ConditionTrue, handledIt.Status.GetCondition(v1.IntegrationConditionUpgradeRequired).Status)
}

func nominalEnvironment() (client.Client, *v1.Integration, error) {
	catalog := &v1.CamelCatalog{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.CamelCatalogKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "camel-k-catalog",
		},
		Spec: v1.CamelCatalogSpec{
			Runtime: v1.RuntimeSpec{
				Provider: v1.RuntimeProviderQuarkus,
				Version:  defaults.DefaultRuntimeVersion,
			},
		},
	}
	platform := &v1.IntegrationPlatform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationPlatformKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "camel-k",
		},
		Status: v1.IntegrationPlatformStatus{
			Phase: v1.IntegrationPlatformPhaseReady,
		},
	}
	kit := &v1.IntegrationKit{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKitKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-kit",
		},
		Status: v1.IntegrationKitStatus{
			Phase: v1.IntegrationKitPhaseReady,
		},
	}
	it := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-it",
		},
		Status: v1.IntegrationStatus{
			Version:        defaults.Version,
			RuntimeVersion: defaults.DefaultRuntimeVersion,
			Phase:          v1.IntegrationPhaseRunning,
			IntegrationKit: &corev1.ObjectReference{
				Name:       kit.Name,
				Namespace:  kit.Namespace,
				Kind:       kit.Kind,
				APIVersion: kit.APIVersion,
			},
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
	hash, _ := digest.ComputeForIntegration(it, nil, nil)
	it.Status.Digest = hash
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-pod",
			Labels: map[string]string{
				v1.IntegrationLabel: "my-it",
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
	c, err := test.NewFakeClient(catalog, platform, it, kit, pod)
	return c, it, err
}
