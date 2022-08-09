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
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"

	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestLookupKitForIntegration_DiscardKitsInError(t *testing.T) {
	c, err := test.NewFakeClient(
		&v1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.IntegrationKitKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-kit-1",
				Labels: map[string]string{
					v1.IntegrationKitTypeLabel: v1.IntegrationKitTypePlatform,
				},
			},
			Spec: v1.IntegrationKitSpec{
				Dependencies: []string{
					"camel-core",
					"camel-irc",
				},
			},
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseError,
			},
		},
		&v1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.IntegrationKitKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-kit-2",
				Labels: map[string]string{
					v1.IntegrationKitTypeLabel: v1.IntegrationKitTypePlatform,
				},
			},
			Spec: v1.IntegrationKitSpec{
				Dependencies: []string{
					"camel-core",
					"camel-irc",
				},
			},
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
	)

	assert.Nil(t, err)

	a := buildKitAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)

	kits, err := lookupKitsForIntegration(context.TODO(), c, &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-integration",
		},
		Status: v1.IntegrationStatus{
			Dependencies: []string{
				"camel-core",
				"camel-irc",
			},
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, kits)
	assert.Len(t, kits, 1)
	assert.Equal(t, "my-kit-2", kits[0].Name)
}

func TestLookupKitForIntegration_DiscardKitsWithIncompatibleTraits(t *testing.T) {
	c, err := test.NewFakeClient(
		// Should be discarded because it does not contain the required traits
		&v1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.IntegrationKitKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-kit-1",
				Labels: map[string]string{
					v1.IntegrationKitTypeLabel: v1.IntegrationKitTypePlatform,
				},
			},
			Spec: v1.IntegrationKitSpec{
				Dependencies: []string{
					"camel-core",
					"camel-irc",
				},
			},
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		// Should be discarded because it contains a subset of the required traits but
		// with different configuration value
		&v1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.IntegrationKitKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-kit-2",
				Labels: map[string]string{
					v1.IntegrationKitTypeLabel: v1.IntegrationKitTypePlatform,
				},
			},
			Spec: v1.IntegrationKitSpec{
				Dependencies: []string{
					"camel-core",
					"camel-irc",
				},
				Traits: v1.IntegrationKitTraits{
					Builder: &traitv1.BuilderTrait{
						Trait: traitv1.Trait{
							Enabled: pointer.Bool(false),
						},
					},
				},
			},
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		// Should NOT be discarded because it contains a subset of the required traits and
		// same configuration values
		&v1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.IntegrationKitKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-kit-3",
				Labels: map[string]string{
					v1.IntegrationKitTypeLabel: v1.IntegrationKitTypePlatform,
				},
			},
			Spec: v1.IntegrationKitSpec{
				Dependencies: []string{
					"camel-core",
					"camel-irc",
				},
				Traits: v1.IntegrationKitTraits{
					Builder: &traitv1.BuilderTrait{
						Trait: traitv1.Trait{
							Enabled: pointer.Bool(true),
						},
						Properties: []string{
							"build-key1=build-value1",
						},
					},
				},
			},
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
	)

	assert.Nil(t, err)

	a := buildKitAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)

	kits, err := lookupKitsForIntegration(context.TODO(), c, &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-integration",
		},
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Builder: &traitv1.BuilderTrait{
					Trait: traitv1.Trait{
						Enabled: pointer.Bool(true),
					},
					Properties: []string{
						"build-key1=build-value1",
					},
				},
			},
		},
		Status: v1.IntegrationStatus{
			Dependencies: []string{
				"camel-core",
				"camel-irc",
			},
		},
	})

	assert.Nil(t, err)
	assert.NotNil(t, kits)
	assert.Len(t, kits, 1)
	assert.Equal(t, "my-kit-3", kits[0].Name)
}

func TestHasMatchingTraits_KitNoTraitShouldNotBePicked(t *testing.T) {
	integration := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-integration",
		},
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Builder: &traitv1.BuilderTrait{
					Trait: traitv1.Trait{
						Enabled: pointer.Bool(true),
					},
				},
			},
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
	}

	a := buildKitAction{}
	a.InjectLogger(log.Log)

	ok, err := trait.IntegrationAndKitHaveSameTraits(integration, kit)
	assert.Nil(t, err)
	assert.False(t, ok)
}

func TestHasMatchingTraits_KitSameTraitShouldBePicked(t *testing.T) {
	integration := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-integration",
		},
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Builder: &traitv1.BuilderTrait{
					Trait: traitv1.Trait{
						Enabled: pointer.Bool(true),
					},
					Properties: []string{
						"build-key1=build-value1",
					},
				},
			},
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
		Spec: v1.IntegrationKitSpec{
			Traits: v1.IntegrationKitTraits{
				Builder: &traitv1.BuilderTrait{
					Trait: traitv1.Trait{
						Enabled: pointer.Bool(true),
					},
					Properties: []string{
						"build-key1=build-value1",
					},
				},
			},
		},
	}

	a := buildKitAction{}
	a.InjectLogger(log.Log)

	ok, err := trait.IntegrationAndKitHaveSameTraits(integration, kit)
	assert.Nil(t, err)
	assert.True(t, ok)
}
