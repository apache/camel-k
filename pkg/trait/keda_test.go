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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/apis/duck/keda/v1alpha1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/gzip"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestKeda(t *testing.T) {
	environment := nominalEnv(t)
	environment.Platform.ResyncStatusFullConfig()
	traitCatalog := environment.Catalog

	_, _, err := traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("keda"))

	scaledObject := getKedaScaledObject(environment.Resources)
	assert.NotNil(t, scaledObject)
	assert.Len(t, scaledObject.Spec.Triggers, 1)
	assert.Equal(t, "kafka", scaledObject.Spec.Triggers[0].Type)
	assert.Equal(t, "my-cluster-kafka-bootstrap.strimzi.svc:9092", scaledObject.Spec.Triggers[0].Metadata["bootstrapServers"])
	assert.Equal(t, "group-1", scaledObject.Spec.Triggers[0].Metadata["consumerGroup"])
	assert.Equal(t, "my-topic", scaledObject.Spec.Triggers[0].Metadata["topic"])
	assert.Equal(t, "10", scaledObject.Spec.Triggers[0].Metadata["lagThreshold"])
}

func TestKedaAutoDiscovery(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		expectedType   string
		expectedParams map[string]string
		manualTrigger  *traitv1.KedaTrigger
		autoMetadata   map[string]map[string]string
		expectedCount  int
	}{
		{
			name:         "kafka",
			source:       `from("kafka:my-topic?brokers=my-broker:9092&groupId=my-group").log("${body}");`,
			expectedType: "kafka",
			expectedParams: map[string]string{
				"topic":            "my-topic",
				"bootstrapServers": "my-broker:9092",
				"consumerGroup":    "my-group",
			},
			manualTrigger: nil,
			expectedCount: 1,
		},
		{
			name:         "manual-trigger-does-not-block-auto-discovery",
			source:       `from("kafka:my-topic?brokers=my-broker:9092&groupId=my-group").log("${body}");`,
			expectedType: "kafka",
			expectedParams: map[string]string{
				"topic":            "my-topic",
				"bootstrapServers": "my-broker:9092",
				"consumerGroup":    "my-group",
			},
			manualTrigger: &traitv1.KedaTrigger{
				Type: "cron",
				Metadata: map[string]string{
					"timezone": "Etc/UTC",
					"start":    "0 * * * *",
					"end":      "59 * * * *",
				},
			},
			expectedCount: 2, // 1 manual (cron) + 1 auto-discovered (kafka)
		},
		{
			name:         "auto-metadata-merge",
			source:       `from("kafka:my-topic?brokers=my-broker:9092&groupId=my-group").log("${body}");`,
			expectedType: "kafka",
			expectedParams: map[string]string{
				"topic":            "my-topic",
				"bootstrapServers": "my-broker:9092",
				"consumerGroup":    "my-group",
				"lagThreshold":     "10", // merged from autoMetadata
			},
			autoMetadata: map[string]map[string]string{
				"kafka": {
					"lagThreshold": "10",
				},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environment := autoDiscoveryEnvWithSource(t, tt.source, tt.manualTrigger, tt.autoMetadata)
			environment.Platform.ResyncStatusFullConfig()
			traitCatalog := environment.Catalog

			_, _, err := traitCatalog.apply(&environment)

			require.NoError(t, err)
			assert.NotEmpty(t, environment.ExecutedTraits)
			assert.NotNil(t, environment.GetTrait("keda"))

			scaledObject := getKedaScaledObject(environment.Resources)
			require.NotNil(t, scaledObject)
			require.Len(t, scaledObject.Spec.Triggers, tt.expectedCount)
			// Find the auto-discovered trigger by type
			var foundTrigger *v1alpha1.ScaleTriggers
			for i := range scaledObject.Spec.Triggers {
				if scaledObject.Spec.Triggers[i].Type == tt.expectedType {
					foundTrigger = &scaledObject.Spec.Triggers[i]
					break
				}
			}
			require.NotNil(t, foundTrigger, "expected trigger type %s not found", tt.expectedType)
			for k, v := range tt.expectedParams {
				assert.Equal(t, v, foundTrigger.Metadata[k], "metadata key %s mismatch", k)
			}
		})
	}
}

