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
)

func TestNatsScalerRegistered(t *testing.T) {
	mapper, found := keda.GetMapper("nats")
	require.True(t, found, "nats scaler should be registered")
	assert.Equal(t, "nats", mapper.Component())
}

func TestNatsScalerMap(t *testing.T) {
	scaler := &NatsScaler{}

	tests := []struct {
		name         string
		params       map[string]string
		props        map[string]string
		expectedType string
		expectedMeta map[string]string
	}{
		{
			name: "durable JetStream consumer from the endpoint",
			params: map[string]string{
				"jetstreamEnabled": "true", "jetstreamName": "ORDERS", "durableName": "keda",
				"servers": "nats://nats.ns.svc:4222",
			},
			expectedType: "nats-jetstream",
			expectedMeta: map[string]string{
				"account":                      "$G",
				"stream":                       "ORDERS",
				"consumer":                     "keda-durable",
				"natsServerMonitoringEndpoint": "nats.ns.svc:8222",
			},
		},
		{
			name:   "options from component properties",
			params: map[string]string{},
			props: map[string]string{
				"camel.component.nats.jetstream-enabled": "true",
				"camel.component.nats.jetstream-name":    "ORDERS",
				"camel.component.nats.durable-name":      "keda",
				"camel.component.nats.servers":           "broker:4222,other:4222",
			},
			expectedType: "nats-jetstream",
			expectedMeta: map[string]string{
				"account":                      "$G",
				"stream":                       "ORDERS",
				"consumer":                     "keda-durable",
				"natsServerMonitoringEndpoint": "broker:8222",
			},
		},
		{
			name:   "endpoint options take precedence over properties",
			params: map[string]string{"jetstreamEnabled": "true", "jetstreamName": "URI", "durableName": "uri", "servers": "uri-host:4222"},
			props: map[string]string{
				"camel.component.nats.jetstream-name": "PROPS",
				"camel.component.nats.durable-name":   "props",
				"camel.component.nats.servers":        "props-host:4222",
			},
			expectedType: "nats-jetstream",
			expectedMeta: map[string]string{
				"account":                      "$G",
				"stream":                       "URI",
				"consumer":                     "uri-durable",
				"natsServerMonitoringEndpoint": "uri-host:8222",
			},
		},
		{
			name:         "no servers leaves the monitoring endpoint to autoMetadata",
			params:       map[string]string{"jetstreamEnabled": "true", "jetstreamName": "ORDERS", "durableName": "keda"},
			expectedType: "nats-jetstream",
			expectedMeta: map[string]string{"account": "$G", "stream": "ORDERS", "consumer": "keda-durable"},
		},
		{
			name:   "core NATS subject",
			params: map[string]string{"servers": "nats://broker:4222"},
		},
		{
			name:   "ephemeral JetStream consumer",
			params: map[string]string{"jetstreamEnabled": "true", "jetstreamName": "ORDERS"},
		},
		{
			name:   "JetStream without a stream name",
			params: map[string]string{"jetstreamEnabled": "true", "durableName": "keda"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kedaType, metadata := scaler.Map("orders", tt.params, tt.props)
			assert.Equal(t, tt.expectedType, kedaType)
			assert.Equal(t, tt.expectedMeta, metadata)
		})
	}
}

func TestMapToKedaTriggerNats(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger(
		"nats:orders?jetstreamEnabled=true&jetstreamName=ORDERS&durableName=keda&servers=nats://broker:4222", nil)
	require.NoError(t, err)
	require.NotNil(t, trigger)
	assert.Equal(t, "nats-jetstream", trigger.Type)
	assert.Equal(t, map[string]string{
		"account":                      "$G",
		"stream":                       "ORDERS",
		"consumer":                     "keda-durable",
		"natsServerMonitoringEndpoint": "broker:8222",
	}, trigger.Metadata)
}

func TestMapToKedaTriggerNatsWithoutDurableConsumer(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger("nats:orders?servers=nats://broker:4222", nil)
	require.NoError(t, err)
	assert.Nil(t, trigger)
}
