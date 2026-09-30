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
	"net/http"
	"net/http/httptest"
	"testing"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	strimziv1 "github.com/apache/camel-k/v2/pkg/apis/duck/strimzi/v1"
	"github.com/apache/camel-k/v2/pkg/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
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

	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	require.NoError(t, client.Create(ctx, &cluster))
	require.NoError(t, client.Create(ctx, &topic))
	provider := StrimziBindingProvider{}

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
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

	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	require.NoError(t, client.Create(ctx, &cluster))
	require.NoError(t, client.Create(ctx, &topic))
	provider := StrimziBindingProvider{}

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
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

	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	require.NoError(t, client.Create(ctx, &cluster))
	provider := StrimziBindingProvider{}

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
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

	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	require.NoError(t, client.Create(ctx, &cluster))
	provider := StrimziBindingProvider{}

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
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

func TestStrimziLookupTopicNamespace(t *testing.T) {
	for _, tc := range []struct {
		name      string
		namespace string
		topicName string
		wantName  string
	}{
		{name: "context namespace", topicName: "shared", wantName: "local"},
		{name: "explicit namespace", namespace: "other", topicName: "shared", wantName: "remote"},
		{name: "resource name", topicName: "local", wantName: "local"},
		{name: "missing topic", topicName: "missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := internal.NewFakeClient()
			require.NoError(t, err)
			for _, topic := range []strimziv1.KafkaTopic{
				{ObjectMeta: metav1.ObjectMeta{Name: "aaa-unrelated", Namespace: "test"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "local", Namespace: "test"}, Status: strimziv1.KafkaTopicStatus{TopicName: "shared"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "remote", Namespace: "other"}, Status: strimziv1.KafkaTopicStatus{TopicName: "shared"}},
			} {
				require.NoError(t, client.Create(ctx, &topic))
			}
			topic, err := (StrimziBindingProvider{}).lookupTopic(BindingContext{
				Ctx: ctx, Client: client, Namespace: "test",
			}, camelv1.Endpoint{Ref: &v1.ObjectReference{Name: tc.topicName, Namespace: tc.namespace}})
			if tc.wantName == "" {
				require.Error(t, err)
				assert.Nil(t, topic)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, topic)
			assert.Equal(t, tc.wantName, topic.Name)
		})
	}
}

func TestStrimziMissingCluster(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)
	servers, err := (StrimziBindingProvider{}).getBootstrapServers(BindingContext{
		Ctx: context.Background(), Client: client,
	}, "missing", "test")
	require.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
	assert.Empty(t, servers)
}

func TestStrimziBypassesCache(t *testing.T) {
	for _, tc := range []struct {
		name    string
		kind    string
		refName string
	}{
		{name: "cluster", kind: "Kafka", refName: "cluster"},
		{name: "topic", kind: "KafkaTopic", refName: "topic"},
		{name: "topic status name", kind: "KafkaTopic", refName: "shared"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				var response string
				switch r.URL.Path {
				case "/apis/kafka.strimzi.io/v1/namespaces/other/kafkas/cluster":
					response = `{"apiVersion":"kafka.strimzi.io/v1","kind":"Kafka","metadata":{"name":"cluster","namespace":"other"},"status":{"listeners":[{"name":"plain","bootstrapServers":"live:9092"}]}}`
				case "/apis/kafka.strimzi.io/v1/namespaces/other/kafkatopics/topic":
					response = `{"apiVersion":"kafka.strimzi.io/v1","kind":"KafkaTopic","metadata":{"name":"topic","namespace":"other","labels":{"strimzi.io/cluster":"cluster"}}}`
				case "/apis/kafka.strimzi.io/v1/namespaces/other/kafkatopics/shared":
					w.WriteHeader(http.StatusNotFound)
					response = `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"NotFound","code":404}`
				case "/apis/kafka.strimzi.io/v1/namespaces/other/kafkatopics":
					response = `{"apiVersion":"kafka.strimzi.io/v1","kind":"KafkaTopicList","items":[{"metadata":{"name":"topic","namespace":"other","labels":{"strimzi.io/cluster":"cluster"}},"status":{"topicName":"shared"}}]}`
				default:
					http.NotFound(w, r)
					return
				}
				_, err := w.Write([]byte(response))
				assert.NoError(t, err)
			}))
			defer server.Close()
			mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{strimziv1.SchemeGroupVersion})
			mapper.Add(strimziv1.SchemeGroupVersion.WithKind("Kafka"), meta.RESTScopeNamespace)
			mapper.Add(strimziv1.SchemeGroupVersion.WithKind("KafkaTopic"), meta.RESTScopeNamespace)
			client, err := internal.NewFakeClient()
			require.NoError(t, err)
			liveClient, err := ctrl.New(&rest.Config{Host: server.URL}, ctrl.Options{
				Scheme: client.GetScheme(), Mapper: mapper,
				Cache: &ctrl.CacheOptions{Reader: strimziRejectingCache{}},
			})
			require.NoError(t, err)
			client.(*internal.FakeClient).Client = liveClient
			endpoint := camelv1.Endpoint{Ref: &v1.ObjectReference{
				APIVersion: "kafka.strimzi.io/v1", Kind: tc.kind, Name: tc.refName, Namespace: "other",
			}}
			if tc.kind == "Kafka" {
				endpoint.Properties = asEndpointProperties(map[string]string{"topic": tc.refName})
			}
			binding, err := (StrimziBindingProvider{}).Translate(BindingContext{
				Ctx: context.Background(), Client: client, Namespace: "test",
			}, EndpointContext{}, endpoint)
			require.NoError(t, err)
			require.NotNil(t, binding)
			assert.Equal(t, "kafka:"+tc.refName+"?brokers=live%3A9092", binding.URI)
		})
	}
}

type strimziRejectingCache struct{}

func (strimziRejectingCache) Get(context.Context, ctrl.ObjectKey, ctrl.Object, ...ctrl.GetOption) error {
	return errors.New("Strimzi reads must not use the cache")
}

func (strimziRejectingCache) List(context.Context, ctrl.ObjectList, ...ctrl.ListOption) error {
	return errors.New("Strimzi reads must not use the cache")
}
