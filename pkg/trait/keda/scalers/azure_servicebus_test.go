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

package scalers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	keda "github.com/apache/camel-k/v2/pkg/trait/keda"
)

func TestAzureServiceBusScalerRegistered(t *testing.T) {
	mapper, found := keda.GetMapper("azure-servicebus")
	require.True(t, found, "azure-servicebus scaler should be registered")
	assert.Equal(t, "azure-servicebus", mapper.Component())
}

func TestAzureServiceBusScalerMap(t *testing.T) {
	scaler := &AzureServiceBusScaler{}

	tests := []struct {
		name         string
		pathValue    string
		params       map[string]string
		props        map[string]string
		expectedType string
		expectedMeta map[string]string
	}{
		{
			name:         "queue by default",
			pathValue:    "orders",
			params:       map[string]string{},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{"queueName": "orders"},
		},
		{
			name:         "explicit queue type",
			pathValue:    "orders",
			params:       map[string]string{"serviceBusType": "queue"},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{"queueName": "orders"},
		},
		{
			name:         "topic with subscription",
			pathValue:    "events",
			params:       map[string]string{"serviceBusType": "topic", "subscriptionName": "audit"},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{"topicName": "events", "subscriptionName": "audit"},
		},
		{
			name:         "topic type is case insensitive",
			pathValue:    "events",
			params:       map[string]string{"serviceBusType": "Topic", "subscriptionName": "audit"},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{"topicName": "events", "subscriptionName": "audit"},
		},
		{
			name:         "topic without subscription is not discoverable",
			pathValue:    "events",
			params:       map[string]string{"serviceBusType": "topic"},
			expectedType: "",
			expectedMeta: nil,
		},
		{
			name:         "empty endpoint path is not discoverable",
			pathValue:    "",
			params:       map[string]string{},
			expectedType: "",
			expectedMeta: nil,
		},
		{
			name:         "public cloud namespace",
			pathValue:    "orders",
			params:       map[string]string{"fullyQualifiedNamespace": "My-Namespace.servicebus.windows.net"},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{
				"queueName": "orders",
				"namespace": "my-namespace",
			},
		},
		{
			name:         "other cloud namespace is left to autoMetadata",
			pathValue:    "orders",
			params:       map[string]string{"fullyQualifiedNamespace": "my-namespace.servicebus.usgovcloudapi.net"},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{"queueName": "orders"},
		},
		{
			name:         "connection string is not copied",
			pathValue:    "orders",
			params:       map[string]string{"connectionString": "Endpoint=sb://ns.servicebus.windows.net/;SharedAccessKey=secret"},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{"queueName": "orders"},
		},
		{
			name:      "component connection string is not copied",
			pathValue: "orders",
			params:    map[string]string{},
			props: map[string]string{
				"camel.component.azure-servicebus.connection-string": "Endpoint=sb://ns.servicebus.windows.net/;SharedAccessKey=secret",
			},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{"queueName": "orders"},
		},
		{
			name:      "component properties",
			pathValue: "events",
			params:    map[string]string{},
			props: map[string]string{
				"camel.component.azure-servicebus.service-bus-type":          "topic",
				"camel.component.azure-servicebus.subscription-name":         "audit",
				"camel.component.azure-servicebus.fully-qualified-namespace": "my-namespace.servicebus.windows.net",
			},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{
				"topicName":        "events",
				"subscriptionName": "audit",
				"namespace":        "my-namespace",
			},
		},
		{
			name:      "endpoint parameters win over component properties",
			pathValue: "orders",
			params:    map[string]string{"serviceBusType": "queue"},
			props: map[string]string{
				"camel.component.azure-servicebus.service-bus-type":          "topic",
				"camel.component.azure-servicebus.subscription-name":         "audit",
			},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{"queueName": "orders"},
		},
		{
			name:      "component topic without subscription is not discoverable",
			pathValue: "events",
			params:    map[string]string{},
			props: map[string]string{
				"camel.component.azure-servicebus.service-bus-type": "topic",
			},
			expectedType: "",
			expectedMeta: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kedaType, metadata := scaler.Map(tt.pathValue, tt.params, tt.props)
			assert.Equal(t, tt.expectedType, kedaType)
			assert.Equal(t, tt.expectedMeta, metadata)
		})
	}
}

func TestMapToKedaTriggerAzureServiceBus(t *testing.T) {
	tests := []struct {
		name         string
		uri          string
		props        map[string]string
		expectedMeta map[string]string
	}{
		{
			name:         "queue",
			uri:          "azure-servicebus:orders?serviceBusType=queue",
			expectedMeta: map[string]string{"queueName": "orders"},
		},
		{
			name: "topic from component properties",
			uri:  "azure-servicebus:events",
			props: map[string]string{
				"camel.component.azure-servicebus.service-bus-type":          "topic",
				"camel.component.azure-servicebus.subscription-name":         "audit",
				"camel.component.azure-servicebus.fully-qualified-namespace": "my-namespace.servicebus.windows.net",
			},
			expectedMeta: map[string]string{
				"topicName":        "events",
				"subscriptionName": "audit",
				"namespace":        "my-namespace",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, err := keda.MapToKedaTrigger(tt.uri, tt.props)
			require.NoError(t, err)
			require.NotNil(t, trigger)
			assert.Equal(t, "azure-servicebus", trigger.Type)
			assert.Equal(t, tt.expectedMeta, trigger.Metadata)
		})
	}
}

func TestMapToKedaTriggerAzureServiceBusSkipsIncompleteTopic(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger("azure-servicebus:events?serviceBusType=topic", nil)
	require.NoError(t, err)
	assert.Nil(t, trigger)
}
