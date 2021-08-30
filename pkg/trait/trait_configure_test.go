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

package trait

import (
	"context"
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTraitConfigurationFromAnnotations(t *testing.T) {
	env := Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"trait.camel.apache.org/cron.concurrency-policy":    "annotated-policy",
					"trait.camel.apache.org/environment.container-meta": "true",
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: map[string]v1.TraitSpec{
					"cron": test.TraitSpecFromMap(t, map[string]interface{}{
						"fallback":          true,
						"concurrencyPolicy": "mypolicy",
					}),
				},
			},
		},
	}
	c := NewCatalog(context.Background(), nil)
	assert.NoError(t, c.configure(&env))
	assert.True(t, *c.GetTrait("cron").(*cronTrait).Fallback)
	assert.Equal(t, "annotated-policy", c.GetTrait("cron").(*cronTrait).ConcurrencyPolicy)
	assert.True(t, *c.GetTrait("environment").(*environmentTrait).ContainerMeta)
}

func TestFailOnWrongTraitAnnotations(t *testing.T) {
	env := Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"trait.camel.apache.org/cron.missing-property": "the-value",
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
			},
		},
	}
	c := NewCatalog(context.Background(), nil)
	assert.Error(t, c.configure(&env))
}

func TestTraitConfigurationOverrideRulesFromAnnotations(t *testing.T) {
	env := Environment{
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"trait.camel.apache.org/cron.components": "cmp2",
					"trait.camel.apache.org/cron.schedule":   "schedule2",
				},
			},
			Spec: v1.IntegrationPlatformSpec{
				Traits: map[string]v1.TraitSpec{
					"cron": test.TraitSpecFromMap(t, map[string]interface{}{
						"components": "cmp1",
						"schedule":   "schedule1",
					}),
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"trait.camel.apache.org/cron.components":         "cmp4",
					"trait.camel.apache.org/cron.concurrency-policy": "policy2",
				},
			},
			Spec: v1.IntegrationKitSpec{
				Traits: map[string]v1.TraitSpec{
					"cron": test.TraitSpecFromMap(t, map[string]interface{}{
						"components":        "cmp3",
						"concurrencyPolicy": "policy1",
					}),
				},
			},
		},
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"trait.camel.apache.org/cron.concurrency-policy": "policy4",
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: map[string]v1.TraitSpec{
					"cron": test.TraitSpecFromMap(t, map[string]interface{}{
						"concurrencyPolicy": "policy3",
					}),
				},
			},
		},
	}
	c := NewCatalog(context.Background(), nil)
	assert.NoError(t, c.configure(&env))
	assert.Equal(t, "schedule2", c.GetTrait("cron").(*cronTrait).Schedule)
	assert.Equal(t, "cmp4", c.GetTrait("cron").(*cronTrait).Components)
	assert.Equal(t, "policy4", c.GetTrait("cron").(*cronTrait).ConcurrencyPolicy)
}

func TestTraitListConfigurationFromAnnotations(t *testing.T) {
	env := Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"trait.camel.apache.org/jolokia.options":                  `["opt1", "opt2"]`,
					"trait.camel.apache.org/service-binding.service-bindings": `Binding:xxx`, // lenient
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
			},
		},
	}
	c := NewCatalog(context.Background(), nil)
	assert.NoError(t, c.configure(&env))
	assert.Equal(t, []string{"opt1", "opt2"}, c.GetTrait("jolokia").(*jolokiaTrait).Options)
	assert.Equal(t, []string{"Binding:xxx"}, c.GetTrait("service-binding").(*serviceBindingTrait).ServiceBindings)
}

func TestTraitSplitConfiguration(t *testing.T) {
	env := Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"trait.camel.apache.org/owner.target-labels": "[\"opt1\", \"opt2\"]",
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
			},
		},
	}
	c := NewCatalog(context.Background(), nil)
	assert.NoError(t, c.configure(&env))
	assert.Equal(t, []string{"opt1", "opt2"}, c.GetTrait("owner").(*ownerTrait).TargetLabels)
}
