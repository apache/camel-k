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
	"encoding/json"
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	assert.Equal(t, []string{"c0", "c1", "c2", "complex-.-.-1a", "complex-.-.-1b", "complex-.-.-1c"}, trait.getKameletKeys())
	assert.Equal(t, []configurationKey{
		newConfigurationKey("c0", ""),
		newConfigurationKey("c1", ""),
		newConfigurationKey("c2", ""),
		newConfigurationKey("complex-.-.-1a", ""),
		newConfigurationKey("complex-.-.-1b", ""),
		newConfigurationKey("complex-.-.-1b", "a"),
		newConfigurationKey("complex-.-.-1c", ""),
		newConfigurationKey("complex-.-.-1c", "b"),
	}, trait.getConfigurationKeys())
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
			Flow: marshalOrFail(map[string]interface{}{
				"from": map[string]interface{}{
					"uri": "timer:tick",
				},
			}),
			Dependencies: []string{
				"camel:timer",
				"camel:log",
			},
		},
		Status: v1alpha1.KameletStatus{Phase: v1alpha1.KameletPhaseReady},
	})
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"timer"}, trait.getKameletKeys())

	err = trait.Apply(environment)
	assert.NoError(t, err)
	cm := environment.Resources.GetConfigMap(func(_ *corev1.ConfigMap) bool { return true })
	assert.NotNil(t, cm)
	assert.Equal(t, "it-kamelet-timer-template", cm.Name)
	assert.Equal(t, "test", cm.Namespace)

	assert.Len(t, environment.Integration.Status.GeneratedSources, 1)
	source := environment.Integration.Status.GeneratedSources[0]
	assert.Equal(t, "timer.yaml", source.Name)
	assert.Equal(t, "", string(source.Type))

	assert.Equal(t, []string{"camel:log", "camel:timer"}, environment.Integration.Status.Dependencies)
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
			Flow: marshalOrFail(map[string]interface{}{
				"from": map[string]interface{}{
					"uri": "timer:tick",
				},
			}),
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
		Status: v1alpha1.KameletStatus{Phase: v1alpha1.KameletPhaseReady},
	})
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"timer"}, trait.getKameletKeys())

	err = trait.Apply(environment)
	assert.NoError(t, err)
	cmFlow := environment.Resources.GetConfigMap(func(c *corev1.ConfigMap) bool { return c.Name == "it-kamelet-timer-template" })
	assert.NotNil(t, cmFlow)
	cmRes := environment.Resources.GetConfigMap(func(c *corev1.ConfigMap) bool { return c.Name == "it-kamelet-timer-000" })
	assert.NotNil(t, cmRes)

	assert.Len(t, environment.Integration.Status.GeneratedSources, 2)

	flowSource := environment.Integration.Status.GeneratedSources[0]
	assert.Equal(t, "timer.yaml", flowSource.Name)
	assert.Equal(t, "", string(flowSource.Type))
	assert.Equal(t, "it-kamelet-timer-template", flowSource.ContentRef)
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
					Type: v1.SourceTypeTemplate,
				},
			},
		},
		Status: v1alpha1.KameletStatus{Phase: v1alpha1.KameletPhaseReady},
	})
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"timer"}, trait.getKameletKeys())

	err = trait.Apply(environment)
	assert.NoError(t, err)
	cm := environment.Resources.GetConfigMap(func(_ *corev1.ConfigMap) bool { return true })
	assert.NotNil(t, cm)
	assert.Equal(t, "it-kamelet-timer-000", cm.Name)
	assert.Equal(t, "test", cm.Namespace)

	assert.Len(t, environment.Integration.Status.GeneratedSources, 1)
	source := environment.Integration.Status.GeneratedSources[0]
	assert.Equal(t, "timer.groovy", source.Name)
	assert.Equal(t, "template", string(source.Type))
}

