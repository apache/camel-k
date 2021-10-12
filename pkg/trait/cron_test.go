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
	"strings"
	"testing"

	passert "github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/assert"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

func TestCronFromURI(t *testing.T) {
	tests := []struct {
		uri        string
		uri2       string
		uri3       string
		cron       string
		components string
	}{
		// Timer only
		{
			uri: "timer:tick?period=60000&delay=12", // invalid
		},
		{
			uri: "timer:tick?period=60000&repeatCount=10", // invalid
		},
		{
			uri:        "timer:tick?period=60000",
			cron:       "0/1 * * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=28800000",
			cron:       "0 0/8 * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=120000",
			cron:       "0/2 * * * ?",
			components: "timer",
		},
		{
			uri: "timer:tick?period=120001", // invalid
		},
		{
			uri:        "timer:tick?period=60000",
			cron:       "0/1 * * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=300000",
			cron:       "0/5 * * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=600000",
			cron:       "0/10 * * * ?",
			components: "timer",
		},
		{
			uri: "timer:tick?period=66000", // invalid
		},
		{
			uri:        "timer:tick?period=7200000",
			cron:       "0 0/2 * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=10800000",
			cron:       "0 0/3 * * ?",
			components: "timer",
		},
		{
			uri:        "timer:tick?period=86400000",
			cron:       "0 0 * * ?",
			components: "timer",
		},
		{
			uri: "timer:tick?period=10860000", // invalid
		},
		{
			uri:        "timer:tick?period=14400000",
			cron:       "0 0/4 * * ?",
			components: "timer",
		},

		// Quartz only
		{
			uri:        "quartz:trigger?cron=0 0 0/4 * * ?",
			cron:       "0 0/4 * * ?",
			components: "quartz",
		},
		{
			uri:        "quartz:trigger?cron=0+0+0/4+*+*+?",
			cron:       "0 0/4 * * ?",
			components: "quartz",
		},
		{
			uri: "quartz:trigger?cron=*+0+0/4+*+*+?", // invalid
		},
		{
			uri: "quartz:trigger?cron=0+0+0/4+*+*+?+2020", // invalid
		},
		{
			uri: "quartz:trigger?cron=1+0+0/4+*+*+?", // invalid
		},
		{
			uri: "quartz:trigger?cron=0+0+0/4+*+*+?&fireNow=true", // invalid
		},

		// Cron only
		{
			uri:        "cron:tab?schedule=1/2 * * * ?",
			cron:       "1/2 * * * ?",
			components: "cron",
		},
		{
			uri:        "cron:tab?schedule=0 0 0/4 * * ?",
			cron:       "0 0/4 * * ?",
			components: "cron",
		},
		{
			uri:        "cron:tab?schedule=0+0+0/4+*+*+?",
			cron:       "0 0/4 * * ?",
			components: "cron",
		},
		{
			uri: "cron:tab?schedule=*+0+0/4+*+*+?", // invalid
		},
		{
			uri:        "cron:tab?schedule=0+0,6+0/4+*+*+MON-THU",
			cron:       "0,6 0/4 * * MON-THU",
			components: "cron",
		},
		{
			uri: "cron:tab?schedule=0+0+0/4+*+*+?+2020", // invalid
		},
		{
			uri: "cron:tab?schedule=1+0+0/4+*+*+?", // invalid
		},

		// Mixed scenarios
		{
			uri:        "cron:tab?schedule=0/2 * * * ?",
			uri2:       "timer:tick?period=120000",
			cron:       "0/2 * * * ?",
			components: "cron,timer",
		},
		{
			uri:        "cron:tab?schedule=0 0/2 * * ?",
			uri2:       "timer:tick?period=7200000",
			uri3:       "quartz:trigger?cron=0 0 0/2 * * ? ?",
			cron:       "0 0/2 * * ?",
			components: "cron,timer,quartz",
		},
		{
			uri:  "cron:tab?schedule=1 0/2 * * ?",
			uri2: "timer:tick?period=7200000",
			uri3: "quartz:trigger?cron=0 0 0/2 * * ? ?",
			// invalid
		},
		{
			uri:  "cron:tab?schedule=0 0/2 * * ?",
			uri2: "timer:tick?period=10800000",
			uri3: "quartz:trigger?cron=0 0 0/2 * * ? ?",
			// invalid
		},
	}

	for _, test := range tests {
		thetest := test
		t.Run(thetest.uri, func(t *testing.T) {
			uris := []string{thetest.uri, thetest.uri2, thetest.uri3}
			filtered := make([]string, 0, len(uris))
			for _, uri := range uris {
				if uri != "" {
					filtered = append(filtered, uri)
				}
			}

			res := getCronForURIs(filtered)
			gotCron := ""
			if res != nil {
				gotCron = res.schedule
			}
			passert.Equal(t, gotCron, thetest.cron)

			gotComponents := ""
			if res != nil {
				gotComponents = strings.Join(res.components, ",")
			}
			passert.Equal(t, gotComponents, thetest.components)
		})
	}
}

