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
		props        map[string]string
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
		{
			name:      "host and port from connection factory bean",
			pathValue: "orders-exchange",
			params:    map[string]string{"queues": "orders"},
			props: map[string]string{
				"camel.component.spring-rabbitmq.connection-factory": "#rabbitConnectionFactory",
				"camel.beans.rabbitConnectionFactory":                "#class:org.springframework.amqp.rabbit.connection.CachingConnectionFactory",
				"camel.beans.rabbitConnectionFactory.host":           "my-rabbitmq.ns.svc",
				"camel.beans.rabbitConnectionFactory.port":           "5673",
				"camel.beans.rabbitConnectionFactory.username":       "admin",
				"camel.beans.rabbitConnectionFactory.password":       "admin",
			},
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{
				"queueName": "orders",
				"host":      "amqp://my-rabbitmq.ns.svc:5673/",
			},
		},
		{
			name:      "host parameter wins over connection factory bean",
			pathValue: "orders-exchange",
			params:    map[string]string{"queues": "orders", "host": "amqp://user:pass@other-rabbitmq:5672/vh"},
			props: map[string]string{
				"camel.component.spring-rabbitmq.connection-factory": "#rabbitConnectionFactory",
				"camel.beans.rabbitConnectionFactory.host":           "my-rabbitmq.ns.svc",
			},
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{
				"queueName": "orders",
				"host":      "amqp://user:pass@other-rabbitmq:5672/vh",
			},
		},
		{
			name:      "default port and virtual host",
			pathValue: "orders-exchange",
			params:    map[string]string{"queues": "orders"},
			props: map[string]string{
				"camel.component.spring-rabbitmq.connection-factory": "#bean:cf",
				"camel.beans.cf.host":                                "my-rabbitmq",
				"camel.beans.cf.virtual-host":                        "/sales",
			},
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{
				"queueName": "orders",
				"host":      "amqp://my-rabbitmq:5672/sales",
			},
		},
		{
			name:      "first of addresses",
			pathValue: "orders-exchange",
			params:    map[string]string{"queues": "orders"},
			props: map[string]string{
				"camel.component.spring-rabbitmq.connection-factory": "#cf",
				"camel.beans.cf.addresses":                           "rabbit-0:5674, rabbit-1:5674",
				"camel.beans.cf.virtualHost":                         "sales",
			},
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{
				"queueName": "orders",
				"host":      "amqp://rabbit-0:5674/sales",
			},
		},
		{
			name:      "uri without credentials",
			pathValue: "orders-exchange",
			params:    map[string]string{"queues": "orders"},
			props: map[string]string{
				"camel.component.spring-rabbitmq.connection-factory": "#cf",
				"camel.beans.cf.uri": "amqp://admin:secret@my-rabbitmq:5672/sales",
			},
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{
				"queueName": "orders",
				"host":      "amqp://my-rabbitmq:5672/sales",
			},
		},
		{
			name:      "endpoint connection factory wins over component",
			pathValue: "orders-exchange",
			params:    map[string]string{"queues": "orders", "connectionFactory": "#endpointCf"},
			props: map[string]string{
				"camel.component.spring-rabbitmq.connection-factory": "#componentCf",
				"camel.beans.componentCf.host":                       "component-rabbit",
				"camel.beans.endpointCf.host":                        "endpoint-rabbit",
			},
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{
				"queueName": "orders",
				"host":      "amqp://endpoint-rabbit:5672/",
			},
		},
		{
			name:      "bean without host is not mapped",
			pathValue: "orders-exchange",
			params:    map[string]string{"queues": "orders"},
			props: map[string]string{
				"camel.component.spring-rabbitmq.connection-factory": "#cf",
				"camel.beans.cf.username":                            "admin",
			},
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{
				"queueName": "orders",
			},
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

func TestMapToKedaTriggerSpringRabbitMQ(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger("spring-rabbitmq:orders-exchange?queues=orders&routingKey=new", nil)
	require.NoError(t, err)
	require.NotNil(t, trigger)
	assert.Equal(t, "rabbitmq", trigger.Type)
	assert.Equal(t, map[string]string{
		"queueName": "orders",
	}, trigger.Metadata)
}

func TestMapToKedaTriggerSpringRabbitMQWithConnectionFactory(t *testing.T) {
	props := map[string]string{
		"camel.component.spring-rabbitmq.connection-factory": "#rabbitConnectionFactory",
		"camel.beans.rabbitConnectionFactory.host":           "my-rabbitmq.ns.svc",
		"camel.component.kafka.brokers":                      "broker:9092",
	}
	trigger, err := keda.MapToKedaTrigger("spring-rabbitmq:orders-exchange?queues=orders", props)
	require.NoError(t, err)
	require.NotNil(t, trigger)
	assert.Equal(t, map[string]string{
		"queueName": "orders",
		"host":      "amqp://my-rabbitmq.ns.svc:5672/",
	}, trigger.Metadata)
}