func TestMultipleKamelets(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:timer
    steps:
    - to: kamelet:logger
    - to: kamelet:logger
`, &v1alpha1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1alpha1.KameletSpec{
			Flow: marshalOrFail(map[string]interface{}{
				"from": map[string]interface{}{
					"uri": "timer:tick",
				},
			}),
			Sources: []v1.SourceSpec{
				{
					DataSpec: v1.DataSpec{
						Name:    "support.groovy",
						Content: "from('xxx:xxx').('to:log:info')",
					},
					Language: v1.LanguageGroovy,
				},
			},
			Dependencies: []string{
				"camel:timer",
				"camel:xxx",
			},
		},
		Status: v1alpha1.KameletStatus{Phase: v1alpha1.KameletPhaseReady},
	}, &v1alpha1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "logger",
		},
		Spec: v1alpha1.KameletSpec{
			Flow: marshalOrFail(map[string]interface{}{
				"from": map[string]interface{}{
					"uri": "tbd:endpoint",
					"steps": []interface{}{
						map[string]interface{}{
							"to": map[string]interface{}{
								"uri": "log:info",
							},
						},
					},
				},
			}),
			Dependencies: []string{
				"camel:log",
				"camel:tbd",
			},
		},
		Status: v1alpha1.KameletStatus{Phase: v1alpha1.KameletPhaseReady},
	})
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"logger", "timer"}, trait.getKameletKeys())

	err = trait.Apply(environment)
	assert.NoError(t, err)

	cmFlow := environment.Resources.GetConfigMap(func(c *corev1.ConfigMap) bool { return c.Name == "it-kamelet-timer-template" })
	assert.NotNil(t, cmFlow)
	cmRes := environment.Resources.GetConfigMap(func(c *corev1.ConfigMap) bool { return c.Name == "it-kamelet-timer-000" })
	assert.NotNil(t, cmRes)
	cmFlow2 := environment.Resources.GetConfigMap(func(c *corev1.ConfigMap) bool { return c.Name == "it-kamelet-logger-template" })
	assert.NotNil(t, cmFlow2)

	assert.Len(t, environment.Integration.Status.GeneratedSources, 3)

	flowSource2 := environment.Integration.Status.GeneratedSources[0]
	assert.Equal(t, "logger.yaml", flowSource2.Name)
	assert.Equal(t, "", string(flowSource2.Type))
	assert.Equal(t, "it-kamelet-logger-template", flowSource2.ContentRef)
	assert.Equal(t, "content", flowSource2.ContentKey)

	flowSource := environment.Integration.Status.GeneratedSources[1]
	assert.Equal(t, "timer.yaml", flowSource.Name)
	assert.Equal(t, "", string(flowSource.Type))
	assert.Equal(t, "it-kamelet-timer-template", flowSource.ContentRef)
	assert.Equal(t, "content", flowSource.ContentKey)

	supportSource := environment.Integration.Status.GeneratedSources[2]
	assert.Equal(t, "support.groovy", supportSource.Name)
	assert.Equal(t, "", string(supportSource.Type))
	assert.Equal(t, "it-kamelet-timer-000", supportSource.ContentRef)
	assert.Equal(t, "content", supportSource.ContentKey)

	assert.Equal(t, []string{"camel:log", "camel:tbd", "camel:timer", "camel:xxx"}, environment.Integration.Status.Dependencies)
}

func TestKameletConfigLookup(t *testing.T) {
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
			Flow: marshalOrFail(map[string]interface{}{
				"from": map[string]interface{}{
					"uri": "timer:tick",
				},
			}),
			Dependencies: []string{
				"camel:timer",
				"camel:log",
			},
		},
		Status: v1alpha1.KameletStatus{Phase: v1alpha1.KameletPhaseReady},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret",
			Labels: map[string]string{
				"camel.apache.org/kamelet": "timer",
			},
		},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret2",
			Labels: map[string]string{
				"camel.apache.org/kamelet":               "timer",
				"camel.apache.org/kamelet.configuration": "id2",
			},
		},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret3",
			Labels: map[string]string{
				"camel.apache.org/kamelet": "timer",
			},
		},
	})
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"timer"}, trait.getKameletKeys())
	assert.Equal(t, []configurationKey{newConfigurationKey("timer", "")}, trait.getConfigurationKeys())

	err = trait.Apply(environment)
	assert.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Configuration, 2)
	assert.Contains(t, environment.Integration.Status.Configuration, v1.ConfigurationSpec{Type: "secret", Value: "my-secret"})
	assert.NotContains(t, environment.Integration.Status.Configuration, v1.ConfigurationSpec{Type: "secret", Value: "my-secret2"})
	assert.Contains(t, environment.Integration.Status.Configuration, v1.ConfigurationSpec{Type: "secret", Value: "my-secret3"})
}

func TestKameletNamedConfigLookup(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:timer/id2
    steps:
    - to: log:info
`, &v1alpha1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1alpha1.KameletSpec{
			Flow: marshalOrFail(map[string]interface{}{
				"from": map[string]interface{}{
					"uri": "timer:tick",
				},
			}),
			Dependencies: []string{
				"camel:timer",
				"camel:log",
			},
		},
		Status: v1alpha1.KameletStatus{Phase: v1alpha1.KameletPhaseReady},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret",
			Labels: map[string]string{
				"camel.apache.org/kamelet": "timer",
			},
		},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret2",
			Labels: map[string]string{
				"camel.apache.org/kamelet":               "timer",
				"camel.apache.org/kamelet.configuration": "id2",
			},
		},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret3",
			Labels: map[string]string{
				"camel.apache.org/kamelet":               "timer",
				"camel.apache.org/kamelet.configuration": "id3",
			},
		},
	})
	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, []string{"timer"}, trait.getKameletKeys())
	assert.Equal(t, []configurationKey{
		newConfigurationKey("timer", ""),
		newConfigurationKey("timer", "id2"),
	}, trait.getConfigurationKeys())

	err = trait.Apply(environment)
	assert.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Configuration, 2)
	assert.Contains(t, environment.Integration.Status.Configuration, v1.ConfigurationSpec{Type: "secret", Value: "my-secret"})
	assert.Contains(t, environment.Integration.Status.Configuration, v1.ConfigurationSpec{Type: "secret", Value: "my-secret2"})
	assert.NotContains(t, environment.Integration.Status.Configuration, v1.ConfigurationSpec{Type: "secret", Value: "my-secret3"})
}

