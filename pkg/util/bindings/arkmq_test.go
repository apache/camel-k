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
	"testing"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	arkmqv1beta1 "github.com/apache/camel-k/v2/pkg/apis/duck/arkmq/v1beta1"
	"github.com/apache/camel-k/v2/pkg/client/arkmq/clientset/internalclientset/fake"
	"github.com/apache/camel-k/v2/pkg/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestArkMQDirect(t *testing.T) {
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
		Ref: &corev1.ObjectReference{
			Kind:       "ActiveMQArtemisAddress",
			Name:       "myqueue",
			APIVersion: "broker.amq.io/v1beta1",
		},
		Properties: asEndpointProperties(map[string]string{
			"brokerURL": "tcp://custom-broker:61616",
		}),
	}

	binding, err := ArkMQBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "jms:queue:myqueue?brokerURL=tcp%3A%2F%2Fcustom-broker%3A61616", binding.URI)
}

func TestArkMQLookupAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker := &arkmqv1beta1.ActiveMQArtemis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mybroker",
		},
		Status: arkmqv1beta1.ActiveMQArtemisStatus{
			PortStatus: []arkmqv1beta1.ActiveMQArtemisPortStatus{
				{
					Name: "core",
					Port: 61616,
				},
			},
		},
	}

	address := &arkmqv1beta1.ActiveMQArtemisAddress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "myaddress",
		},
		Spec: arkmqv1beta1.ActiveMQArtemisAddressSpec{
			QueueName: "resolved-queue",
			ApplyTo:   "mybroker",
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mybroker-hdls-svc",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "core",
					Port: 61616,
				},
			},
		},
	}

	client, err := internal.NewFakeClient(svc)
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "ActiveMQArtemisAddress",
			Name:       "myaddress",
			APIVersion: "broker.amq.io/v1beta1",
		},
	}

	provider := ArkMQBindingProvider{
		Client: fake.NewSimpleClientset(broker, address),
	}

	binding, err := provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "jms:queue:resolved-queue?brokerURL=tcp%3A%2F%2Fmybroker-hdls-svc.test.svc%3A61616", binding.URI)
}

func TestArkMQMulticastUnsupported(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	address := &arkmqv1beta1.ActiveMQArtemisAddress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mytopic",
		},
		Spec: arkmqv1beta1.ActiveMQArtemisAddressSpec{
			RoutingType: "multicast",
			ApplyTo:     "mybroker",
		},
	}

	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "ActiveMQArtemisAddress",
			Name:       "mytopic",
			APIVersion: "broker.amq.io/v1beta1",
		},
	}

	provider := ArkMQBindingProvider{
		Client: fake.NewSimpleClientset(address),
	}

	_, err = provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multicast addresses (topics) are not supported on queue binding mytopic")
}

func TestArkMQLookupAddressByName(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker := &arkmqv1beta1.ActiveMQArtemis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mybroker",
		},
	}

	address := &arkmqv1beta1.ActiveMQArtemisAddress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "myevents",
			Labels: map[string]string{
				arkmqv1beta1.ArkMQBrokerLabel: "mybroker",
			},
		},
		Spec: arkmqv1beta1.ActiveMQArtemisAddressSpec{
			AddressName: "events-queue",
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mybroker-hdls-svc",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "core",
					Port: 61616,
				},
			},
		},
	}

	client, err := internal.NewFakeClient(svc)
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "ActiveMQArtemisAddress",
			Name:       "myevents",
			APIVersion: "broker.amq.io/v1beta1",
		},
	}

	provider := ArkMQBindingProvider{
		Client: fake.NewSimpleClientset(broker, address),
	}

	binding, err := provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "jms:queue:events-queue?brokerURL=tcp%3A%2F%2Fmybroker-hdls-svc.test.svc%3A61616", binding.URI)
}

func TestArkMQBrokerDirect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker := &arkmqv1beta1.ActiveMQArtemis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mybroker",
		},
		Status: arkmqv1beta1.ActiveMQArtemisStatus{
			PortStatus: []arkmqv1beta1.ActiveMQArtemisPortStatus{
				{
					Name: "core",
					Port: 61616,
				},
			},
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mybroker-hdls-svc",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "core",
					Port: 61616,
				},
			},
		},
	}

	client, err := internal.NewFakeClient(svc)
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	// 1. With "destination" property
	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "ActiveMQArtemis",
			Name:       "mybroker",
			APIVersion: "broker.amq.io/v1beta1",
		},
		Properties: asEndpointProperties(map[string]string{
			"destination": "orders",
		}),
	}

	provider := ArkMQBindingProvider{
		Client: fake.NewSimpleClientset(broker),
	}

	binding, err := provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "jms:queue:orders?brokerURL=tcp%3A%2F%2Fmybroker-hdls-svc.test.svc%3A61616", binding.URI)

	// 2. With "queue" property fallback
	endpointQueue := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "ActiveMQArtemis",
			Name:       "mybroker",
			APIVersion: "broker.amq.io/v1beta1",
		},
		Properties: asEndpointProperties(map[string]string{
			"queue": "invoices",
		}),
	}
	binding, err = provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpointQueue)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "jms:queue:invoices?brokerURL=tcp%3A%2F%2Fmybroker-hdls-svc.test.svc%3A61616", binding.URI)

	// 3. Missing destination/queue property returns error
	endpointMissing := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "ActiveMQArtemis",
			Name:       "mybroker",
			APIVersion: "broker.amq.io/v1beta1",
		},
	}
	_, err = provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpointMissing)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing destination or queue property")
}

