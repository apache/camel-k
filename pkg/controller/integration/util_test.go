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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestLookupContextForIntegration_DiscardContextsInError(t *testing.T) {
	c, err := test.NewFakeClient(
		&v1alpha1.IntegrationContext{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationContextKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-context-1",
				Labels: map[string]string{
					"camel.apache.org/context.type": v1alpha1.IntegrationContextTypePlatform,
				},
			},
			Spec: v1alpha1.IntegrationContextSpec{
				Dependencies: []string{
					"camel-core",
					"camel-irc",
				},
			},
			Status: v1alpha1.IntegrationContextStatus{
				Phase: v1alpha1.IntegrationContextPhaseError,
			},
		},
		&v1alpha1.IntegrationContext{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationContextKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-context-2",
				Labels: map[string]string{
					"camel.apache.org/context.type": v1alpha1.IntegrationContextTypePlatform,
				},
			},
			Spec: v1alpha1.IntegrationContextSpec{
				Dependencies: []string{
					"camel-core",
					"camel-irc",
				},
			},
			Status: v1alpha1.IntegrationContextStatus{
				Phase: v1alpha1.IntegrationContextPhaseReady,
			},
		},
	)

	assert.Nil(t, err)

	i, err := LookupContextForIntegration(context.TODO(), c, &v1alpha1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-integration",
		},
		Status: v1alpha1.IntegrationStatus{
			Dependencies: []string{
				"camel-core",
				"camel-irc",
			},
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, i)
	assert.Equal(t, "my-context-2", i.Name)
}

func TestLookupContextForIntegration_DiscardContextsWithIncompatibleTraits(t *testing.T) {
	c, err := test.NewFakeClient(
		//
		// Should be discarded because it contains both of the required traits but one
		// contains a different configuration value
		//
		&v1alpha1.IntegrationContext{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationContextKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-context-1",
				Labels: map[string]string{
					"camel.apache.org/context.type": v1alpha1.IntegrationContextTypePlatform,
				},
			},
			Spec: v1alpha1.IntegrationContextSpec{
				Dependencies: []string{
					"camel-core",
					"camel-irc",
				},
				Traits: map[string]v1alpha1.TraitSpec{
					"knative": {
						Configuration: map[string]string{
							"enabled": "true",
						},
					},
					"knative-service": {
						Configuration: map[string]string{
							"enabled": "false",
						},
					},
				},
			},
			Status: v1alpha1.IntegrationContextStatus{
				Phase: v1alpha1.IntegrationContextPhaseReady,
			},
		},
		//
		// Should be discarded because it contains a subset of the required traits but
		// with different configuration value
		//
		&v1alpha1.IntegrationContext{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationContextKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-context-2",
				Labels: map[string]string{
					"camel.apache.org/context.type": v1alpha1.IntegrationContextTypePlatform,
				},
			},
			Spec: v1alpha1.IntegrationContextSpec{
				Dependencies: []string{
					"camel-core",
					"camel-irc",
				},
				Traits: map[string]v1alpha1.TraitSpec{
					"knative": {
						Configuration: map[string]string{
							"enabled": "false",
						},
					},
				},
			},
			Status: v1alpha1.IntegrationContextStatus{
				Phase: v1alpha1.IntegrationContextPhaseReady,
			},
		},
		//
		// Should be discarded because it contains both of the required traits but
		// also an additional one
		//
		&v1alpha1.IntegrationContext{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationContextKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-context-3",
				Labels: map[string]string{
					"camel.apache.org/context.type": v1alpha1.IntegrationContextTypePlatform,
				},
			},
			Spec: v1alpha1.IntegrationContextSpec{
				Dependencies: []string{
					"camel-core",
					"camel-irc",
				},
				Traits: map[string]v1alpha1.TraitSpec{
					"knative": {
						Configuration: map[string]string{
							"enabled": "true",
						},
					},
					"knative-service": {
						Configuration: map[string]string{
							"enabled": "true",
						},
					},
					"gc": {
						Configuration: map[string]string{
							"enabled": "true",
						},
					},
				},
			},
			Status: v1alpha1.IntegrationContextStatus{
				Phase: v1alpha1.IntegrationContextPhaseReady,
			},
		},
		//
		// Should be discarded because it contains a subset of the required traits and
		// same configuration values
		//
		&v1alpha1.IntegrationContext{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationContextKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-context-4",
				Labels: map[string]string{
					"camel.apache.org/context.type": v1alpha1.IntegrationContextTypePlatform,
				},
			},
			Spec: v1alpha1.IntegrationContextSpec{
				Dependencies: []string{
					"camel-core",
					"camel-irc",
				},
				Traits: map[string]v1alpha1.TraitSpec{
					"knative": {
						Configuration: map[string]string{
							"enabled": "true",
						},
					},
				},
			},
			Status: v1alpha1.IntegrationContextStatus{
				Phase: v1alpha1.IntegrationContextPhaseReady,
			},
		},
	)

	assert.Nil(t, err)

	i, err := LookupContextForIntegration(context.TODO(), c, &v1alpha1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-integration",
		},
		Spec: v1alpha1.IntegrationSpec{
			Traits: map[string]v1alpha1.TraitSpec{
				"knative": {
					Configuration: map[string]string{
						"enabled": "true",
					},
				},
				"knative-service": {
					Configuration: map[string]string{
						"enabled": "true",
					},
				},
			},
		},
		Status: v1alpha1.IntegrationStatus{
			Dependencies: []string{
				"camel-core",
				"camel-irc",
			},
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, i)
	assert.Equal(t, "my-context-4", i.Name)
}
