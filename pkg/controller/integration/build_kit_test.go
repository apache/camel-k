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
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestCamelBuildKitDigestChanged(t *testing.T) {
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
			Phase:  v1.IntegrationPhaseBuildingKit,
			Digest: "NonMatchingDigest",
		},
	}
	c, err := internal.NewFakeClient(it)
	require.NoError(t, err)

	a := buildKitAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "build-kit", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationPhaseInitialization, handledIt.Status.Phase)
	// Ready condition
	assert.Nil(t, handledIt.Status.GetCondition(v1.IntegrationConditionReady))
	assert.NotEqual(t, "NonMatchingDigest", handledIt.Status.Digest)
}

func TestCamelBuildKitKitSetOnIntegration(t *testing.T) {
	ik := &v1.IntegrationKit{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKitKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-ik",
			Labels: map[string]string{
				v1.IntegrationKitTypeLabel: v1.IntegrationKitTypePlatform,
			},
		},
		Status: v1.IntegrationKitStatus{
			Phase: v1.IntegrationKitPhaseBuildRunning,
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
			Phase: v1.IntegrationPhaseBuildingKit,
			IntegrationKit: &corev1.ObjectReference{
				APIVersion: ik.APIVersion,
				Kind:       ik.Kind,
				Namespace:  ik.Namespace,
				Name:       ik.Name,
			},
		},
	}
	hash, err := digest.ComputeForIntegration(it, nil, nil)
	require.NoError(t, err)
	it.Status.Digest = hash
	c, err := internal.NewFakeClient(it, ik)
	require.NoError(t, err)

	a := buildKitAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "build-kit", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.NoError(t, err)
	// If the kit is under construction, we expect the Integration to be nil
	assert.Nil(t, handledIt)

	// Move IntegrationKit phase to ready status
	it.Status.Phase = v1.IntegrationPhaseBuildingKit
	ik.Status.Phase = v1.IntegrationKitPhaseReady
	c, err = internal.NewFakeClient(it, ik)
	require.NoError(t, err)
	a.InjectClient(c)
	handledIt, err = a.Handle(context.TODO(), it)
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationPhaseDeploying, handledIt.Status.Phase)

	// Move IntegrationKit phase to ready status
	it.Status.Phase = v1.IntegrationPhaseBuildingKit
	ik.Status.Phase = v1.IntegrationKitPhaseError
	c, err = internal.NewFakeClient(it, ik)
	require.NoError(t, err)
	a.InjectClient(c)
	handledIt, err = a.Handle(context.TODO(), it)
	require.NoError(t, err)
	assert.Equal(t, v1.IntegrationPhaseError, handledIt.Status.Phase)

	// Remove IntegrationKit
	it.Status.Phase = v1.IntegrationPhaseBuildingKit
	c, err = internal.NewFakeClient(it)
	require.NoError(t, err)
	a.InjectClient(c)
	handledIt, err = a.Handle(context.TODO(), it)
	require.Error(t, err)
	assert.Nil(t, handledIt)
}

