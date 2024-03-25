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

package strimzi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/apache/camel-k/v2/addons/strimzi/duck/client/internalclientset/fake"
	"github.com/apache/camel-k/v2/addons/strimzi/duck/v1beta2"
	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/apache/camel-k/v2/pkg/util/bindings"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStrimziDirect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := test.NewFakeClient()
	require.NoError(t, err)

	bindingContext := bindings.BindingContext{
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

	binding, err := BindingProvider{}.Translate(bindingContext, bindings.EndpointContext{
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

	cluster := v1beta2.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "myclusterx",
		},
		Status: v1beta2.KafkaStatus{
			Listeners: []v1beta2.KafkaStatusListener{
				{
					Type: "tls",
				},
				{
					BootstrapServers: "my-clusterx-kafka-bootstrap:9092",
					Type:             "plain",
				},
			},
		},
	}

	topic := v1beta2.KafkaTopic{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mytopicy",
			Labels: map[string]string{
				v1beta2.StrimziKafkaClusterLabel: "myclusterx",
			},
		},
	}

	client := fake.NewSimpleClientset(&cluster, &topic)
	provider := BindingProvider{
		Client: client,
	}

	bindingContext := bindings.BindingContext{
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

	binding, err := provider.Translate(bindingContext, bindings.EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "kafka:mytopicy?brokers=my-clusterx-kafka-bootstrap%3A9092", binding.URI)
	assert.Equal(t, camelv1.Traits{}, binding.Traits)
}

func asEndpointProperties(props map[string]string) *camelv1.EndpointProperties {
	serialized, err := json.Marshal(props)
	if err != nil {
		panic(err)
	}
	return &camelv1.EndpointProperties{
		RawMessage: serialized,
	}
}

func TestStrimziLookupByTopicName(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := v1beta2.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "myclusterx",
		},
		Status: v1beta2.KafkaStatus{
			Listeners: []v1beta2.KafkaStatusListener{
				{
					Type: "tls",
				},
				{
					BootstrapServers: "my-clusterx-kafka-bootstrap:9092",
					Type:             "plain",
				},
			},
		},
	}

	topic := v1beta2.KafkaTopic{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mytopicy",
			Labels: map[string]string{
				v1beta2.StrimziKafkaClusterLabel: "myclusterx",
			},
		},
		Status: v1beta2.KafkaTopicStatus{
			TopicName: "my-topic-name",
		},
	}

	client := fake.NewSimpleClientset(&cluster, &topic)
	provider := BindingProvider{
		Client: client,
	}

	bindingContext := bindings.BindingContext{
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

	binding, err := provider.Translate(bindingContext, bindings.EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "kafka:my-topic-name?brokers=my-clusterx-kafka-bootstrap%3A9092", binding.URI)
	assert.Equal(t, camelv1.Traits{}, binding.Traits)
}