func TestKedaAuthentication(t *testing.T) {
	environment := nominalEnv(t)
	environment.Integration.Spec.Traits.Keda.Triggers[0].Secrets = []*traitv1.KedaSecret{
		&traitv1.KedaSecret{
			Name: "my-secret",
			Mapping: map[string]string{
				"secret-name":     "kafka-name",
				"secret-password": "kafka-password",
			},
		},
	}
	environment.Platform.ResyncStatusFullConfig()
	traitCatalog := environment.Catalog

	_, _, err := traitCatalog.apply(&environment)

	require.NoError(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("keda"))

	scaledObject := getKedaScaledObject(environment.Resources)
	assert.NotNil(t, scaledObject)
	triggerAuths := getKedaTriggersAuth(environment.Resources)
	assert.NotNil(t, triggerAuths)
	assert.Len(t, scaledObject.Spec.Triggers, 1)
	assert.Len(t, triggerAuths, 1)
	// Check scaledObj trigger auth ref
	assert.Equal(t, "kafka", scaledObject.Spec.Triggers[0].Type)
	assert.NotNil(t, scaledObject.Spec.Triggers[0].AuthenticationRef)
	assert.Equal(t, "test-kafka", scaledObject.Spec.Triggers[0].AuthenticationRef.Name)
	// Check triggers auth
	assert.Equal(t, "test-kafka", triggerAuths[0].Name)
	assert.Equal(t, "ns", triggerAuths[0].Namespace)
	assert.NotNil(t, triggerAuths[0].Spec.SecretTargetRef)
	assert.Len(t, triggerAuths[0].Spec.SecretTargetRef, 2)
	assert.ElementsMatch(
		t,
		[]v1alpha1.AuthSecretTargetRef{
			{
				Name:      "my-secret",
				Key:       "secret-name",
				Parameter: "kafka-name",
			},
			{
				Name:      "my-secret",
				Key:       "secret-password",
				Parameter: "kafka-password",
			},
		},
		triggerAuths[0].Spec.SecretTargetRef)
}

func getKedaScaledObject(c *kubernetes.Collection) *v1alpha1.ScaledObject {
	var scaledObject *v1alpha1.ScaledObject
	c.Visit(func(res runtime.Object) {
		if conv, ok := res.(*v1alpha1.ScaledObject); ok {
			scaledObject = conv
		}
	})

	return scaledObject
}

func getKedaTriggersAuth(c *kubernetes.Collection) []*v1alpha1.TriggerAuthentication {
	var triggerAuths []*v1alpha1.TriggerAuthentication
	c.Visit(func(res runtime.Object) {
		if conv, ok := res.(*v1alpha1.TriggerAuthentication); ok {
			triggerAuths = append(triggerAuths, conv)
		}
	})

	return triggerAuths
}

func nominalEnv(t *testing.T) Environment {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	compressedRoute, err := gzip.CompressBase64([]byte(`from("kafka:test").log("hello");`))
	require.NoError(t, err)

	return Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ServiceTestName,
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:        "routes.java",
							Content:     string(compressedRoute),
							Compression: true,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Keda: &traitv1.KedaTrait{
						Trait: traitv1.Trait{
							Enabled: ptr.To(true),
						},
						Auto: ptr.To(false), // Disable auto-discovery for manual trigger test
						Triggers: []traitv1.KedaTrigger{
							traitv1.KedaTrigger{
								Type: "kafka",
								Metadata: map[string]string{
									"bootstrapServers": "my-cluster-kafka-bootstrap.strimzi.svc:9092",
									"consumerGroup":    "group-1",
									"topic":            "my-topic",
									"lagThreshold":     "10",
								},
							},
						},
					},
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
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
}

// autoDiscoveryEnvWithSource creates an environment with the given source and optional manual trigger.
func autoDiscoveryEnvWithSource(t *testing.T, source string, manualTrigger *traitv1.KedaTrigger, autoMetadata map[string]map[string]string) Environment {
	t.Helper()
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	client, _ := internal.NewFakeClient()
	traitCatalog := NewCatalog(nil)

	return Environment{
		CamelCatalog: catalog,
		Catalog:      traitCatalog,
		Client:       client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name:        "routes.java",
							Content:     source,
							Compression: false,
						},
						Language: v1.LanguageJavaSource,
					},
				},
				Traits: v1.Traits{
					Keda: func() *traitv1.KedaTrait {
						keda := &traitv1.KedaTrait{
							Trait: traitv1.Trait{
								Enabled: ptr.To(true),
							},
						}
						if manualTrigger != nil {
							keda.Triggers = []traitv1.KedaTrigger{*manualTrigger}
						}
						if autoMetadata != nil {
							keda.AutoMetadata = autoMetadata
						}
						return keda
					}(),
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
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyJib,
					Registry:        v1.RegistrySpec{Address: "registry"},
					RuntimeVersion:  catalog.Runtime.Version,
				},
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
	}
}
