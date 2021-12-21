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

package keda

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/apache/camel-k/addons/keda/duck/v1alpha1"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	camelv1alpha1 "github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/controller/kameletbinding"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	testingTrue  = true
	testingFalse = false
)

func TestManualConfig(t *testing.T) {
	keda, _ := NewKedaTrait().(*kedaTrait)
	keda.Enabled = &testingTrue
	keda.Auto = &testingFalse
	meta := map[string]string{
		"prop":      "val",
		"camelCase": "VAL",
	}
	keda.Triggers = append(keda.Triggers, kedaTrigger{
		Type:     "mytype",
		Metadata: meta,
	})
	env := createBasicTestEnvironment()

	res, err := keda.Configure(env)
	assert.NoError(t, err)
	assert.True(t, res)
	assert.NoError(t, keda.Apply(env))
	so := getScaledObject(env)
	assert.NotNil(t, so)
	assert.Len(t, so.Spec.Triggers, 1)
	assert.Equal(t, "mytype", so.Spec.Triggers[0].Type)
	assert.Equal(t, meta, so.Spec.Triggers[0].Metadata)
	assert.Nil(t, so.Spec.Triggers[0].AuthenticationRef)
	assert.Nil(t, getTriggerAuthentication(env))
	assert.Nil(t, getSecret(env))
}

func TestConfigFromSecret(t *testing.T) {
	keda, _ := NewKedaTrait().(*kedaTrait)
	keda.Enabled = &testingTrue
	keda.Auto = &testingFalse
	meta := map[string]string{
		"prop":      "val",
		"camelCase": "VAL",
	}
	keda.Triggers = append(keda.Triggers, kedaTrigger{
		Type:                 "mytype",
		Metadata:             meta,
		AuthenticationSecret: "my-secret",
	})
	env := createBasicTestEnvironment(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret",
		},
		Data: map[string][]byte{
			"bbb": []byte("val1"),
			"aaa": []byte("val2"),
		},
	})

	res, err := keda.Configure(env)
	assert.NoError(t, err)
	assert.True(t, res)
	assert.NoError(t, keda.Apply(env))
	so := getScaledObject(env)
	assert.NotNil(t, so)
	assert.Len(t, so.Spec.Triggers, 1)
	assert.Equal(t, "mytype", so.Spec.Triggers[0].Type)
	assert.Equal(t, meta, so.Spec.Triggers[0].Metadata)
	triggerAuth := getTriggerAuthentication(env)
	assert.NotNil(t, triggerAuth)
	assert.Equal(t, so.Spec.Triggers[0].AuthenticationRef.Name, triggerAuth.Name)
	assert.NotEqual(t, "my-secret", triggerAuth.Name)
	assert.Len(t, triggerAuth.Spec.SecretTargetRef, 2)
	assert.Equal(t, "aaa", triggerAuth.Spec.SecretTargetRef[0].Key)
	assert.Equal(t, "aaa", triggerAuth.Spec.SecretTargetRef[0].Parameter)
	assert.Equal(t, "my-secret", triggerAuth.Spec.SecretTargetRef[0].Name)
	assert.Equal(t, "bbb", triggerAuth.Spec.SecretTargetRef[1].Key)
	assert.Equal(t, "bbb", triggerAuth.Spec.SecretTargetRef[1].Parameter)
	assert.Equal(t, "my-secret", triggerAuth.Spec.SecretTargetRef[1].Name)
	assert.Nil(t, getSecret(env)) // Secret is already present, not generated
}

