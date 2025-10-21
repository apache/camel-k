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
