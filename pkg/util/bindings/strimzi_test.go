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

package bindings

import (
	"context"
	"errors"
	"testing"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	strimziv1 "github.com/apache/camel-k/v2/pkg/apis/duck/strimzi/v1"
	"github.com/apache/camel-k/v2/pkg/client/strimzi/clientset/internalclientset/fake"
	"github.com/apache/camel-k/v2/pkg/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestStrimziDirect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &v1.ObjectReference{
			Kind:       "KafkaTopic",
			Name:       "mytopic",
			APIVersion: "kafka.strimzi.io/v1beta2",
		},
		Properties: asEndpointProperties(map[string]string{
			"brokers": "my-cluster-kafka-bootstrap:9092",
		}),
	}

	binding, err := StrimziBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "kafka:mytopic?brokers=my-cluster-kafka-bootstrap%3A9092", binding.URI)
	assert.Equal(t, camelv1.Traits{}, binding.Traits)
}

func TestStrimziLookup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := strimziv1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "myclusterx",
		},
		Status: strimziv1.KafkaStatus{
			Listeners: []strimziv1.KafkaStatusListener{
				{
					Name: "tls",
				},
				{
					BootstrapServers: "my-clusterx-kafka-bootstrap:9092",
					Name:             "plain",
				},
			},
		},
	}

	topic := strimziv1.KafkaTopic{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mytopicy",
			Labels: map[string]string{
				strimziv1.StrimziKafkaClusterLabel: "myclusterx",
			},
		},
	}

	client := fake.NewSimpleClientset(&cluster, &topic)
	provider := StrimziBindingProvider{
		Client: client,
	}

	bindingContext := BindingContext{
		Ctx:       ctx,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &v1.ObjectReference{
			Kind:       "KafkaTopic",
			Name:       "mytopicy",
			APIVersion: "kafka.strimzi.io/v1beta2",
		},
	}

	binding, err := provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "kafka:mytopicy?brokers=my-clusterx-kafka-bootstrap%3A9092", binding.URI)
	assert.Equal(t, camelv1.Traits{}, binding.Traits)
}

func TestStrimziLookupByTopicName(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := strimziv1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "myclusterx",
		},
		Status: strimziv1.KafkaStatus{
			Listeners: []strimziv1.KafkaStatusListener{
				{
					Name: "tls",
				},
				{
					BootstrapServers: "my-clusterx-kafka-bootstrap:9092",
					Name:             "plain",
				},
			},
		},
	}

	topic := strimziv1.KafkaTopic{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mytopicy",
			Labels: map[string]string{
				strimziv1.StrimziKafkaClusterLabel: "myclusterx",
			},
		},
		Status: strimziv1.KafkaTopicStatus{
			TopicName: "my-topic-name",
		},
	}

	client := fake.NewSimpleClientset(&cluster, &topic)
	provider := StrimziBindingProvider{
		Client: client,
	}

	bindingContext := BindingContext{
		Ctx:       ctx,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &v1.ObjectReference{
			Kind:       "KafkaTopic",
			Name:       "my-topic-name",
			APIVersion: "kafka.strimzi.io/v1beta2",
		},
	}

	binding, err := provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "kafka:my-topic-name?brokers=my-clusterx-kafka-bootstrap%3A9092", binding.URI)
	assert.Equal(t, camelv1.Traits{}, binding.Traits)
}

func TestStrimziKafkaCR(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := strimziv1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-kafka",
		},
		Status: strimziv1.KafkaStatus{
			Listeners: []strimziv1.KafkaStatusListener{
				{
					Name: "tls",
				},
				{
					BootstrapServers: "my-clusterx-kafka-bootstrap:9092",
					Name:             "plain",
				},
			},
		},
	}

	client := fake.NewSimpleClientset(&cluster)
	provider := StrimziBindingProvider{
		Client: client,
	}

	bindingContext := BindingContext{
		Ctx:       ctx,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &v1.ObjectReference{
			Kind:       "Kafka",
			Name:       "my-kafka",
			APIVersion: "kafka.strimzi.io/v1beta2",
		},
		Properties: asEndpointProperties(map[string]string{
			"topic": "my-topic",
		}),
	}

	binding, err := provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "kafka:my-topic?brokers=my-clusterx-kafka-bootstrap%3A9092", binding.URI)
	assert.Equal(t, camelv1.Traits{}, binding.Traits)
}