func TestKameletAutoDetection(t *testing.T) {
	keda, _ := NewKedaTrait().(*kedaTrait)
	keda.Enabled = &testingTrue
	env := createBasicTestEnvironment(
		&camelv1alpha1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "my-kamelet",
				Annotations: map[string]string{
					"camel.apache.org/keda.type": "my-scaler",
				},
			},
			Spec: camelv1alpha1.KameletSpec{
				Definition: &camelv1alpha1.JSONSchemaProps{
					Properties: map[string]camelv1alpha1.JSONSchemaProp{
						"a": {
							XDescriptors: []string{
								"urn:keda:metadata:a",
							},
						},
						"b": {
							XDescriptors: []string{
								"urn:keda:metadata:bb",
							},
						},
						"c": {
							XDescriptors: []string{
								"urn:keda:authentication:cc",
							},
						},
					},
				},
			},
		},
		&camelv1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "my-it",
			},
			Spec: camelv1.IntegrationSpec{
				Sources: []camelv1.SourceSpec{
					{
						DataSpec: camelv1.DataSpec{
							Name: "my-it.yaml",
							Content: "" +
								"- route:\n" +
								"    from:\n" +
								"      uri: kamelet:my-kamelet\n" +
								"      parameters:\n" +
								"        a: v1\n" +
								"        b: v2\n" +
								"        c: v3\n" +
								"    steps:\n" +
								"    - to: log:sink\n",
						},
						Language: camelv1.LanguageYaml,
					},
				},
			},
			Status: camelv1.IntegrationStatus{
				Phase: camelv1.IntegrationPhaseDeploying,
			},
		})

	res, err := keda.Configure(env)
	assert.NoError(t, err)
	assert.True(t, res)
	assert.NoError(t, keda.Apply(env))
	so := getScaledObject(env)
	assert.NotNil(t, so)
	assert.Len(t, so.Spec.Triggers, 1)
	assert.Equal(t, "my-scaler", so.Spec.Triggers[0].Type)
	assert.Equal(t, map[string]string{
		"a":  "v1",
		"bb": "v2",
	}, so.Spec.Triggers[0].Metadata)
	triggerAuth := getTriggerAuthentication(env)
	assert.NotNil(t, triggerAuth)
	assert.Equal(t, so.Spec.Triggers[0].AuthenticationRef.Name, triggerAuth.Name)
	assert.Len(t, triggerAuth.Spec.SecretTargetRef, 1)
	assert.Equal(t, "cc", triggerAuth.Spec.SecretTargetRef[0].Key)
	assert.Equal(t, "cc", triggerAuth.Spec.SecretTargetRef[0].Parameter)
	secretName := triggerAuth.Spec.SecretTargetRef[0].Name
	secret := getSecret(env)
	assert.NotNil(t, secret)
	assert.Equal(t, secretName, secret.Name)
	assert.Len(t, secret.StringData, 1)
	assert.Contains(t, secret.StringData, "cc")
}

func TestKameletBindingAutoDetection(t *testing.T) {
	keda, _ := NewKedaTrait().(*kedaTrait)
	keda.Enabled = &testingTrue
	logEndpoint := "log:info"
	klb := camelv1alpha1.KameletBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-binding",
		},
		Spec: camelv1alpha1.KameletBindingSpec{
			Source: camelv1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: camelv1alpha1.SchemeGroupVersion.String(),
					Name:       "my-kamelet",
				},
				Properties: asEndpointProperties(map[string]string{
					"a": "v1",
					"b": "v2",
					"c": "v3",
				}),
			},
			Sink: camelv1alpha1.Endpoint{
				URI: &logEndpoint,
			},
		},
	}

	env := createBasicTestEnvironment(
		&camelv1alpha1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "my-kamelet",
				Annotations: map[string]string{
					"camel.apache.org/keda.type": "my-scaler",
				},
			},
			Spec: camelv1alpha1.KameletSpec{
				Definition: &camelv1alpha1.JSONSchemaProps{
					Properties: map[string]camelv1alpha1.JSONSchemaProp{
						"a": {
							XDescriptors: []string{
								"urn:keda:metadata:a",
							},
						},
						"b": {
							XDescriptors: []string{
								"urn:keda:metadata:bb",
							},
						},
						"c": {
							XDescriptors: []string{
								"urn:keda:authentication:cc",
							},
						},
					},
				},
			},
		},
		&klb,
		&camelv1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "camel-k",
			},
			Spec: camelv1.IntegrationPlatformSpec{
				Cluster: camelv1.IntegrationPlatformClusterKubernetes,
				Profile: camelv1.TraitProfileKubernetes,
			},
			Status: camelv1.IntegrationPlatformStatus{
				Phase: camelv1.IntegrationPlatformPhaseReady,
			},
		})

	it, err := kameletbinding.CreateIntegrationFor(env.Ctx, env.Client, &klb)
	assert.NoError(t, err)
	assert.NotNil(t, it)
	env.Integration = it

	it.Status.Phase = camelv1.IntegrationPhaseInitialization
	init := trait.NewInitTrait()
	ok, err := init.Configure(env)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.NoError(t, init.Apply(env))

	it.Status.Phase = camelv1.IntegrationPhaseDeploying
	res, err := keda.Configure(env)
	assert.NoError(t, err)
	assert.True(t, res)
	assert.NoError(t, keda.Apply(env))
	so := getScaledObject(env)
	assert.NotNil(t, so)
	assert.Len(t, so.Spec.Triggers, 1)
	assert.Equal(t, "my-scaler", so.Spec.Triggers[0].Type)
	assert.Equal(t, map[string]string{
		"a":  "v1",
		"bb": "v2",
	}, so.Spec.Triggers[0].Metadata)
	triggerAuth := getTriggerAuthentication(env)
	assert.NotNil(t, triggerAuth)
	assert.Equal(t, so.Spec.Triggers[0].AuthenticationRef.Name, triggerAuth.Name)
	assert.Len(t, triggerAuth.Spec.SecretTargetRef, 1)
	assert.Equal(t, "cc", triggerAuth.Spec.SecretTargetRef[0].Key)
	assert.Equal(t, "cc", triggerAuth.Spec.SecretTargetRef[0].Parameter)
	secretName := triggerAuth.Spec.SecretTargetRef[0].Name
	secret := getSecret(env)
	assert.NotNil(t, secret)
	assert.Equal(t, secretName, secret.Name)
	assert.Len(t, secret.StringData, 1)
	assert.Contains(t, secret.StringData, "cc")
}

