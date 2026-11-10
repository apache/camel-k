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

func TestPulsarScalerRegistered(t *testing.T) {
	mapper, found := keda.GetMapper("pulsar")
	require.True(t, found, "pulsar scaler should be registered")
	assert.Equal(t, "pulsar", mapper.Component())
}

func TestPulsarScalerMap(t *testing.T) {
	scaler := &PulsarScaler{}

	tests := []struct {
		name         string
		pathValue    string
		params       map[string]string
		props        map[string]string
		expectedMeta map[string]string
	}{
		{
			name:      "subscription and service URL from the endpoint",
			pathValue: "persistent://public/default/orders",
			params:    map[string]string{"subscriptionName": "orders-sub", "serviceUrl": "pulsar://pulsar.ns.svc:6650"},
			expectedMeta: map[string]string{
				"topic":        "persistent://public/default/orders",
				"subscription": "orders-sub",
				"adminURL":     "http://pulsar.ns.svc:8080",
			},
		},
		{
			name:      "subscription and service URL from component properties",
			pathValue: "persistent://public/default/orders",
			params:    map[string]string{},
			props: map[string]string{
				"camel.component.pulsar.subscription-name": "props-sub",
				"camel.component.pulsar.service-url":       "pulsar://broker:6650",
			},
			expectedMeta: map[string]string{
				"topic":        "persistent://public/default/orders",
				"subscription": "props-sub",
				"adminURL":     "http://broker:8080",
			},
		},
		{
			name:      "endpoint values take precedence over properties",
			pathValue: "persistent://public/default/orders",
			params:    map[string]string{"subscriptionName": "uri-sub", "serviceUrl": "pulsar://uri-host:6650"},
			props: map[string]string{
				"camel.component.pulsar.subscription-name": "props-sub",
				"camel.component.pulsar.service-url":       "pulsar://props-host:6650",
			},
			expectedMeta: map[string]string{
				"topic":        "persistent://public/default/orders",
				"subscription": "uri-sub",
				"adminURL":     "http://uri-host:8080",
			},
		},
		{
			name:      "Camel default subscription and no service URL",
			pathValue: "persistent://public/default/orders",
			params:    map[string]string{},
			expectedMeta: map[string]string{
				"topic":        "persistent://public/default/orders",
				"subscription": "subs",
			},
		},
		{
			name:      "TLS service URL leaves adminURL to autoMetadata",
			pathValue: "persistent://public/default/orders",
			params:    map[string]string{"serviceUrl": "pulsar+ssl://broker:6651"},
			expectedMeta: map[string]string{
				"topic":        "persistent://public/default/orders",
				"subscription": "subs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kedaType, metadata := scaler.Map(tt.pathValue, tt.params, tt.props)
			assert.Equal(t, "pulsar", kedaType)
			assert.Equal(t, tt.expectedMeta, metadata)
		})
	}
}

func TestMapToKedaTriggerPulsar(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger(
		"pulsar:persistent://public/default/orders?subscriptionName=orders-sub&serviceUrl=pulsar://broker:6650", nil)
	require.NoError(t, err)
	require.NotNil(t, trigger)
	assert.Equal(t, "pulsar", trigger.Type)
	assert.Equal(t, map[string]string{
		"topic":        "persistent://public/default/orders",
		"subscription": "orders-sub",
		"adminURL":     "http://broker:8080",
	}, trigger.Metadata)
}