func TestCronDeps(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "routes.java",
							Content: `from("cron:tab?schedule=0 0/2 * * ?").to("log:test")`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Resources: []v1.ResourceSpec{},
				Traits:    map[string]v1.TraitSpec{},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1.RegistrySpec{Address: "registry"},
				},
				Profile: v1.TraitProfileKnative,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	c, err := NewFakeClient("ns")
	assert.Nil(t, err)

	tc := NewCatalog(c)

	err = tc.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)

	ct := environment.GetTrait("cron").(*cronTrait)
	assert.NotNil(t, ct)
	assert.Nil(t, ct.Fallback)
	assert.True(t, util.StringSliceExists(environment.Integration.Status.Capabilities, v1.CapabilityCron))
	assert.Contains(t, environment.Integration.Status.Dependencies, "mvn:org.apache.camel.k:camel-k-cron")
}

func TestCronDepsFallback(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "routes.java",
							Content: `from("cron:tab?schedule=0 0/2 * * ?").to("log:test")`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Resources: []v1.ResourceSpec{},
				Traits: map[string]v1.TraitSpec{
					"cron": test.TraitSpecFromMap(t, map[string]interface{}{
						"fallback": true,
					}),
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1.RegistrySpec{Address: "registry"},
				},
				Profile: v1.TraitProfileKnative,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	c, err := NewFakeClient("ns")
	assert.Nil(t, err)

	tc := NewCatalog(c)

	err = tc.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)

	ct := environment.GetTrait("cron").(*cronTrait)
	assert.NotNil(t, ct)
	assert.NotNil(t, ct.Fallback)
	assert.True(t, util.StringSliceExists(environment.Integration.Status.Capabilities, v1.CapabilityCron))
	assert.Contains(t, environment.Integration.Status.Dependencies, "camel:quartz")
	assert.Contains(t, environment.Integration.Status.Dependencies, "mvn:org.apache.camel.k:camel-k-cron")
}

func TestCronWithActiveDeadline(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "routes.java",
							Content: `from("cron:tab?schedule=0 0/2 * * ?").to("log:test")`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Resources: []v1.ResourceSpec{},
				Traits: map[string]v1.TraitSpec{
					"cron": test.TraitSpecFromMap(t, map[string]interface{}{
						"activeDeadlineSeconds": 120,
					}),
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform:       &v1.IntegrationPlatform{},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	c, err := NewFakeClient("ns")
	assert.Nil(t, err)

	tc := NewCatalog(c)

	err = tc.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)

	ct := environment.GetTrait("cron").(*cronTrait)
	assert.NotNil(t, ct)
	assert.Nil(t, ct.Fallback)
	assert.Contains(t, environment.Interceptors, "cron")

	cronJob := environment.Resources.GetCronJob(func(job *batchv1beta1.CronJob) bool { return true })
	assert.NotNil(t, cronJob)

	assert.NotNil(t, cronJob.Spec.JobTemplate.Spec.ActiveDeadlineSeconds)
	assert.EqualValues(t, *cronJob.Spec.JobTemplate.Spec.ActiveDeadlineSeconds, 120)

	assert.NotNil(t, cronJob.Spec.JobTemplate.Spec.BackoffLimit)
	assert.EqualValues(t, *cronJob.Spec.JobTemplate.Spec.BackoffLimit, 2)
}

func TestCronWithBackoffLimit(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	traitCatalog := NewCatalog(nil)

	environment := Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKnative,
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "routes.java",
							Content: `from("cron:tab?schedule=0 0/2 * * ?").to("log:test")`,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Resources: []v1.ResourceSpec{},
				Traits: map[string]v1.TraitSpec{
					"cron": test.TraitSpecFromMap(t, map[string]interface{}{
						"backoffLimit": 5,
					}),
				},
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform:       &v1.IntegrationPlatform{},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
	environment.Platform.ResyncStatusFullConfig()

	c, err := NewFakeClient("ns")
	assert.Nil(t, err)

	tc := NewCatalog(c)

	err = tc.apply(&environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)

	ct := environment.GetTrait("cron").(*cronTrait)
	assert.NotNil(t, ct)
	assert.Nil(t, ct.Fallback)
	assert.Contains(t, environment.Interceptors, "cron")

	cronJob := environment.Resources.GetCronJob(func(job *batchv1beta1.CronJob) bool { return true })
	assert.NotNil(t, cronJob)

	assert.NotNil(t, cronJob.Spec.JobTemplate.Spec.ActiveDeadlineSeconds)
	assert.EqualValues(t, *cronJob.Spec.JobTemplate.Spec.ActiveDeadlineSeconds, 60)

	assert.NotNil(t, cronJob.Spec.JobTemplate.Spec.BackoffLimit)
	assert.EqualValues(t, *cronJob.Spec.JobTemplate.Spec.BackoffLimit, 5)
}
