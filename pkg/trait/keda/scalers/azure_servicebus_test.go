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
		expectedType string
		expectedMeta map[string]string
	}{
		{
			name:         "queue by default",
			pathValue:    "orders",
			params:       map[string]string{},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{
				"queueName": "orders",
			},
		},
		{
			name:         "queue explicit",
			pathValue:    "orders",
			params:       map[string]string{"serviceBusType": "queue"},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{
				"queueName": "orders",
			},
		},
		{
			name:         "topic with subscription",
			pathValue:    "events",
			params:       map[string]string{"serviceBusType": "topic", "subscriptionName": "sub1"},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{
				"topicName":        "events",
				"subscriptionName": "sub1",
			},
		},
		{
			name:         "topic without subscription",
			pathValue:    "events",
			params:       map[string]string{"serviceBusType": "topic"},
			expectedType: "azure-servicebus",
			expectedMeta: map[string]string{
				"topicName": "events",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kedaType, metadata := scaler.Map(tt.pathValue, tt.params)
			assert.Equal(t, tt.expectedType, kedaType)
			assert.Equal(t, tt.expectedMeta, metadata)
		})
	}
}

func TestMapToKedaTriggerAzureServiceBus(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger("azure-servicebus:orders")
	require.NoError(t, err)
	require.NotNil(t, trigger)
	assert.Equal(t, "azure-servicebus", trigger.Type)
	assert.Equal(t, map[string]string{
		"queueName": "orders",
	}, trigger.Metadata)

	trigger, err = keda.MapToKedaTrigger("azure-servicebus:events?serviceBusType=topic&subscriptionName=sub1")
	require.NoError(t, err)
	require.NotNil(t, trigger)
	assert.Equal(t, "azure-servicebus", trigger.Type)
	assert.Equal(t, map[string]string{
		"topicName":        "events",
		"subscriptionName": "sub1",
	}, trigger.Metadata)
}