func TestKameletConditionFalse(t *testing.T) {
	flow := `
- from:
    uri: kamelet:timer
    steps:
    - to: kamelet:none
`
	trait, environment := createKameletsTestEnvironment(
		flow,
		&v1alpha1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "timer",
			},
			Spec: v1alpha1.KameletSpec{
				Flow: marshalOrFail(map[string]interface{}{
					"from": map[string]interface{}{
						"uri": "timer:tick",
					},
				}),
			},
			Status: v1alpha1.KameletStatus{Phase: v1alpha1.KameletPhaseReady},
		})

	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)

	err = trait.Apply(environment)
	assert.Error(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	cond := environment.Integration.Status.GetCondition(v1.IntegrationConditionKameletsAvailable)
	assert.Equal(t, corev1.ConditionFalse, cond.Status)
	assert.Equal(t, v1.IntegrationConditionKameletsAvailableReason, cond.Reason)
	assert.Contains(t, cond.Message, "timer found")
	assert.Contains(t, cond.Message, "none not found")
}

func TestKameletConditionTrue(t *testing.T) {
	flow := `
- from:
    uri: kamelet:timer
    steps:
    - to: kamelet:none
`
	trait, environment := createKameletsTestEnvironment(
		flow,
		&v1alpha1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "timer",
			},
			Spec: v1alpha1.KameletSpec{
				Flow: marshalOrFail(map[string]interface{}{
					"from": map[string]interface{}{
						"uri": "timer:tick",
					},
				}),
			},
			Status: v1alpha1.KameletStatus{Phase: v1alpha1.KameletPhaseReady},
		},
		&v1alpha1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "none",
			},
			Spec: v1alpha1.KameletSpec{
				Flow: marshalOrFail(map[string]interface{}{
					"from": map[string]interface{}{
						"uri": "timer:tick",
					},
				}),
			},
			Status: v1alpha1.KameletStatus{Phase: v1alpha1.KameletPhaseReady},
		})

	enabled, err := trait.Configure(environment)
	assert.NoError(t, err)
	assert.True(t, enabled)

	err = trait.Apply(environment)
	assert.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	cond := environment.Integration.Status.GetCondition(v1.IntegrationConditionKameletsAvailable)
	assert.Equal(t, corev1.ConditionTrue, cond.Status)
	assert.Equal(t, v1.IntegrationConditionKameletsAvailableReason, cond.Reason)
	assert.Contains(t, cond.Message, "none,timer found")
}

func createKameletsTestEnvironment(flow string, objects ...runtime.Object) (*kameletsTrait, *Environment) {
	catalog, _ := camel.DefaultCatalog()

	client, _ := test.NewFakeClient(objects...)
	trait := newKameletsTrait().(*kameletsTrait)
	trait.Ctx = context.TODO()
	trait.Client = client

	environment := &Environment{
		Catalog:      NewCatalog(context.TODO(), client),
		Client:       client,
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

func marshalOrFail(flow map[string]interface{}) *v1.Flow {
	data, err := json.Marshal(flow)
	if err != nil {
		panic(err)
	}
	f := v1.Flow{RawMessage: data}
	return &f
}