func TestHackReplicas(t *testing.T) {
	keda, _ := NewKedaTrait().(*kedaTrait)
	keda.Enabled = &testingTrue
	keda.Auto = &testingFalse
	keda.Triggers = append(keda.Triggers, kedaTrigger{
		Type: "custom",
		Metadata: map[string]string{
			"a": "b",
		},
	})
	keda.HackControllerReplicas = &testingTrue
	env := createBasicTestEnvironment(
		&camelv1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "my-it",
			},
			Status: camelv1.IntegrationStatus{
				Phase: camelv1.IntegrationPhaseInitialization,
			},
		},
	)

	res, err := keda.Configure(env)
	assert.NoError(t, err)
	assert.True(t, res)
	assert.NoError(t, keda.Apply(env))
	it := camelv1.Integration{}
	key := client.ObjectKey{
		Namespace: "test",
		Name:      "my-it",
	}
	assert.NoError(t, env.Client.Get(env.Ctx, key, &it))
	assert.NotNil(t, it.Spec.Replicas)
	assert.Equal(t, int32(1), *it.Spec.Replicas)
}

func getScaledObject(e *trait.Environment) *v1alpha1.ScaledObject {
	var res *v1alpha1.ScaledObject
	for _, o := range e.Resources.Items() {
		if so, ok := o.(*v1alpha1.ScaledObject); ok {
			if res != nil {
				panic("multiple ScaledObjects found in env")
			}
			res = so
		}
	}
	return res
}

func getTriggerAuthentication(e *trait.Environment) *v1alpha1.TriggerAuthentication {
	var res *v1alpha1.TriggerAuthentication
	for _, o := range e.Resources.Items() {
		if so, ok := o.(*v1alpha1.TriggerAuthentication); ok {
			if res != nil {
				panic("multiple TriggerAuthentication found in env")
			}
			res = so
		}
	}
	return res
}

func getSecret(e *trait.Environment) *corev1.Secret {
	var res *corev1.Secret
	for _, o := range e.Resources.Items() {
		if so, ok := o.(*corev1.Secret); ok {
			if res != nil {
				panic("multiple Secret found in env")
			}
			res = so
		}
	}
	return res
}

func createBasicTestEnvironment(resources ...runtime.Object) *trait.Environment {
	fakeClient, err := test.NewFakeClient(resources...)
	if err != nil {
		panic(errors.Wrap(err, "could not create fake client"))
	}

	var it *camelv1.Integration
	for _, res := range resources {
		if integration, ok := res.(*camelv1.Integration); ok {
			it = integration
		}
	}
	if it == nil {
		it = &camelv1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "integration-name",
			},
			Status: camelv1.IntegrationStatus{
				Phase: camelv1.IntegrationPhaseDeploying,
			},
		}
	}

	var pl *camelv1.IntegrationPlatform
	for _, res := range resources {
		if platform, ok := res.(*camelv1.IntegrationPlatform); ok {
			pl = platform
		}
	}
	if pl == nil {
		pl = &camelv1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "camel-k",
			},
			Spec: camelv1.IntegrationPlatformSpec{
				Cluster: camelv1.IntegrationPlatformClusterKubernetes,
				Profile: camelv1.TraitProfileKubernetes,
			},
		}
	}

	return &trait.Environment{
		Catalog:     trait.NewCatalog(nil),
		Ctx:         context.Background(),
		Client:      fakeClient,
		Integration: it,
		CamelCatalog: &camel.RuntimeCatalog{
			CamelCatalogSpec: camelv1.CamelCatalogSpec{
				Runtime: camelv1.RuntimeSpec{
					Version:  "0.0.1",
					Provider: camelv1.RuntimeProviderQuarkus,
				},
			},
		},
		Platform:              pl,
		Resources:             kubernetes.NewCollection(),
		ApplicationProperties: make(map[string]string),
	}
}

func asEndpointProperties(props map[string]string) *camelv1alpha1.EndpointProperties {
	serialized, err := json.Marshal(props)
	if err != nil {
		panic(err)
	}
	return &camelv1alpha1.EndpointProperties{
		RawMessage: serialized,
	}
}
