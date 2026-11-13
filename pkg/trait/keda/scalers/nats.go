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
	// natsDefaultAccount is the account that holds the streams of a NATS server without configured accounts.
	natsDefaultAccount = "$G"
	// natsMonitoringPort is the default port of the NATS server monitoring endpoint.
	natsMonitoringPort = "8222"
	// natsDurableSuffix is appended by Camel to the durableName option to name the JetStream consumer.
	natsDurableSuffix = "-durable"
)

// NatsScaler maps Camel NATS JetStream consumer endpoints to the KEDA nats-jetstream scaler.
type NatsScaler struct{}

func init() {
	keda.Register(&NatsScaler{})
}

func (n *NatsScaler) Component() string {
	return "nats"
}

func (n *NatsScaler) Map(pathValue string, params map[string]string, props map[string]string) (string, map[string]string) {
	option := func(name, property string) string {
		if v := params[name]; v != "" {
			return v
		}

		return props["camel.component.nats."+property]
	}

	// KEDA scales on the pending messages of a durable JetStream consumer. A core NATS subject keeps no backlog, and an
	// ephemeral consumer is removed while the Integration is scaled to zero, so neither produces a trigger.
	stream := option("jetstreamName", "jetstream-name")
	durable := option("durableName", "durable-name")
	if !strings.EqualFold(option("jetstreamEnabled", "jetstream-enabled"), "true") || stream == "" || durable == "" {
		return "", nil
	}

	metadata := map[string]string{
		"account":  natsDefaultAccount,
		"stream":   stream,
		"consumer": durable + natsDurableSuffix,
	}
	// The monitoring endpoint is not part of the Camel configuration. Derive it from the first server on the default
	// monitoring port; for other setups it can be set with keda.autoMetadata.
	if host := natsServerHost(option("servers", "servers")); host != "" {
		metadata["natsServerMonitoringEndpoint"] = net.JoinHostPort(host, natsMonitoringPort)
	}

	return "nats-jetstream", metadata
}

// natsServerHost returns the host of the first server of a Camel servers option, for example
// "nats://nats.ns.svc:4222" or "nats.ns.svc:4222,other.ns.svc:4222".
func natsServerHost(servers string) string {
	first := strings.TrimSpace(strings.Split(servers, ",")[0])
	if first == "" {
		return ""
	}
	if !strings.Contains(first, "://") {
		first = "nats://" + first
	}
	u, err := url.Parse(first)
	if err != nil {
		return ""
	}

	return u.Hostname()
}
