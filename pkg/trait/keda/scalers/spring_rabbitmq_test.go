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
	"github.com/apache/camel-k/v2/pkg/util/source"
)

func TestSpringRabbitMQScalerRegistered(t *testing.T) {
	mapper, found := keda.GetMapper("spring-rabbitmq")
	require.True(t, found)
	assert.Equal(t, "spring-rabbitmq", mapper.Component())
}

func TestSpringRabbitMQScalerMapWithBean(t *testing.T) {
	bean := source.BeanDefinition{
		Name: "rabbitConnectionFactory",
		Type: "org.springframework.amqp.rabbit.connection.CachingConnectionFactory",
		Properties: map[string]string{
			"host":     "my-rabbitmq.my-ns.svc",
			"port":     "5672",
			"username": "admin",
			"password": "admin",
		},
	}
	mapper := &SpringRabbitMQScaler{}
	kedaType, metadata := mapper.MapWithBeans(
		"my-exchange",
		map[string]string{"queues": "my-queue", "routingKey": "my-key"},
		map[string]string{springRabbitMQConnectionFactoryProperty: "#rabbitConnectionFactory"},
		[]source.BeanDefinition{bean},
	)

	assert.Equal(t, "rabbitmq", kedaType)
	assert.Equal(t, map[string]string{
		"queueName": "my-queue",
		"host":      "amqp://my-rabbitmq.my-ns.svc:5672/",
		"protocol":  "amqp",
	}, metadata)
	assert.NotContains(t, metadata, "username")
	assert.NotContains(t, metadata, "password")
}

func TestSpringRabbitMQScalerIgnoresUnsupportedBeanType(t *testing.T) {
	bean := source.BeanDefinition{
		Name: "rabbitConnectionFactory",
		Type: "com.example.CustomConnectionFactory",
		Properties: map[string]string{
			"host": "my-rabbitmq.my-ns.svc",
			"port": "5672",
		},
	}
	mapper := &SpringRabbitMQScaler{}
	kedaType, metadata := mapper.MapWithBeans(
		"my-exchange",
		map[string]string{"queues": "my-queue"},
		map[string]string{springRabbitMQConnectionFactoryProperty: "#rabbitConnectionFactory"},
		[]source.BeanDefinition{bean},
	)

	assert.Equal(t, "rabbitmq", kedaType)
	assert.Equal(t, "my-queue", metadata["queueName"])
	assert.NotContains(t, metadata, "host")
}

func TestSpringRabbitMQScalerDoesNotResolveDynamicBeanProperties(t *testing.T) {
	bean := source.BeanDefinition{
		Name: "rabbitConnectionFactory",
		Type: springRabbitMQConnectionFactoryType,
		Properties: map[string]string{
			"host": "${RABBITMQ_HOST}",
			"port": "5672",
		},
	}
	mapper := &SpringRabbitMQScaler{}
	kedaType, metadata := mapper.MapWithBeans(
		"my-exchange",
		map[string]string{"queues": "my-queue"},
		map[string]string{springRabbitMQConnectionFactoryProperty: "#rabbitConnectionFactory"},
		[]source.BeanDefinition{bean},
	)

	assert.Equal(t, "rabbitmq", kedaType)
	assert.Equal(t, "my-queue", metadata["queueName"])
	assert.NotContains(t, metadata, "host")
}
