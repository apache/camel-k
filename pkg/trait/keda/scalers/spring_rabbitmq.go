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
	"strings"

	keda "github.com/apache/camel-k/v2/pkg/trait/keda"
)

const (
	springRabbitMQConnectionFactoryProp = "camel.component.spring-rabbitmq.connection-factory"
	camelBeansPrefix                    = "camel.beans."
	rabbitMQDefaultPort                 = "5672"
)

// SpringRabbitMQScaler maps Camel Spring RabbitMQ consumer endpoints to the KEDA rabbitmq scaler.
type SpringRabbitMQScaler struct{}

func init() {
	keda.Register(&SpringRabbitMQScaler{})
}

func (s *SpringRabbitMQScaler) Component() string {
	return "spring-rabbitmq"
}

// Map takes the queue from the `queues` parameter. The URI path is the exchange name, which the
// scaler does not use. A `host` parameter is used as the broker address as is; otherwise the address is
// read from the connection factory bean when it is declared with camel.beans properties.
func (s *SpringRabbitMQScaler) Map(pathValue string, params map[string]string, props map[string]string) (string, map[string]string) {
	metadata := make(map[string]string)
	// A KEDA rabbitmq trigger watches a single queue, so a list of queues is left for the user to choose.
	if v, ok := params["queues"]; ok && v != "" && !strings.Contains(v, ",") {
		metadata["queueName"] = v
	}
	if v := params["host"]; v != "" {
		metadata["host"] = v
	} else if host := springRabbitMQHost(params, props); host != "" {
		metadata["host"] = host
	}

	return "rabbitmq", metadata
}

// springRabbitMQHost builds an amqp:// URL (without credentials) from the connection factory bean properties.
func springRabbitMQHost(params map[string]string, props map[string]string) string {
	beanName := connectionFactoryBeanName(params, props)
	if beanName == "" {
		return ""
	}
	beanProps := beanProperties(beanName, props)

	host, port, vhost := brokerFromURI(beanProps["uri"])
	if host == "" {
		host, port, vhost = brokerFromHostProperties(beanProps)
	}
	if host == "" {
		return ""
	}
	if port == "" {
		port = rabbitMQDefaultPort
	}

	// A vhost may contain "/" itself, so it is escaped as a single path segment.
	u := url.URL{Scheme: "amqp", Host: net.JoinHostPort(host, port), Path: "/" + vhost, RawPath: "/" + url.PathEscape(vhost)}

	return u.String()
}

// brokerFromURI returns the broker host, port and vhost of an amqp:// URI.
func brokerFromURI(uri string) (string, string, string) {
	u, err := url.Parse(uri)
	if uri == "" || err != nil {
		return "", "", ""
	}

	return u.Hostname(), u.Port(), strings.TrimPrefix(u.Path, "/")
}

// brokerFromHostProperties returns the broker host, port and vhost of the host, port, addresses and
// virtualHost bean properties.
func brokerFromHostProperties(beanProps map[string]string) (string, string, string) {
	host, port := beanProps["host"], beanProps["port"]
	// The connection factory tries addresses in order, so the first one is the primary broker.
	if addresses := beanProps["addresses"]; addresses != "" {
		first := strings.TrimSpace(strings.Split(addresses, ",")[0])
		if h, p, err := net.SplitHostPort(first); err == nil {
			host, port = h, p
		} else {
			host, port = first, ""
		}
	}

	return host, port, strings.TrimPrefix(beanProps["virtualhost"], "/")
}

// connectionFactoryBeanName returns the bean referenced by the endpoint or component connectionFactory option.
func connectionFactoryBeanName(params map[string]string, props map[string]string) string {
	ref := params["connectionFactory"]
	if ref == "" {
		ref = props[springRabbitMQConnectionFactoryProp]
	}
	ref = strings.TrimPrefix(ref, "#")
	ref = strings.TrimPrefix(ref, "bean:")

	return ref
}

// beanProperties collects the camel.beans.<name>.* properties, with keys lower cased and dashes removed
// so that both `virtualHost` and `virtual-host` match.
func beanProperties(beanName string, props map[string]string) map[string]string {
	prefix := camelBeansPrefix + beanName + "."
	beanProps := make(map[string]string)
	for k, v := range props {
		if name, ok := strings.CutPrefix(k, prefix); ok {
			beanProps[strings.ToLower(strings.ReplaceAll(name, "-", ""))] = strings.TrimSpace(v)
		}
	}

	return beanProps
}
