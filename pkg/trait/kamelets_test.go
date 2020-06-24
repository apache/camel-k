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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	"github.com/apache/camel-k/pkg/util/camel"

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

func TestConfigurationNoKameletsUsed(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: timer:tick
    steps:
    - to: log:info
`)
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.False(t, enabled)
	assert.Equal(t, "", trait.List)
}

func TestConfigurationWithKamelets(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:c1
    steps:
    - to: kamelet:c2
    - to: telegram:bots
    - to: kamelet://c0?prop=x
    - to: kamelet://complex-.-.-1a?prop=x&prop2
    - to: kamelet://complex-.-.-1b
    - to: kamelet:complex-.-.-1b
    - to: kamelet://complex-.-.-1b/a
    - to: kamelet://complex-.-.-1c/b
`)
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"c0", "c1", "c2", "complex-.-.-1a", "complex-.-.-1b", "complex-.-.-1c"}, trait.getKamelets())
}

func TestKameletLookup(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:timer
    steps:
    - to: log:info
`, &v1alpha1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1alpha1.KameletSpec{
			Flow: &v1.Flow{
				"from": map[string]interface{}{
					"uri": "timer:tick",
				},
			},
		},
	})
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"timer"}, trait.getKamelets())

	err = trait.Apply(environment)
	assert.NoError(t, err)
	cm := environment.Resources.GetConfigMap(func(_ *corev1.ConfigMap) bool { return true })
	assert.NotNil(t, cm)
	assert.Equal(t, "it-kamelet-timer-flow", cm.Name)
	assert.Equal(t, "test", cm.Namespace)

	assert.Len(t, environment.Integration.Status.GeneratedSources, 1)
	source := environment.Integration.Status.GeneratedSources[0]
	assert.Equal(t, "timer.yaml", source.Name)
	assert.Equal(t, "kamelet", string(source.Type))
}

func TestKameletSecondarySourcesLookup(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:timer
    steps:
    - to: log:info
`, &v1alpha1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1alpha1.KameletSpec{
			Flow: &v1.Flow{
				"from": map[string]interface{}{
					"uri": "timer:tick",
				},
			},
			Sources: []v1.SourceSpec{
				{
					DataSpec: v1.DataSpec{
						Name:    "support.groovy",
						Content: "from('xxx:xxx').('to:log:info')",
					},
					Language: v1.LanguageGroovy,
				},
			},
		},
	})
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"timer"}, trait.getKamelets())

	err = trait.Apply(environment)
	assert.NoError(t, err)
	cmFlow := environment.Resources.GetConfigMap(func(c *corev1.ConfigMap) bool { return c.Name == "it-kamelet-timer-flow" })
	assert.NotNil(t, cmFlow)
	cmRes := environment.Resources.GetConfigMap(func(c *corev1.ConfigMap) bool { return c.Name == "it-kamelet-timer-000" })
	assert.NotNil(t, cmRes)

	assert.Len(t, environment.Integration.Status.GeneratedSources, 2)

	flowSource := environment.Integration.Status.GeneratedSources[0]
	assert.Equal(t, "timer.yaml", flowSource.Name)
	assert.Equal(t, "kamelet", string(flowSource.Type))
	assert.Equal(t, "it-kamelet-timer-flow", flowSource.ContentRef)
	assert.Equal(t, "content", flowSource.ContentKey)

	supportSource := environment.Integration.Status.GeneratedSources[1]
	assert.Equal(t, "support.groovy", supportSource.Name)
	assert.Equal(t, "", string(supportSource.Type))
	assert.Equal(t, "it-kamelet-timer-000", supportSource.ContentRef)
	assert.Equal(t, "content", supportSource.ContentKey)
}

func TestNonYAMLKameletLookup(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:timer
    steps:
    - to: log:info
`, &v1alpha1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1alpha1.KameletSpec{
			Sources: []v1.SourceSpec{
				{
					DataSpec: v1.DataSpec{
						Name:    "mykamelet.groovy",
						Content: `from("timer").to("log:info")`,
					},
					Type: v1.SourceTypeKamelet,
				},
			},
		},
	})
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"timer"}, trait.getKamelets())

	err = trait.Apply(environment)
	assert.NoError(t, err)
	cm := environment.Resources.GetConfigMap(func(_ *corev1.ConfigMap) bool { return true })
	assert.NotNil(t, cm)
	assert.Equal(t, "it-kamelet-timer-000", cm.Name)
	assert.Equal(t, "test", cm.Namespace)

	assert.Len(t, environment.Integration.Status.GeneratedSources, 1)
	source := environment.Integration.Status.GeneratedSources[0]
	assert.Equal(t, "timer.groovy", source.Name)
	assert.Equal(t, "kamelet", string(source.Type))
}

func TestErrorMultipleKameletSources(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:timer
    steps:
    - to: log:info
`, &v1alpha1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1alpha1.KameletSpec{
			Sources: []v1.SourceSpec{
				{
					DataSpec: v1.DataSpec{
						Name:    "mykamelet.groovy",
						Content: `from("timer").to("log:info")`,
					},
					Type: v1.SourceTypeKamelet,
				},
			},
			Flow: &v1.Flow{
				"from": map[string]interface{}{
					"uri": "timer:tick",
				},
			},
		},
	})
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"timer"}, trait.getKamelets())

	err = trait.Apply(environment)
	assert.Error(t, err)
}

func createKameletsTestEnvironment(flow string, objects ...runtime.Object) (*kameletsTrait, *Environment) {
	catalog, _ := camel.DefaultCatalog()

	client, _ := test.NewFakeClient(objects...)
	trait := newKameletsTrait().(*kameletsTrait)
	trait.Ctx = context.TODO()
	trait.Client = client

	environment := &Environment{
		Catalog:      NewCatalog(context.TODO(), nil),
		CamelCatalog: catalog,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "it",
			},
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:    "flow.yaml",
							Content: flow,
						},
						Language: v1.LanguageYaml,
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
		Resources: kubernetes.NewCollection(),
	}

	return trait, environment
}
