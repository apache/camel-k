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
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
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
				Traits: v1.Traits{
					Cron: &traitv1.CronTrait{
						Fallback:          pointer.Bool(true),
						ConcurrencyPolicy: "mypolicy",
					},
				},
			},
		},
	}
	c := NewCatalog(nil)
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
	c := NewCatalog(nil)
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
				Traits: v1.Traits{
					Cron: &traitv1.CronTrait{
						Components:        "cmp1",
						Schedule:          "schedule1",
						ConcurrencyPolicy: "policy1",
					},
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"trait.camel.apache.org/cron.components":         "cmp3",
					"trait.camel.apache.org/cron.concurrency-policy": "policy2",
					"trait.camel.apache.org/builder.verbose":         "true",
				},
			},
			Spec: v1.IntegrationKitSpec{
				Traits: v1.IntegrationKitTraits{
					Builder: &traitv1.BuilderTrait{
						Verbose: pointer.Bool(false),
					},
				},
			},
		},
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"trait.camel.apache.org/cron.components":         "cmp4",
					"trait.camel.apache.org/cron.concurrency-policy": "policy4",
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: v1.Traits{
					Cron: &traitv1.CronTrait{
						ConcurrencyPolicy: "policy3",
					},
				},
			},
		},
	}
	c := NewCatalog(nil)
	assert.NoError(t, c.configure(&env))
	assert.Equal(t, "schedule2", c.GetTrait("cron").(*cronTrait).Schedule)
	assert.Equal(t, "cmp4", c.GetTrait("cron").(*cronTrait).Components)
	assert.Equal(t, "policy4", c.GetTrait("cron").(*cronTrait).ConcurrencyPolicy)
	assert.Equal(t, pointer.Bool(true), c.GetTrait("builder").(*builderTrait).Verbose)
}

func TestTraitListConfigurationFromAnnotations(t *testing.T) {
	env := Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"trait.camel.apache.org/jolokia.options":          `["opt1", "opt2"]`,
					"trait.camel.apache.org/service-binding.services": `Binding:xxx`, // lenient
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
			},
		},
	}
	c := NewCatalog(nil)
	assert.NoError(t, c.configure(&env))
	assert.Equal(t, []string{"opt1", "opt2"}, c.GetTrait("jolokia").(*jolokiaTrait).Options)
	assert.Equal(t, []string{"Binding:xxx"}, c.GetTrait("service-binding").(*serviceBindingTrait).Services)
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
	c := NewCatalog(nil)
	assert.NoError(t, c.configure(&env))
	assert.Equal(t, []string{"opt1", "opt2"}, c.GetTrait("owner").(*ownerTrait).TargetLabels)
}
