package strimzi

import (
	"context"
	"encoding/json"
	"github.com/apache/camel-k/addons/strimzi/duck/v1beta1"
	"github.com/apache/camel-k/addons/strimzi/duck/v1beta1/client/internalclientset/fake"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/bindings"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestStrimziDirect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := test.NewFakeClient()
	assert.NoError(t, err)

	bindingContext := bindings.BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
	}

	endpoint := v1alpha1.Endpoint{
		Ref: &v1.ObjectReference{
			Kind:       "KafkaTopic",
			Name:       "mytopic",
			APIVersion: "kafka.strimzi.io/v1beta1",
		},
		Properties: asEndpointProperties(map[string]string{
			"brokers": "my-cluster-kafka-bootstrap:9092",
		}),
	}

	binding, err := StrimziBindingProvider{}.Translate(bindingContext, v1alpha1.EndpointTypeSink, endpoint)
	assert.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "kafka:mytopic?brokers=my-cluster-kafka-bootstrap%3A9092", binding.URI)
	assert.Nil(t, binding.Traits)
}

func TestStrimziLookup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := v1beta1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "myclusterx",
		},
		Status: v1beta1.KafkaStatus{
			Listeners: []v1beta1.KafkaStatusListener{
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

	topic := v1beta1.KafkaTopic{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mytopicy",
			Labels: map[string]string{
				v1beta1.StrimziKafkaClusterLabel: "myclusterx",
			},
		},
	}

	client := fake.NewSimpleClientset(&cluster, &topic)
	provider := StrimziBindingProvider{
		Client: client.KafkaV1beta1(),
	}

	bindingContext := bindings.BindingContext{
		Ctx:       ctx,
		Namespace: "test",
	}

	endpoint := v1alpha1.Endpoint{
		Ref: &v1.ObjectReference{
			Kind:       "KafkaTopic",
			Name:       "mytopicy",
			APIVersion: "kafka.strimzi.io/v1beta1",
		},
	}

	binding, err := provider.Translate(bindingContext, v1alpha1.EndpointTypeSink, endpoint)
	assert.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "kafka:mytopicy?brokers=my-clusterx-kafka-bootstrap%3A9092", binding.URI)
	assert.Nil(t, binding.Traits)
}

func asEndpointProperties(props map[string]string) v1alpha1.EndpointProperties {
	serialized, err := json.Marshal(props)
	if err != nil {
		panic(err)
	}
	return v1alpha1.EndpointProperties{
		RawMessage: serialized,
	}
}
