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

// azureServiceBusPublicSuffix is the host suffix of Service Bus namespaces in the Azure public cloud.
const azureServiceBusPublicSuffix = ".servicebus.windows.net"

// AzureServiceBusScaler maps Camel Azure Service Bus consumer endpoints to the KEDA azure-servicebus scaler.
type AzureServiceBusScaler struct{}

func init() {
	keda.Register(&AzureServiceBusScaler{})
}

func (a *AzureServiceBusScaler) Component() string {
	return "azure-servicebus"
}

// Map takes the queue or topic name from the URI path. The Camel serviceBusType option (queue by
// default) decides whether it becomes the KEDA queueName or topicName; a topic also requires a
// subscriptionName. Component-level properties are used when an endpoint does not override them.
// Connection strings are deliberately not copied into trigger metadata.
func (a *AzureServiceBusScaler) Map(pathValue string, params map[string]string, props map[string]string) (string, map[string]string) {
	if pathValue == "" {
		return "", nil
	}

	metadata := make(map[string]string)
	serviceBusType := azureServiceBusOption(params, props, "serviceBusType", "service-bus-type")
	if strings.EqualFold(serviceBusType, "topic") {
		subscriptionName := azureServiceBusOption(params, props, "subscriptionName", "subscription-name")
		if subscriptionName == "" {
			// KEDA requires both topicName and subscriptionName for this scaler.
			return "", nil
		}

		metadata["topicName"] = pathValue
		metadata["subscriptionName"] = subscriptionName
	} else {
		metadata["queueName"] = pathValue
	}

	// KEDA expects the namespace name, e.g. "my-namespace" for my-namespace.servicebus.windows.net.
	// Other Azure clouds also need cloud/endpointSuffix, so they are left to keda.autoMetadata.
	fqns := strings.ToLower(azureServiceBusOption(params, props, "fullyQualifiedNamespace", "fully-qualified-namespace"))
	if name, ok := strings.CutSuffix(fqns, azureServiceBusPublicSuffix); ok && name != "" && !strings.Contains(name, ".") {
		metadata["namespace"] = name
	}

	return "azure-servicebus", metadata
}

// azureServiceBusOption returns an endpoint parameter, falling back to the component-level
// property (camel.component.azure-servicebus.<property>) when the endpoint does not set it.
func azureServiceBusOption(params map[string]string, props map[string]string, param string, property string) string {
	if value := params[param]; value != "" {
		return value
	}

	return props["camel.component.azure-servicebus."+property]
}
