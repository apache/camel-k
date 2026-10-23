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

type ActiveMQScaler struct{}

func init() {
	keda.Register(&ActiveMQScaler{})
}

func (a *ActiveMQScaler) Component() string {
	return "activemq"
}

func (a *ActiveMQScaler) Map(pathValue string, params map[string]string) (string, map[string]string) {
	metadata := make(map[string]string)

	destination := pathValue
	if strings.HasPrefix(destination, "queue:") {
		destination = strings.TrimPrefix(destination, "queue:")
	} else if strings.HasPrefix(destination, "topic:") {
		destination = strings.TrimPrefix(destination, "topic:")
	}

	if destination != "" {
		metadata["destinationName"] = destination
	}
	if v, ok := params["destinationName"]; ok {
		metadata["destinationName"] = v
	}
	if v, ok := params["managementEndpoint"]; ok {
		metadata["managementEndpoint"] = v
	}
	if v, ok := params["brokerName"]; ok {
		metadata["brokerName"] = v
	}
	if v, ok := params["targetQueueSize"]; ok {
		metadata["targetQueueSize"] = v
	}
	if v, ok := params["activationTargetQueueSize"]; ok {
		metadata["activationTargetQueueSize"] = v
	}
	if v, ok := params["corsHeader"]; ok {
		metadata["corsHeader"] = v
	}
	if v, ok := params["restAPITemplate"]; ok {
		metadata["restAPITemplate"] = v
	}

	return "activemq", metadata
}
