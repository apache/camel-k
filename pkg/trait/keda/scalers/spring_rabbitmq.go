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
	"net"
	"net/url"
	"strconv"
	"strings"

	keda "github.com/apache/camel-k/v2/pkg/trait/keda"
	"github.com/apache/camel-k/v2/pkg/util/source"
)

const springRabbitMQConnectionFactoryProperty = "camel.component.spring-rabbitmq.connection-factory"
const springRabbitMQConnectionFactoryType = "org.springframework.amqp.rabbit.connection.CachingConnectionFactory"

// SpringRabbitMQScaler maps Camel Spring RabbitMQ consumers to the KEDA RabbitMQ scaler.
type SpringRabbitMQScaler struct{}

func init() {
	keda.Register(&SpringRabbitMQScaler{})
}

func (s *SpringRabbitMQScaler) Component() string {
	return "spring-rabbitmq"
}

func (s *SpringRabbitMQScaler) Map(_ string, params map[string]string, _ map[string]string) (string, map[string]string) {
	queueName := strings.TrimSpace(params["queues"])
	if queueName == "" || strings.ContainsAny(queueName, ",;{}$") {
		return "", nil
	}

	return "rabbitmq", map[string]string{
		"queueName": queueName,
	}
}

// MapWithBeans uses a directly declared connection-factory bean only when the
// component property contains a simple reference to exactly one YAML bean definition.
// Credentials are intentionally not mapped to KEDA metadata.
func (s *SpringRabbitMQScaler) MapWithBeans(pathValue string, params map[string]string, props map[string]string, beans []source.BeanDefinition) (string, map[string]string) {
	kedaType, metadata := s.Map(pathValue, params, props)
	if kedaType == "" {
		return "", metadata
	}

	beanReference := props[springRabbitMQConnectionFactoryProperty]
	bean, found := keda.ResolveBeanReference(beanReference, beans)
	if !found || bean.Type != springRabbitMQConnectionFactoryType {
		return kedaType, metadata
	}

	host := strings.TrimSpace(bean.Properties["host"])
	port := strings.TrimSpace(bean.Properties["port"])
	if !isStaticHost(host) || !isValidPort(port) {
		return kedaType, metadata
	}

	metadata["host"] = (&url.URL{
		Scheme: "amqp",
		Host:   net.JoinHostPort(strings.Trim(host, "[]"), port),
		Path:   "/",
	}).String()
	metadata["protocol"] = "amqp"

	virtualHost := strings.TrimSpace(bean.Properties["virtualHost"])
	if isStaticVirtualHost(virtualHost) {
		metadata["vhostName"] = virtualHost
	}

	return kedaType, metadata
}

func isStaticHost(host string) bool {
	if host == "" || strings.ContainsAny(host, "/\\?#@ \t\r\n{}$") {
		return false
	}
	if strings.Contains(host, ":") && net.ParseIP(strings.Trim(host, "[]")) == nil {
		return false
	}

	return true
}

func isStaticVirtualHost(virtualHost string) bool {
	return virtualHost != "" &&
		!strings.Contains(virtualHost, "${") &&
		!strings.Contains(virtualHost, "#{") &&
		!strings.Contains(virtualHost, "{{")
}

func isValidPort(port string) bool {
	value, err := strconv.Atoi(port)

	return err == nil && value > 0 && value <= 65535
}
