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

const (
	ibmmqScalerType       = "ibmmq"
	ibmmqQueueTypeKey     = "queueType"
	ibmmqQueueTypeValue   = "ibm"
	ibmmqAdminHostKey     = "host"
)

// IBMMQScaler maps explicitly selected JMS queue endpoints to the KEDA IBM MQ scaler.
type IBMMQScaler struct{}

func init() {
	keda.Register(&IBMMQScaler{})
}

func (i *IBMMQScaler) Component() string {
	return "jms"
}

// Map intentionally declines discovery without autoMetadata. IBM MQ and other
// JMS providers share the same Camel component, so the provider cannot be
// inferred safely from the endpoint URI.
func (i *IBMMQScaler) Map(pathValue string, params map[string]string, props map[string]string) (string, map[string]string) {
	return "", nil
}

// MapWithAutoMetadata maps JMS queue consumers only when the Integration
// explicitly opts in to IBM MQ and provides KEDA's required admin REST host.
func (i *IBMMQScaler) MapWithAutoMetadata(pathValue string, params map[string]string, props map[string]string, autoMetadata map[string]map[string]string) (string, map[string]string) {
	ibmmqMetadata, ok := autoMetadata[ibmmqScalerType]
	if !ok || !strings.EqualFold(strings.TrimSpace(ibmmqMetadata[ibmmqQueueTypeKey]), ibmmqQueueTypeValue) {
		return "", nil
	}
	if strings.TrimSpace(ibmmqMetadata[ibmmqAdminHostKey]) == "" {
		// The KEDA IBM MQ scaler requires its Queue Manager Admin REST endpoint.
		// It cannot be derived reliably from a generic JMS endpoint URI.
		return "", nil
	}

	queueName, ok := jmsQueueName(pathValue, params)
	if !ok {
		return "", nil
	}

	return ibmmqScalerType, map[string]string{
		"queueName": queueName,
	}
}

// jmsQueueName returns the destination name only for queue endpoints. Camel
// JMS treats an unqualified destination as a queue; topic and temporary
// destinations are not supported by the KEDA IBM MQ queue scaler.
func jmsQueueName(pathValue string, params map[string]string) (string, bool) {
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		return "", false
	}

	destinationType := "queue"
	destinationName := pathValue

	if prefix, name, found := strings.Cut(pathValue, ":"); found {
		switch strings.ToLower(prefix) {
		case "queue":
			destinationName = name
		case "topic", "temp-queue", "temp-topic":
			return "", false
		default:
			// Do not guess when the destination syntax is not recognized.
			return "", false
		}
	}

	if configuredType := strings.TrimSpace(params["destinationType"]); configuredType != "" {
		destinationType = strings.ToLower(configuredType)
	}
	destinationName = strings.TrimSpace(destinationName)
	if destinationType != "queue" || destinationName == "" {
		return "", false
	}

	return destinationName, true
}