func TestArkMQMissingService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker := &arkmqv1beta1.ActiveMQArtemis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "mybroker",
		},
	}

	client, err := internal.NewFakeClient() // no service in client!
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "ActiveMQArtemis",
			Name:       "mybroker",
			APIVersion: "broker.amq.io/v1beta1",
		},
		Properties: asEndpointProperties(map[string]string{
			"destination": "orders",
		}),
	}

	provider := ArkMQBindingProvider{
		Client: fake.NewSimpleClientset(broker),
	}

	_, err = provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not find service mybroker-hdls-svc in namespace test")
}

func TestArkMQUnsupportedKind(t *testing.T) {
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

	for _, kind := range []string{"ActiveMQArtemisQueue", "UnknownKind"} {
		endpoint := camelv1.Endpoint{
			Ref: &corev1.ObjectReference{
				Kind:       kind,
				Name:       "myresource",
				APIVersion: "broker.amq.io/v1beta1",
			},
		}

		_, err := ArkMQBindingProvider{}.Translate(bindingContext, EndpointContext{
			Type: camelv1.EndpointTypeSink,
		}, endpoint)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Can only work with ActiveMQArtemis or ActiveMQArtemisAddress kind")
	}
}

func TestArkMQMissingBrokerLabel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	address := &arkmqv1beta1.ActiveMQArtemisAddress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "unattached-address",
		},
	}

	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "ActiveMQArtemisAddress",
			Name:       "unattached-address",
			APIVersion: "broker.amq.io/v1beta1",
		},
	}

	provider := ArkMQBindingProvider{
		Client: fake.NewSimpleClientset(address),
	}

	_, err = provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no ActiveMQArtemis broker found in namespace test")
}

func TestArkMQBrokerFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker := &arkmqv1beta1.ActiveMQArtemis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "default-broker",
		},
		Status: arkmqv1beta1.ActiveMQArtemisStatus{
			PortStatus: []arkmqv1beta1.ActiveMQArtemisPortStatus{
				{
					Name: "core",
					Port: 61616,
				},
			},
		},
	}

	address := &arkmqv1beta1.ActiveMQArtemisAddress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "unattached-address",
		},
		Spec: arkmqv1beta1.ActiveMQArtemisAddressSpec{
			QueueName: "my-fallback-queue",
			// No applyTo and no labels!
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "default-broker-hdls-svc",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "core",
					Port: 61616,
				},
			},
		},
	}

	client, err := internal.NewFakeClient(svc)
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "ActiveMQArtemisAddress",
			Name:       "unattached-address",
			APIVersion: "broker.amq.io/v1beta1",
		},
	}

	provider := ArkMQBindingProvider{
		Client: fake.NewSimpleClientset(broker, address),
	}

	binding, err := provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "jms:queue:my-fallback-queue?brokerURL=tcp%3A%2F%2Fdefault-broker-hdls-svc.test.svc%3A61616", binding.URI)
}

func TestArkMQMultipleBrokersFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker1 := &arkmqv1beta1.ActiveMQArtemis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "broker-1",
		},
	}
	broker2 := &arkmqv1beta1.ActiveMQArtemis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "broker-2",
		},
	}

	address := &arkmqv1beta1.ActiveMQArtemisAddress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "unattached-address",
		},
	}

	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
		Profile:   camelv1.TraitProfileKubernetes,
	}

	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "ActiveMQArtemisAddress",
			Name:       "unattached-address",
			APIVersion: "broker.amq.io/v1beta1",
		},
	}

	provider := ArkMQBindingProvider{
		Client: fake.NewSimpleClientset(broker1, broker2, address),
	}

	_, err = provider.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple ActiveMQArtemis brokers found in namespace test (2 found)")
}

func TestArkMQPassThrough(t *testing.T) {
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

	// Nil Ref
	binding, err := ArkMQBindingProvider{}.Translate(bindingContext, EndpointContext{}, camelv1.Endpoint{})
	require.NoError(t, err)
	assert.Nil(t, binding)

	// Different API group
	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "KafkaTopic",
			Name:       "mytopic",
			APIVersion: "kafka.strimzi.io/v1beta2",
		},
	}
	binding, err = ArkMQBindingProvider{}.Translate(bindingContext, EndpointContext{}, endpoint)
	require.NoError(t, err)
	assert.Nil(t, binding)
}
