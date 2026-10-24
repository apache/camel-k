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

func TestSpringRabbitMQScalerRegistered(t *testing.T) {
	mapper, found := keda.GetMapper("spring-rabbitmq")
	require.True(t, found, "spring-rabbitmq scaler should be registered")
	assert.Equal(t, "spring-rabbitmq", mapper.Component())
}

func TestSpringRabbitMQScalerMap(t *testing.T) {
	scaler := &SpringRabbitMQScaler{}

	tests := []struct {
		name         string
		pathValue    string
		params       map[string]string
		expectedType string
		expectedMeta map[string]string
	}{
		{
			name:         "single queue",
			pathValue:    "orders-exchange",
			params:       map[string]string{"queues": "orders", "routingKey": "new", "autoDeclare": "true"},
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{
				"queueName": "orders",
			},
		},
		{
			name:         "no queue",
			pathValue:    "orders-exchange",
			params:       map[string]string{},
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{},
		},
		{
			name:         "empty queue",
			pathValue:    "default",
			params:       map[string]string{"queues": ""},
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{},
		},
		{
			name:         "several queues",
			pathValue:    "orders-exchange",
			params:       map[string]string{"queues": "orders,invoices"},
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{},
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

func TestMapToKedaTriggerSpringRabbitMQ(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger("spring-rabbitmq:orders-exchange?queues=orders&routingKey=new")
	require.NoError(t, err)
	require.NotNil(t, trigger)
	assert.Equal(t, "rabbitmq", trigger.Type)
	assert.Equal(t, map[string]string{
		"queueName": "orders",
	}, trigger.Metadata)
}
