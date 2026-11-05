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

	keda "github.com/apache/camel-k/v2/pkg/trait/keda"
)

const (
	// pulsarDefaultSubscription is the default value of the Camel Pulsar subscriptionName option.
	pulsarDefaultSubscription = "subs"
	// pulsarAdminPort is the default port of the Pulsar admin REST API.
	pulsarAdminPort = "8080"
)

// PulsarScaler maps Camel Pulsar consumer endpoints to the KEDA pulsar scaler.
type PulsarScaler struct{}

func init() {
	keda.Register(&PulsarScaler{})
}

func (p *PulsarScaler) Component() string {
	return "pulsar"
}

func (p *PulsarScaler) Map(pathValue string, params map[string]string, props map[string]string) (string, map[string]string) {
	metadata := make(map[string]string)
	// The Camel path is already in the format KEDA expects, e.g. persistent://public/default/my-topic.
	if pathValue != "" {
		metadata["topic"] = pathValue
	}

	subscription := params["subscriptionName"]
	if subscription == "" {
		subscription = props["camel.component.pulsar.subscription-name"]
	}
	if subscription == "" {
		subscription = pulsarDefaultSubscription
	}
	metadata["subscription"] = subscription

	// The admin URL is not part of the Camel configuration. Derive it from a plain pulsar:// service URL
	// on the default admin port; for other setups (TLS, custom ports) it can be set with keda.autoMetadata.
	serviceURL := params["serviceUrl"]
	if serviceURL == "" {
		serviceURL = props["camel.component.pulsar.service-url"]
	}
	if u, err := url.Parse(serviceURL); err == nil && u.Scheme == "pulsar" && u.Hostname() != "" {
		metadata["adminURL"] = "http://" + net.JoinHostPort(u.Hostname(), pulsarAdminPort)
	}

	return "pulsar", metadata
}
