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
	"encoding/json"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

func TestConfigurationNoKameletsUsed(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: timer:tick
    steps:
    - to: log:info
`)
	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.False(t, enabled)
	assert.Nil(t, condition)
	assert.Equal(t, "", trait.List)
}

func TestConfigurationWithKamelets(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:c1
    steps:
    - to: kamelet:c2
    - to: kamelet:c3?kameletVersion=v1
    - to: telegram:bots
    - to: kamelet://c0?prop=x
    - to: kamelet://complex-.-.-1a?prop=x&prop2
    - to: kamelet://complex-.-.-1b
    - to: kamelet:complex-.-.-1b
    - to: kamelet://complex-.-.-1b/a
    - to: kamelet://complex-.-.-1c/b
`)
	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)
	assert.Equal(t, []string{"c0", "c1", "c2", "c3", "complex-.-.-1a", "complex-.-.-1b", "complex-.-.-1c"}, trait.getKameletKeys(false))
	assert.Equal(t, []string{"c0", "c1", "c2", "c3-v1", "complex-.-.-1a", "complex-.-.-1b", "complex-.-.-1c"}, trait.getKameletKeys(true))
}

func TestKameletLookup(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:timer
    steps:
    - to: log:info
`, &v1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1.KameletSpec{
			KameletSpecBase: v1.KameletSpecBase{
				Template: templateOrFail(map[string]interface{}{
					"from": map[string]interface{}{
						"uri": "timer:tick",
					},
				}),
				Dependencies: []string{
					"camel:timer",
					"camel:log",
				},
			},
		},
	})
	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)
	assert.Equal(t, []string{"timer"}, trait.getKameletKeys(false))

	err = trait.Apply(environment)
	require.NoError(t, err)
	cm := environment.Resources.GetConfigMap(func(_ *corev1.ConfigMap) bool { return true })
	assert.NotNil(t, cm)
	assert.Equal(t, "kamelets-bundle-it-001", cm.Name)
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
`, &v1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1.KameletSpec{
			KameletSpecBase: v1.KameletSpecBase{
				Template: templateOrFail(map[string]interface{}{
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
		},
	})
	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)
	assert.Equal(t, []string{"timer"}, trait.getKameletKeys(false))

	err = trait.Apply(environment)
	require.NoError(t, err)
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
`, &v1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1.KameletSpec{
			KameletSpecBase: v1.KameletSpecBase{
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
		},
	})
	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)
	assert.Equal(t, []string{"timer"}, trait.getKameletKeys(false))

	err = trait.Apply(environment)
	require.NoError(t, err)
	cm := environment.Resources.GetConfigMap(func(_ *corev1.ConfigMap) bool { return true })
	assert.NotNil(t, cm)
	assert.Equal(t, "kamelets-bundle-it-001", cm.Name)
	assert.Equal(t, "test", cm.Namespace)

	assert.Len(t, environment.Integration.Status.GeneratedSources, 1)
	source := environment.Integration.Status.GeneratedSources[0]
	assert.Equal(t, "timer.groovy", source.Name)
	assert.Equal(t, "template", string(source.Type))
}

func TestMultipleKamelets(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:timer?kameletVersion=v1
    steps:
    - to: kamelet:logger
`, &v1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1.KameletSpec{
			KameletSpecBase: v1.KameletSpecBase{
				Template: templateOrFail(map[string]interface{}{
					"from": map[string]interface{}{
						"uri": "timer:tick",
					},
				}),
				Dependencies: []string{
					"camel:timer",
					"camel:xxx",
				},
			},
			Versions: map[string]v1.KameletSpecBase{
				"v1": {
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
						"camel:xxx-2",
					},
				},
			},
		},
	}, &v1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "logger",
		},
		Spec: v1.KameletSpec{
			KameletSpecBase: v1.KameletSpecBase{
				Template: templateOrFail(map[string]interface{}{
					"from": map[string]interface{}{
						"uri": "tbd:endpoint",
						"steps": []interface{}{
							map[string]interface{}{
								"to": map[string]interface{}{
									"uri": "log:info?option=main",
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
			Versions: map[string]v1.KameletSpecBase{
				"v2": {
					Template: templateOrFail(map[string]interface{}{
						"from": map[string]interface{}{
							"uri": "tbd:endpoint",
							"steps": []interface{}{
								map[string]interface{}{
									"to": map[string]interface{}{
										"uri": "log:info?option=version2",
									},
								},
							},
						},
					}),
					Dependencies: []string{
						"camel:log",
						"camel:tbd-2",
					},
				},
			},
		},
	})
	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)
	assert.Equal(t, "logger,timer?kameletVersion=v1", trait.List)
	assert.Equal(t, []string{"logger", "timer"}, trait.getKameletKeys(false))
	assert.Equal(t, []string{"logger", "timer-v1"}, trait.getKameletKeys(true))

	err = trait.Apply(environment)
	require.NoError(t, err)

	cmFlowTimerSource := environment.Resources.GetConfigMap(func(c *corev1.ConfigMap) bool { return c.Name == "it-kamelet-timer-000" })
	assert.NotNil(t, cmFlowTimerSource)
	assert.Contains(t, cmFlowTimerSource.Data[contentKey], "from('xxx:xxx').('to:log:info')")
	cmFlowMissing := environment.Resources.GetConfigMap(func(c *corev1.ConfigMap) bool { return c.Name == "it-kamelet-timer-template" })
	assert.Nil(t, cmFlowMissing)
	cmFlowLoggerTemplateMain := environment.Resources.GetConfigMap(func(c *corev1.ConfigMap) bool { return c.Name == "it-kamelet-logger-template" })
	assert.NotNil(t, cmFlowLoggerTemplateMain)
	assert.Contains(t, cmFlowLoggerTemplateMain.Data[contentKey], "log:info?option=main")

	assert.Len(t, environment.Integration.Status.GeneratedSources, 2)

	expectedFlowSourceTimerV1 := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:       "support.groovy",
			ContentRef: "it-kamelet-timer-000",
			ContentKey: "content",
		},
		Language:    v1.LanguageGroovy,
		FromKamelet: true,
	}

	expectedFlowSinkLoggerMain := v1.SourceSpec{
		DataSpec: v1.DataSpec{
			Name:       "logger.yaml",
			ContentRef: "it-kamelet-logger-template",
			ContentKey: "content",
		},
		Language:    v1.LanguageYaml,
		FromKamelet: true,
	}

	assert.Contains(t, environment.Integration.Status.GeneratedSources, expectedFlowSourceTimerV1, expectedFlowSinkLoggerMain)

	assert.Contains(t, environment.Integration.Status.Dependencies,
		"camel:log", "camel:tbd", "camel:timer", "camel:xxx", "camel:xxx-2")
}

func TestKameletConfigLookup(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:timer
    steps:
    - to: log:info
`, &v1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1.KameletSpec{
			KameletSpecBase: v1.KameletSpecBase{
				Template: templateOrFail(map[string]interface{}{
					"from": map[string]interface{}{
						"uri": "timer:tick",
					},
				}),
				Dependencies: []string{
					"camel:timer",
					"camel:log",
				},
			},
		},
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
	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)
	assert.Equal(t, []string{"timer"}, trait.getKameletKeys(false))
}

func TestKameletNamedConfigLookup(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(`
- from:
    uri: kamelet:timer/id2
    steps:
    - to: log:info
`, &v1.Kamelet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "timer",
		},
		Spec: v1.KameletSpec{
			KameletSpecBase: v1.KameletSpecBase{
				Template: templateOrFail(map[string]interface{}{
					"from": map[string]interface{}{
						"uri": "timer:tick",
					},
				}),
				Dependencies: []string{
					"camel:timer",
					"camel:log",
				},
			},
		},
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
	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)
	assert.Equal(t, []string{"timer"}, trait.getKameletKeys(false))
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
		&v1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "timer",
			},
			Spec: v1.KameletSpec{
				KameletSpecBase: v1.KameletSpecBase{
					Template: templateOrFail(map[string]interface{}{
						"from": map[string]interface{}{
							"uri": "timer:tick",
						},
					}),
				},
			},
		})

	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

	err = trait.Apply(environment)
	require.Error(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	cond := environment.Integration.Status.GetCondition(v1.IntegrationConditionKameletsAvailable)
	assert.Equal(t, corev1.ConditionFalse, cond.Status)
	assert.Equal(t, v1.IntegrationConditionKameletsAvailableReason, cond.Reason)
	assert.Contains(t, cond.Message, "kamelets [none] not found")
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
		&v1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "timer",
			},
			Spec: v1.KameletSpec{
				KameletSpecBase: v1.KameletSpecBase{
					Template: templateOrFail(map[string]interface{}{
						"from": map[string]interface{}{
							"uri": "timer:tick",
						},
					}),
				},
			},
		},
		&v1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "none",
			},
			Spec: v1.KameletSpec{
				KameletSpecBase: v1.KameletSpecBase{
					Template: templateOrFail(map[string]interface{}{
						"from": map[string]interface{}{
							"uri": "timer:tick",
						},
					}),
				},
			},
		})

	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

	err = trait.Apply(environment)
	require.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	cond := environment.Integration.Status.GetCondition(v1.IntegrationConditionKameletsAvailable)
	assert.Equal(t, corev1.ConditionTrue, cond.Status)
	assert.Equal(t, v1.IntegrationConditionKameletsAvailableReason, cond.Reason)
	assert.Contains(t, cond.Message, "[none,timer] found")

	kameletsBundle := environment.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels[kubernetes.ConfigMapTypeLabel] == KameletBundleType
	})
	assert.NotNil(t, kameletsBundle)
	assert.Contains(t, kameletsBundle.Data, "timer.kamelet.yaml", "uri: timer:tick")
}

func createKameletsTestEnvironment(flow string, objects ...runtime.Object) (*kameletsTrait, *Environment) {
	catalog, _ := camel.DefaultCatalog()

	client, _ := test.NewFakeClient(objects...)
	trait, _ := newKameletsTrait().(*kameletsTrait)
	trait.Client = client

	environment := &Environment{
		Catalog:      NewCatalog(client),
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

func templateOrFail(template map[string]interface{}) *v1.Template {
	data, err := json.Marshal(template)
	if err != nil {
		panic(err)
	}
	t := v1.Template{RawMessage: data}
	return &t
}

func TestKameletSyntheticKitConditionTrue(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(
		"",
		&v1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "timer-source",
			},
			Spec: v1.KameletSpec{
				KameletSpecBase: v1.KameletSpecBase{
					Template: templateOrFail(map[string]interface{}{
						"from": map[string]interface{}{
							"uri": "timer:tick",
						},
					}),
				},
			},
		})
	environment.CamelCatalog = nil
	environment.Integration.Spec.Sources = nil
	trait.Auto = ptr.To(false)
	trait.List = "timer-source"

	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

	err = trait.Apply(environment)
	require.NoError(t, err)
	assert.Len(t, environment.Integration.Status.Conditions, 1)

	cond := environment.Integration.Status.GetCondition(v1.IntegrationConditionKameletsAvailable)
	assert.NotNil(t, cond)
	assert.Equal(t, corev1.ConditionTrue, cond.Status)
	assert.Equal(t, v1.IntegrationConditionKameletsAvailableReason, cond.Reason)
	assert.Contains(t, cond.Message, "[timer-source] found")

	kameletsBundle := environment.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels[kubernetes.ConfigMapTypeLabel] == KameletBundleType
	})
	assert.NotNil(t, kameletsBundle)
	assert.Contains(t, kameletsBundle.Data, "timer-source.kamelet.yaml", "uri: timer:tick")
}

func TestKameletSyntheticKitAutoConditionFalse(t *testing.T) {
	trait, environment := createKameletsTestEnvironment(
		"",
		&v1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "timer-source",
			},
			Spec: v1.KameletSpec{
				KameletSpecBase: v1.KameletSpecBase{
					Template: templateOrFail(map[string]interface{}{
						"from": map[string]interface{}{
							"uri": "timer:tick",
						},
					}),
				},
			},
		})
	environment.Integration.Spec.Sources = nil
	trait.List = "timer-source"

	// Auto=true by default. The source parsing will be empty as
	// there are no available sources.

	enabled, condition, err := trait.Configure(environment)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

	kameletsBundle := environment.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels[kubernetes.ConfigMapTypeLabel] == KameletBundleType
	})
	assert.Nil(t, kameletsBundle)
}