func TestStrimziPassThrough(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := strimziv1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-kafka",
		},
		Status: strimziv1.KafkaStatus{
			Listeners: []strimziv1.KafkaStatusListener{
				{
					Name: "tls",
				},
				{
					BootstrapServers: "my-clusterx-kafka-bootstrap:9092",
					Name:             "plain",
				},
			},
		},
	}

	client := fake.NewSimpleClientset(&cluster)
	provider := StrimziBindingProvider{
		Client: client,
	}

	bindingContext := BindingContext{
		Ctx:       ctx,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &v1.ObjectReference{
			Kind:       "AnotherKind",
			Name:       "my-kafka",
			APIVersion: "anotherApiVersion",
		},
	}

	binding, err := provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.Nil(t, binding)
}

func TestStrimziNamespace(t *testing.T) {
	for _, tc := range []struct {
		name      string
		namespace string
		wantURI   string
	}{
		{name: "context namespace", wantURI: "kafka:topic?brokers=local%3A9092"},
		{name: "explicit namespace", namespace: "other", wantURI: "kafka:topic?brokers=remote%3A9092"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(
				&strimziv1.KafkaTopic{Name: "topic", Namespace: "test", Labels: map[string]string{strimziv1.StrimziKafkaClusterLabel: "local-cluster"}},
				&strimziv1.KafkaTopic{Name: "topic", Namespace: "other", Labels: map[string]string{strimziv1.StrimziKafkaClusterLabel: "remote-cluster"}},
				&strimziv1.Kafka{Name: "local-cluster", Namespace: "test", Status: strimziv1.KafkaStatus{Listeners: []strimziv1.KafkaStatusListener{{Name: "plain", BootstrapServers: "local:9092"}}}},
				&strimziv1.Kafka{Name: "remote-cluster", Namespace: "other", Status: strimziv1.KafkaStatus{Listeners: []strimziv1.KafkaStatusListener{{Name: "plain", BootstrapServers: "remote:9092"}}}},
			)
			binding, err := (StrimziBindingProvider{Client: client}).Translate(BindingContext{
				Ctx: context.Background(), Namespace: "test",
			}, EndpointContext{}, camelv1.Endpoint{Ref: &v1.ObjectReference{
				APIVersion: "kafka.strimzi.io/v1", Kind: "KafkaTopic", Name: "topic", Namespace: tc.namespace,
			}})
			require.NoError(t, err)
			require.NotNil(t, binding)
			assert.Equal(t, tc.wantURI, binding.URI)
		})
	}
}

func TestStrimziMissingCluster(t *testing.T) {
	provider := StrimziBindingProvider{Client: fake.NewSimpleClientset()}
	servers, err := provider.getBootstrapServers(BindingContext{Ctx: context.Background()}, "missing", "test")
	require.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
	assert.Empty(t, servers)
}

func TestStrimziListenerValidation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		listeners []strimziv1.KafkaStatusListener
		wantError string
	}{
		{name: "no listeners", wantError: `cluster "cluster" has no listeners of name "plain"`},
		{name: "only TLS", listeners: []strimziv1.KafkaStatusListener{{Name: "tls", BootstrapServers: "tls:9093"}}, wantError: `cluster "cluster" has no listeners of name "plain"`},
		{name: "empty bootstrap servers", listeners: []strimziv1.KafkaStatusListener{{Name: "plain"}}, wantError: `cluster "cluster" has no bootstrap servers in "plain" listener`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := StrimziBindingProvider{Client: fake.NewSimpleClientset(&strimziv1.Kafka{
				Name: "cluster", Namespace: "test",
				Status: strimziv1.KafkaStatus{Listeners: tc.listeners},
			})}
			servers, err := provider.getBootstrapServers(BindingContext{Ctx: context.Background()}, "cluster", "test")
			require.EqualError(t, err, tc.wantError)
			assert.Empty(t, servers)
		})
	}
}

func TestStrimziLookupErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		verb string
	}{
		{name: "get failure", verb: "get"},
		{name: "list failure", verb: "list"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			wantErr := errors.New("API request failed")
			client.PrependReactor(tc.verb, "kafkatopics", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, wantErr
			})
			topic, err := (StrimziBindingProvider{Client: client}).lookupTopic(BindingContext{
				Ctx: context.Background(), Namespace: "test",
			}, camelv1.Endpoint{Ref: &v1.ObjectReference{Name: "missing"}})
			require.ErrorIs(t, err, wantErr)
			assert.Nil(t, topic)
		})
	}
}

func TestStrimziMissingTopic(t *testing.T) {
	provider := StrimziBindingProvider{Client: fake.NewSimpleClientset()}
	topic, err := provider.lookupTopic(BindingContext{Ctx: context.Background(), Namespace: "test"},
		camelv1.Endpoint{Ref: &v1.ObjectReference{Name: "missing"}})
	require.EqualError(t, err, "couldn't find any KafkaTopic with either name or topicName missing")
	assert.Nil(t, topic)
}