func TestCamelBuildKitKitLookupExistingKit(t *testing.T) {
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
	ip := &v1.IntegrationPlatform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationPlatformKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "camel-k",
		},
		Status: v1.IntegrationPlatformStatus{
			IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeVersion: defaults.DefaultRuntimeVersion,
				},
			},
			Phase: v1.IntegrationPlatformPhaseReady,
		},
	}
	ik := &v1.IntegrationKit{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKitKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-ik",
			Labels: map[string]string{
				v1.IntegrationKitTypeLabel:           v1.IntegrationKitTypePlatform,
				kubernetes.CamelLabelRuntimeVersion:  defaults.DefaultRuntimeVersion,
				kubernetes.CamelLabelRuntimeProvider: string(v1.RuntimeProviderQuarkus),
			},
		},
		Status: v1.IntegrationKitStatus{
			Phase:           v1.IntegrationKitPhaseBuildRunning,
			RuntimeVersion:  defaults.DefaultRuntimeVersion,
			RuntimeProvider: v1.RuntimeProviderQuarkus,
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
			Phase:           v1.IntegrationPhaseBuildingKit,
			RuntimeVersion:  defaults.DefaultRuntimeVersion,
			RuntimeProvider: v1.RuntimeProviderQuarkus,
		},
	}
	hash, err := digest.ComputeForIntegration(it, nil, nil)
	require.NoError(t, err)
	it.Status.Digest = hash
	c, err := internal.NewFakeClient(it, ik, ip, catalog)
	require.NoError(t, err)

	a := buildKitAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "build-kit", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.NoError(t, err)
	// Found a matching kit (building)
	assert.Equal(t, v1.IntegrationPhaseBuildingKit, handledIt.Status.Phase)
	assert.Equal(t, ik.Name, it.Status.IntegrationKit.Name)
	assert.Equal(t, ik.Namespace, it.Status.IntegrationKit.Namespace)
	// Found a matching kit (ready)
	it.Status.Phase = v1.IntegrationPhaseBuildingKit
	ik.Status.Phase = v1.IntegrationKitPhaseReady
	c, err = internal.NewFakeClient(it, ik, ip, catalog)
	require.NoError(t, err)
	a.InjectClient(c)
	handledIt, err = a.Handle(context.TODO(), it)
	require.NoError(t, err)
	assert.NotNil(t, handledIt)
	assert.Equal(t, v1.IntegrationPhaseDeploying, handledIt.Status.Phase)
	assert.Equal(t, ik.Name, it.Status.IntegrationKit.Name)
	assert.Equal(t, ik.Namespace, it.Status.IntegrationKit.Namespace)
	// Found a matching kit (error)
	it.Status.Phase = v1.IntegrationPhaseBuildingKit
	ik.Status.Phase = v1.IntegrationKitPhaseError
	c, err = internal.NewFakeClient(it, ik, ip, catalog)
	require.NoError(t, err)
	a.InjectClient(c)
	handledIt, err = a.Handle(context.TODO(), it)
	require.NoError(t, err)
	assert.NotNil(t, handledIt)
	assert.Equal(t, v1.IntegrationPhaseError, handledIt.Status.Phase)
	assert.Equal(t, ik.Name, it.Status.IntegrationKit.Name)
	assert.Equal(t, ik.Namespace, it.Status.IntegrationKit.Namespace)
}

func TestCamelBuildKitKitLookupNoMatchingKits(t *testing.T) {
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
	ip := &v1.IntegrationPlatform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationPlatformKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "camel-k",
		},
		Status: v1.IntegrationPlatformStatus{
			IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					RuntimeVersion: defaults.DefaultRuntimeVersion,
				},
			},
			Phase: v1.IntegrationPlatformPhaseReady,
		},
	}
	ik := &v1.IntegrationKit{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKitKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-ik",
			Labels: map[string]string{
				v1.IntegrationKitTypeLabel:           v1.IntegrationKitTypePlatform,
				kubernetes.CamelLabelRuntimeVersion:  defaults.DefaultRuntimeVersion,
				kubernetes.CamelLabelRuntimeProvider: string(v1.RuntimeProviderQuarkus),
			},
		},
		Status: v1.IntegrationKitStatus{
			Phase:           v1.IntegrationKitPhaseBuildRunning,
			RuntimeVersion:  "1.2.3",
			RuntimeProvider: v1.RuntimeProviderQuarkus,
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
			Phase:           v1.IntegrationPhaseBuildingKit,
			RuntimeVersion:  defaults.DefaultRuntimeVersion,
			RuntimeProvider: v1.RuntimeProviderQuarkus,
		},
	}
	hash, err := digest.ComputeForIntegration(it, nil, nil)
	require.NoError(t, err)
	it.Status.Digest = hash
	c, err := internal.NewFakeClient(it, ik, ip, catalog)
	require.NoError(t, err)

	a := buildKitAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "build-kit", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.NoError(t, err)
	// No matching kit found
	assert.Equal(t, v1.IntegrationPhaseBuildingKit, handledIt.Status.Phase)
	assert.NotEqual(t, ik.Name, it.Status.IntegrationKit.Name)
	newKit := v1.NewIntegrationKit(it.Status.IntegrationKit.Namespace, it.Status.IntegrationKit.Name)
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: newKit.Namespace, Name: newKit.Name}, newKit)
	assert.NoError(t, err)
	assert.Equal(t, it.Status.RuntimeVersion, newKit.Labels[kubernetes.CamelLabelRuntimeVersion])
	assert.Equal(t, string(it.Status.RuntimeProvider), newKit.Labels[kubernetes.CamelLabelRuntimeProvider])
	assert.Equal(t, v1.IntegrationKind, newKit.Labels[kubernetes.CamelCreatorLabelKind])
	assert.Equal(t, it.Name, newKit.Labels[kubernetes.CamelCreatorLabelName])
	assert.Equal(t, it.Namespace, newKit.Labels[kubernetes.CamelCreatorLabelNamespace])
	assert.Equal(t, v1.IntegrationKitPhase(""), newKit.Status.Phase)
}
