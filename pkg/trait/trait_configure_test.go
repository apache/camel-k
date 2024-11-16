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
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
)

func TestTraitConfiguration(t *testing.T) {
	env := Environment{
		Integration: &v1.Integration{
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: v1.Traits{
					Logging: &traitv1.LoggingTrait{
						JSON:            ptr.To(true),
						JSONPrettyPrint: ptr.To(false),
						Level:           "DEBUG",
					},
					Service: &traitv1.ServiceTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						Auto: ptr.To(true),
					},
				},
			},
		},
	}
	c := NewCatalog(nil)
	require.NoError(t, c.Configure(&env))
	logging, ok := c.GetTrait("logging").(*loggingTrait)
	require.True(t, ok)
	assert.True(t, *logging.JSON)
	assert.False(t, *logging.JSONPrettyPrint)
	assert.Equal(t, "DEBUG", logging.Level)
	service, ok := c.GetTrait("service").(*serviceTrait)
	require.True(t, ok)
	assert.True(t, *service.Enabled)
	assert.True(t, *service.Auto)
}

func TestTraitConfigurationFromAnnotations(t *testing.T) {
	env := Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.TraitAnnotationPrefix + "cron.concurrency-policy":    "annotated-policy",
					v1.TraitAnnotationPrefix + "environment.container-meta": boolean.TrueString,
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
				Traits: v1.Traits{
					Cron: &traitv1.CronTrait{
						Fallback:          ptr.To(true),
						ConcurrencyPolicy: "mypolicy",
					},
				},
			},
		},
	}
	c := NewCatalog(nil)
	require.NoError(t, c.Configure(&env))
	ct, _ := c.GetTrait("cron").(*cronTrait)
	assert.True(t, *ct.Fallback)
	assert.Equal(t, "annotated-policy", ct.ConcurrencyPolicy)
	et, _ := c.GetTrait("environment").(*environmentTrait)
	assert.True(t, *et.ContainerMeta)
}

func TestFailOnWrongTraitAnnotations(t *testing.T) {
	env := Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.TraitAnnotationPrefix + "cron.missing-property": "the-value",
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
			},
		},
	}
	c := NewCatalog(nil)
	assert.Error(t, c.Configure(&env))
}

func TestTraitConfigurationOverrideRulesFromAnnotations(t *testing.T) {
	env := Environment{
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.TraitAnnotationPrefix + "cron.components": "cmp2",
					v1.TraitAnnotationPrefix + "cron.schedule":   "schedule2",
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
					v1.TraitAnnotationPrefix + "cron.components":         "cmp3",
					v1.TraitAnnotationPrefix + "cron.concurrency-policy": "policy2",
					v1.TraitAnnotationPrefix + "builder.verbose":         boolean.TrueString,
				},
			},
			Spec: v1.IntegrationKitSpec{
				Traits: v1.IntegrationKitTraits{
					Builder: &traitv1.BuilderTrait{
						Verbose: ptr.To(false),
					},
				},
			},
		},
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.TraitAnnotationPrefix + "cron.components":         "cmp4",
					v1.TraitAnnotationPrefix + "cron.concurrency-policy": "policy4",
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
	require.NoError(t, c.Configure(&env))
	ct, _ := c.GetTrait("cron").(*cronTrait)
	assert.Equal(t, "schedule2", ct.Schedule)
	assert.Equal(t, "cmp4", ct.Components)
	assert.Equal(t, "policy4", ct.ConcurrencyPolicy)
	bt, _ := c.GetTrait("builder").(*builderTrait)
	assert.True(t, *bt.Verbose)
}

func TestTraitListConfigurationFromAnnotations(t *testing.T) {
	env := Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.TraitAnnotationPrefix + "jolokia.options":          `["opt1", "opt2"]`,
					v1.TraitAnnotationPrefix + "service-binding.services": `Binding:xxx`, // lenient
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
			},
		},
	}
	c := NewCatalog(nil)
	require.NoError(t, c.Configure(&env))
	jt, _ := c.GetTrait("jolokia").(*jolokiaTrait)
	assert.Equal(t, []string{"opt1", "opt2"}, jt.Options)
	sbt, _ := c.GetTrait("service-binding").(*serviceBindingTrait)
	assert.Equal(t, []string{"Binding:xxx"}, sbt.Services)
}

func TestTraitSplitConfiguration(t *testing.T) {
	env := Environment{
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					v1.TraitAnnotationPrefix + "owner.target-labels": "[\"opt1\", \"opt2\"]",
				},
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
			},
		},
	}
	c := NewCatalog(nil)
	require.NoError(t, c.Configure(&env))
	ot, _ := c.GetTrait("owner").(*ownerTrait)
	assert.Equal(t, []string{"opt1", "opt2"}, ot.TargetLabels)
}
