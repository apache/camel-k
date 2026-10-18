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

// ArtemisScaler maps Camel AMQP queue endpoints to the KEDA artemis-queue scaler.
type ArtemisScaler struct{}

func init() {
	keda.Register(&ArtemisScaler{})
}

func (a *ArtemisScaler) Component() string {
	return "amqp"
}

func (a *ArtemisScaler) Map(pathValue string, params map[string]string) (string, map[string]string) {
	metadata := make(map[string]string)
	queueName := strings.TrimPrefix(pathValue, "queue:")
	if queueName != "" && !strings.HasPrefix(queueName, "topic:") {
		metadata["queueName"] = queueName
		// For anycast queues created by ArkMQ, the address name matches the queue name.
		metadata["brokerAddress"] = queueName
	}

	return "artemis-queue", metadata
}
