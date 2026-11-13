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
	"strings"

	keda "github.com/apache/camel-k/v2/pkg/trait/keda"
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
// scaler does not use. The broker address is taken from a `host` parameter when the user sets one;
// otherwise it has to be provided with keda.autoMetadata.
func (s *SpringRabbitMQScaler) Map(pathValue string, params map[string]string, props map[string]string) (string, map[string]string) {
	metadata := make(map[string]string)
	// A KEDA rabbitmq trigger watches a single queue, so a list of queues is left for the user to choose.
	if v, ok := params["queues"]; ok && v != "" && !strings.Contains(v, ",") {
		metadata["queueName"] = v
	}
	if v := params["host"]; v != "" {
		metadata["host"] = v
	}

	return "rabbitmq", metadata
}
